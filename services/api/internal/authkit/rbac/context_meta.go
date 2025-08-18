package rbac

import "context"

type ctxMetaKeyType struct{}

var ctxMetaKey = ctxMetaKeyType{}

// WithRequestMeta stores request path and method for contextual planning.
func WithRequestMeta(ctx context.Context, path, method string) context.Context {
	if ctx == nil {
		return context.Background()
	}
	m := map[string]string{"path": path, "method": method}
	return context.WithValue(ctx, ctxMetaKey, m)
}

// ExtractRequestMeta returns meta map if present; otherwise a non-nil empty map.
func ExtractRequestMeta(ctx context.Context) map[string]string {
	if ctx == nil {
		return map[string]string{}
	}
	if v := ctx.Value(ctxMetaKey); v != nil {
		if m, ok := v.(map[string]string); ok && m != nil {
			return m
		}
	}
	return map[string]string{}
}
