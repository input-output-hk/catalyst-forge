package rbac

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	condMu   sync.RWMutex
	condRegs = map[string]ConditionEvaluator{}
)

// RegisterCondition registers a condition evaluator by its Name().
func RegisterCondition(e ConditionEvaluator) {
	condMu.Lock()
	defer condMu.Unlock()
	condRegs[strings.ToLower(e.Name())] = e
}

func getCondition(name string) (ConditionEvaluator, bool) {
	condMu.RLock()
	defer condMu.RUnlock()
	e, ok := condRegs[strings.ToLower(name)]
	return e, ok
}

// Built-in conditions

// conditionStepUp enforces a recent step-up window.
// Params: {"within":"5m"}
type conditionStepUp struct{}

func (conditionStepUp) Name() string { return "requires_step_up" }

func (conditionStepUp) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	withinStr, _ := params["within"].(string)
	if withinStr == "" {
		return false, "missing within window", errors.New("requires_step_up: missing within")
	}
	within, err := time.ParseDuration(withinStr)
	if err != nil {
		return false, "invalid within window", err
	}
	now := ctx.Now
	// Must have a performed-at timestamp
	if ctx.StepUpAt.IsZero() {
		return false, "no step-up performed", nil
	}
	// Must not be expired
	if ctx.StepUpValidUntil.IsZero() || !ctx.StepUpValidUntil.After(now) {
		return false, "step-up expired", nil
	}
	// Pass if performed within the window
	elapsed := now.Sub(ctx.StepUpAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed <= within {
		return true, "step-up within window", nil
	}
	return false, "step-up too old", nil
}

// conditionAttrEquals compares a resource attribute with a subject attribute.
// Params: {"attr":"owner_id", "subject_field":"id"}
type conditionAttrEquals struct{}

func (conditionAttrEquals) Name() string { return "attr_equals" }

func (conditionAttrEquals) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	attrName, _ := params["attr"].(string)
	subjField, _ := params["subject_field"].(string)
	if attrName == "" || subjField == "" {
		return false, "missing parameters", nil
	}
	resVal, _ := ctx.Resource.Attrs[attrName]
	var subjVal any
	if strings.EqualFold(subjField, "id") {
		subjVal = ctx.Subject.ID
	} else {
		subjVal, _ = ctx.Subject.Attrs[subjField]
	}
	if resVal == nil || subjVal == nil {
		return false, "attribute missing", nil
	}
	return resVal == subjVal, " compared", nil
}

// conditionOrgMatches ensures subject and resource orgs align if both present.
type conditionOrgMatches struct{}

func (conditionOrgMatches) Name() string { return "org_matches" }

func (conditionOrgMatches) Evaluate(ctx EvalContext, _ map[string]any) (bool, string, error) {
	if ctx.Subject.OrgID == nil || ctx.Resource.OrgID == nil {
		return true, "org not set on one side", nil
	}
	return *ctx.Subject.OrgID == *ctx.Resource.OrgID, "orgs compared", nil
}

// conditionTimeWindow restricts access to specific hours/days (UTC).
// Params: {"start":"09:00Z", "end":"17:00Z", "days":["Mon","Tue",...]}
type conditionTimeWindow struct{}

func (conditionTimeWindow) Name() string { return "time_window" }

func (conditionTimeWindow) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	startStr, _ := params["start"].(string)
	endStr, _ := params["end"].(string)
	daysAny, _ := params["days"].([]any)

	if startStr == "" || endStr == "" {
		return false, "missing start/end", nil
	}
	// Parse HH:MMZ
	parseTOD := func(s string) (hour, min int, ok bool) {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(strings.ToUpper(s), "Z") {
			s = s[:len(s)-1]
		}
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, 0, false
		}
		var h, m int
		_, err1 := fmtSscanf(parts[0], &h)
		_, err2 := fmtSscanf(parts[1], &m)
		if err1 != nil || err2 != nil {
			return 0, 0, false
		}
		return h, m, true
	}
	sh, sm, ok := parseTOD(startStr)
	if !ok {
		return false, "invalid start", nil
	}
	eh, em, ok := parseTOD(endStr)
	if !ok {
		return false, "invalid end", nil
	}

	// Days filter
	allowedDays := map[string]struct{}{}
	for _, d := range daysAny {
		if s, ok := d.(string); ok {
			allowedDays[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
		}
	}
	now := ctx.Now.UTC()
	if len(allowedDays) > 0 {
		wd := strings.ToLower(now.Weekday().String()[:3]) // Mon, Tue, ...
		if _, ok := allowedDays[wd]; !ok {
			return false, "day not allowed", nil
		}
	}
	// Time of day window
	tod := now.Hour()*60 + now.Minute()
	startMin := sh*60 + sm
	endMin := eh*60 + em
	if startMin <= endMin {
		return tod >= startMin && tod <= endMin, "window checked", nil
	}
	// Overnight window
	return tod >= startMin || tod <= endMin, "overnight window checked", nil
}

// fmtSscanf is a tiny helper to avoid importing fmt for two ints.
func fmtSscanf(s string, out *int) (int, error) {
	n := 0
	val := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid int")
		}
		val = val*10 + int(ch-'0')
		n++
	}
	*out = val
	return n, nil
}

// init registers built-in conditions.
func init() {
	RegisterCondition(conditionStepUp{})
	RegisterCondition(conditionAttrEquals{})
	RegisterCondition(conditionOrgMatches{})
	RegisterCondition(conditionTimeWindow{})
}

// NOTE: SAN-related conditions will be appended by a follow-on edit to keep this change minimal.
