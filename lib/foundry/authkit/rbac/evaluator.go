package rbac

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Errors that can be mapped by middleware.
var (
	ErrConditionStepUpRequired = errors.New("step_up_required")
)

// simpleEvaluator is a placeholder evaluator that wires condition registry and precedence.
type simpleEvaluator struct {
	cfg   Config
	deps  Deps
	cache Cache
}

func newEvaluator(cfg Config, deps Deps) *simpleEvaluator {
	c := deps.Cache
	if c == nil {
		c = NewMemoryCache()
	}
	return &simpleEvaluator{cfg: cfg, deps: deps, cache: c}
}

func (e *simpleEvaluator) Check(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, error) {
	dec, _, err := e.evaluate(ctx, subj, action, res, false)
	return dec, err
}

func (e *simpleEvaluator) Explain(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, Trace, error) {
	dec, trace, err := e.evaluate(ctx, subj, action, res, true)
	return dec, trace, err
}

func (e *simpleEvaluator) evaluate(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef, wantTrace bool) (Decision, Trace, error) {
	// Super roles bypass
	if e.isSuper(subj) {
		return DecisionAllow, Trace{}, nil
	}

	// Principal cache: subject + principalVersion + action + resource type
	if pv, err := e.deps.Store.GetPrincipalVersion(ctx, subj); err == nil {
		pKey := principalCacheKey(subj) + "|" + strconv.FormatInt(pv, 10) + "|" + string(action) + "|" + res.Type
		if cached, ok := e.cache.GetPrincipal(pKey); ok {
			// Evaluate cached entries
			for _, re := range cached {
				if e.cfg.EnableDeny && re.Effect == Deny {
					okPass, condErr := e.conditionsPass(ctx, subj, res, re.Conditions)
					if condErr != nil {
						return DecisionDeny, Trace{}, condErr
					}
					if okPass {
						return DecisionDeny, nilTrace(wantTrace), nil
					}
				}
			}
			for _, re := range cached {
				if re.Effect == Allow {
					okPass, condErr := e.conditionsPass(ctx, subj, res, re.Conditions)
					if condErr != nil {
						return DecisionDeny, Trace{}, condErr
					}
					if okPass {
						return DecisionAllow, nilTrace(wantTrace), nil
					}
				}
			}
			// fall through to deny
			return DecisionDeny, nilTrace(wantTrace), nil
		}
		// Remember key for later set by placing it in a local var in outer scope
		// (no-op here; set happens after filtering below)
	}

	// Gather bindings for subject
	bindings, err := e.deps.Store.ListBindings(ctx, subj)
	if err != nil {
		return DecisionDeny, Trace{}, err
	}

	// Determine resource ancestry
	resID := res.ID
	var projectID string
	var orgID string
	for p := &res; p != nil; p = p.Parent {
		if p.Type == "project" && projectID == "" {
			projectID = p.ID
		}
		if p.Type == "org" && orgID == "" {
			orgID = p.ID
		}
		if p.Parent == nil {
			break
		}
	}

	var entriesRes, entriesProject, entriesOrg, entriesGlobal []RoleEntry
	for _, b := range bindings {
		applies := false
		switch b.ScopeType {
		case ScopeGlobal:
			applies = true
		case ScopeRes:
			applies = b.ScopeID == resID
		case ScopeProject:
			applies = projectID != "" && b.ScopeID == projectID
		case ScopeOrg:
			applies = orgID != "" && b.ScopeID == orgID
		}
		if !applies {
			continue
		}

		role, err := e.deps.Store.GetRole(ctx, b.RoleSlug)
		if err != nil {
			return DecisionDeny, Trace{}, err
		}
		if role == nil {
			continue
		}

		compiled, ok := e.cache.GetRoleEntries(role.Slug, role.Version)
		if !ok {
			compiled = role.Entries
			e.cache.SetRoleEntries(role.Slug, role.Version, compiled, e.cfg.RoleCacheTTL)
		}
		for _, re := range compiled {
			if strings.EqualFold(string(re.Permission), string(action)) {
				if re.ResourceType == "" || strings.EqualFold(re.ResourceType, res.Type) {
					switch b.ScopeType {
					case ScopeRes:
						entriesRes = append(entriesRes, re)
					case ScopeProject:
						entriesProject = append(entriesProject, re)
					case ScopeOrg:
						entriesOrg = append(entriesOrg, re)
					case ScopeGlobal:
						entriesGlobal = append(entriesGlobal, re)
					}
				}
			}
		}
	}

	// Cache combined entries
	if pv, err := e.deps.Store.GetPrincipalVersion(ctx, subj); err == nil {
		pKey := principalCacheKey(subj) + "|" + strconv.FormatInt(pv, 10) + "|" + string(action) + "|" + res.Type
		combined := make([]RoleEntry, 0, len(entriesRes)+len(entriesProject)+len(entriesOrg)+len(entriesGlobal))
		combined = append(combined, entriesRes...)
		combined = append(combined, entriesProject...)
		combined = append(combined, entriesOrg...)
		combined = append(combined, entriesGlobal...)
		e.cache.SetPrincipal(pKey, combined, e.cfg.PrincipalCacheTTL)
	}

	// Deny across scopes first
	for _, list := range [][]RoleEntry{entriesRes, entriesProject, entriesOrg, entriesGlobal} {
		for _, re := range list {
			if e.cfg.EnableDeny && re.Effect == Deny {
				ok, condErr := e.conditionsPass(ctx, subj, res, re.Conditions)
				if condErr != nil {
					return DecisionDeny, Trace{}, condErr
				}
				if ok {
					return DecisionDeny, nilTrace(wantTrace), nil
				}
			}
		}
	}

	// Allows in specificity order
	for _, list := range [][]RoleEntry{entriesRes, entriesProject, entriesOrg, entriesGlobal} {
		for _, re := range list {
			if re.Effect == Allow {
				ok, condErr := e.conditionsPass(ctx, subj, res, re.Conditions)
				if condErr != nil {
					return DecisionDeny, Trace{}, condErr
				}
				if ok {
					return DecisionAllow, nilTrace(wantTrace), nil
				}
			}
		}
	}
	return DecisionDeny, nilTrace(wantTrace), nil
}

func (e *simpleEvaluator) conditionsPass(ctx context.Context, subj Subject, res ResourceRef, conds []Condition) (bool, error) {
	if len(conds) == 0 {
		return true, nil
	}
	// Build eval context
	ec := EvalContext{Subject: subj, Resource: res}
	// If authkit clock exists, use it
	if e.deps.Clock != nil {
		ec.Now = e.deps.Clock.Now()
	}
	for _, c := range conds {
		ev, ok := getCondition(c.Name)
		if !ok {
			return false, fmt.Errorf("unknown condition: %s", c.Name)
		}
		okPass, _, err := ev.Evaluate(ec, c.Params)
		if err != nil {
			// Map known condition error
			if c.Name == "requires_step_up" {
				return false, ErrConditionStepUpRequired
			}
			return false, err
		}
		if !okPass {
			if c.Name == "requires_step_up" {
				return false, ErrConditionStepUpRequired
			}
			return false, nil
		}
	}
	return true, nil
}

func (e *simpleEvaluator) isSuper(subj Subject) bool {
	if len(e.cfg.SuperRoles) == 0 {
		return false
	}
	if subj.Attrs == nil {
		return false
	}
	// Expect a roles attribute for super roles check if present
	if v, ok := subj.Attrs["roles"]; ok {
		if roles, ok := v.([]string); ok {
			for _, sr := range e.cfg.SuperRoles {
				for _, have := range roles {
					if strings.EqualFold(sr, have) {
						return true
					}
				}
			}
		}
	}
	return false
}

func nilTrace(want bool) Trace {
	if want {
		return Trace{}
	}
	return Trace{}
}

// principalCacheKey builds a stable cache key from subject and optional org.
func principalCacheKey(subj Subject) string {
	h := sha1.New()
	h.Write([]byte(string(subj.Type)))
	h.Write([]byte{"|"[0]})
	h.Write([]byte(subj.ID))
	if subj.OrgID != nil {
		h.Write([]byte("|" + subj.OrgID.String()))
	}
	return hex.EncodeToString(h.Sum(nil))
}
