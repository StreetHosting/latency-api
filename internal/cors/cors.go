package cors

import (
	"net/http"
	"net/url"
	"strings"
)

// DefaultSuffixes are base domains; any host equal to or ending in .<suffix> is allowed.
var DefaultSuffixes = []string{
	"streethosting.com.br",
	"strt.host",
	"ruas.run",
}

// Policy defines exact origins and permitted domain suffixes.
type Policy struct {
	Exact    map[string]struct{}
	Suffixes []string
}

// NewPolicy builds a policy from comma-separated env values.
func NewPolicy(exactRaw, suffixRaw string) Policy {
	suffixes := ParseSuffixes(suffixRaw)
	if len(suffixes) == 0 {
		suffixes = append([]string(nil), DefaultSuffixes...)
	}
	return Policy{
		Exact:    ParseOrigins(exactRaw),
		Suffixes: suffixes,
	}
}

// Allowed reports whether the Origin header may receive CORS headers.
func (p Policy) Allowed(origin string) bool {
	if origin == "" {
		return false
	}
	if _, ok := p.Exact[origin]; ok {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}

	for _, suffix := range p.Suffixes {
		suffix = strings.ToLower(strings.TrimSpace(suffix))
		if suffix == "" {
			continue
		}
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

// Middleware reflects Access-Control-Allow-Origin when Origin matches the policy.
func Middleware(p Policy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if p.Allowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ParseOrigins splits a comma-separated origin list into a set.
func ParseOrigins(raw string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}
	return allowed
}

// ParseSuffixes splits comma-separated base domains (without leading dot).
func ParseSuffixes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
