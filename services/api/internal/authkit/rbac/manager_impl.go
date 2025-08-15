package rbac

import (
	"context"
	"sync"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/gin-gonic/gin"
)

type manager struct {
	cfg       Config
	deps      Deps
	eval      Evaluator
	resolvers map[string]ResourceResolver
	mu        sync.RWMutex
}

func newManager(cfg Config, deps Deps) *manager {
	return &manager{
		cfg:       cfg,
		deps:      deps,
		eval:      newEvaluator(cfg, deps),
		resolvers: make(map[string]ResourceResolver),
	}
}

func (m *manager) WithPolicyRegistry(reg *authkit.PolicyRegistry) *authkit.PolicyRegistry { return reg }

func (m *manager) RegisterResolver(pattern string, resolver ResourceResolver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resolvers[pattern] = resolver
}

func (m *manager) Check(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, error) {
	return m.eval.Check(ctx, subj, action, res)
}

func (m *manager) Explain(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, Trace, error) {
	return m.eval.Explain(ctx, subj, action, res)
}

func (m *manager) Resolve(c *gin.Context, path string) (ResourceRef, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if resolver, ok := m.resolvers[path]; ok {
		ref, err := resolver(c)
		return ref, true, err
	}
	// simple exact match only for now; wildcard/regex support can be added later
	return ResourceRef{}, false, nil
}
