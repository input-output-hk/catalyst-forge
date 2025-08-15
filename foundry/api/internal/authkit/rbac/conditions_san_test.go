package rbac

import (
	"net"
	"testing"
	"time"
)

func TestCondition_DNSSuffixIn(t *testing.T) {
	ev, ok := getCondition("dns_sans_suffix_in")
	if !ok {
		t.Fatal("condition not registered")
	}
	ctx := EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"dns_sans": []string{"a.projectcatalyst.io", "b.svc.cluster.local"}}}}
	pass, _, _ := ev.Evaluate(ctx, map[string]any{"suffixes": []any{".projectcatalyst.io", ".svc.cluster.local"}})
	if !pass {
		t.Fatal("expected pass")
	}

	ctx = EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"dns_sans": []string{"bad.example.com"}}}}
	pass, _, _ = ev.Evaluate(ctx, map[string]any{"suffixes": []any{".projectcatalyst.io"}})
	if pass {
		t.Fatal("expected deny")
	}
}

func TestCondition_URIPrefixIn(t *testing.T) {
	ev, ok := getCondition("uri_sans_prefix_in")
	if !ok {
		t.Fatal("condition not registered")
	}
	ctx := EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"uri_sans": []string{"spiffe://org/service/1"}}}}
	pass, _, _ := ev.Evaluate(ctx, map[string]any{"prefixes": []any{"spiffe://org/"}})
	if !pass {
		t.Fatal("expected pass")
	}

	ctx = EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"uri_sans": []string{"spiffe://other/1"}}}}
	pass, _, _ = ev.Evaluate(ctx, map[string]any{"prefixes": []any{"spiffe://org/"}})
	if pass {
		t.Fatal("expected deny")
	}
}

func TestCondition_IPCIDRs(t *testing.T) {
	ev, ok := getCondition("ip_sans_in_cidrs")
	if !ok {
		t.Fatal("condition not registered")
	}
	// valid range
	ctx := EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"ip_sans": []string{"10.1.2.3"}}}}
	pass, _, _ := ev.Evaluate(ctx, map[string]any{"cidrs": []any{"10.0.0.0/8"}})
	if !pass {
		t.Fatal("expected pass")
	}

	// invalid ip format
	ctx = EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"ip_sans": []string{"not_an_ip"}}}}
	pass, _, _ = ev.Evaluate(ctx, map[string]any{"cidrs": []any{"10.0.0.0/8"}})
	if pass {
		t.Fatal("expected deny for invalid IP")
	}

	// ipv6 example
	_, n, _ := net.ParseCIDR("fd00::/8")
	_ = n
	ctx = EvalContext{Now: time.Now().UTC(), Resource: ResourceRef{Attrs: map[string]any{"ip_sans": []string{"fd00::1"}}}}
	pass, _, _ = ev.Evaluate(ctx, map[string]any{"cidrs": []any{"fd00::/8"}})
	if !pass {
		t.Fatal("expected pass for ipv6")
	}
}
