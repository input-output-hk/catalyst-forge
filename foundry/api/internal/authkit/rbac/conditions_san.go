package rbac

import (
	"net"
	"strings"
)

// conditionDnsSANsSuffixIn ensures all DNS SANs end with one of the allowed suffixes.
// Params: {"suffixes": [".example.com", "*.projectcatalyst.io"]}
type conditionDnsSANsSuffixIn struct{}

func (conditionDnsSANsSuffixIn) Name() string { return "dns_sans_suffix_in" }

func (conditionDnsSANsSuffixIn) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	anySuffixes, _ := params["suffixes"].([]any)
	var suffixes []string
	for _, s := range anySuffixes {
		if v, ok := s.(string); ok {
			suffixes = append(suffixes, strings.ToLower(strings.TrimSpace(v)))
		}
	}
	sansAny, ok := ctx.Resource.Attrs["dns_sans"]
	if !ok {
		return false, "dns_sans missing", nil
	}
	var sans []string
	switch v := sansAny.(type) {
	case []string:
		sans = v
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok {
				sans = append(sans, s)
			}
		}
	default:
		return false, "dns_sans wrong type", nil
	}
	for _, name := range sans {
		ln := strings.ToLower(name)
		matched := false
		for _, suf := range suffixes {
			if strings.HasPrefix(suf, "*.") {
				base := strings.TrimPrefix(suf, "*.")
				if strings.HasSuffix(ln, base) && ln != base {
					matched = true
					break
				}
			} else if strings.HasSuffix(ln, suf) {
				matched = true
				break
			}
		}
		if !matched {
			return false, "dns san not allowed", nil
		}
	}
	return true, "dns sans allowed", nil
}

// conditionURISANsPrefixIn ensures all URI SANs start with one of the allowed prefixes.
// Params: {"prefixes": ["spiffe://org/"]}
type conditionURISANsPrefixIn struct{}

func (conditionURISANsPrefixIn) Name() string { return "uri_sans_prefix_in" }

func (conditionURISANsPrefixIn) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	anyPrefixes, _ := params["prefixes"].([]any)
	var prefixes []string
	for _, s := range anyPrefixes {
		if v, ok := s.(string); ok {
			prefixes = append(prefixes, v)
		}
	}
	sansAny, ok := ctx.Resource.Attrs["uri_sans"]
	if !ok {
		return true, "no uri sans", nil
	}
	var sans []string
	switch v := sansAny.(type) {
	case []string:
		sans = v
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok {
				sans = append(sans, s)
			}
		}
	default:
		return false, "uri_sans wrong type", nil
	}
	for _, u := range sans {
		allowed := false
		for _, p := range prefixes {
			if strings.HasPrefix(u, p) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, "uri san not allowed", nil
		}
	}
	return true, "uri sans allowed", nil
}

// conditionIPSANsInCIDRs ensures all IP SANs fall within allowed CIDR ranges.
// Params: {"cidrs": ["10.0.0.0/8", "fd00::/8"]}
type conditionIPSANsInCIDRs struct{}

func (conditionIPSANsInCIDRs) Name() string { return "ip_sans_in_cidrs" }

func (conditionIPSANsInCIDRs) Evaluate(ctx EvalContext, params map[string]any) (bool, string, error) {
	anyCIDRs, _ := params["cidrs"].([]any)
	var nets []*net.IPNet
	for _, s := range anyCIDRs {
		if v, ok := s.(string); ok {
			if _, n, err := net.ParseCIDR(strings.TrimSpace(v)); err == nil {
				nets = append(nets, n)
			}
		}
	}
	sansAny, ok := ctx.Resource.Attrs["ip_sans"]
	if !ok {
		return true, "no ip sans", nil
	}
	var sans []string
	switch v := sansAny.(type) {
	case []string:
		sans = v
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok {
				sans = append(sans, s)
			}
		}
	default:
		return false, "ip_sans wrong type", nil
	}
	for _, ipStr := range sans {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false, "invalid ip san", nil
		}
		allowed := false
		for _, n := range nets {
			if n.Contains(ip) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, "ip san not in cidr", nil
		}
	}
	return true, "ip sans allowed", nil
}

func init() {
	RegisterCondition(conditionDnsSANsSuffixIn{})
	RegisterCondition(conditionURISANsPrefixIn{})
	RegisterCondition(conditionIPSANsInCIDRs{})
}
