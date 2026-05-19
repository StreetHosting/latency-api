package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/streethosting/latency-api/internal/cors"
)

func testPolicy() cors.Policy {
	return cors.NewPolicy("http://localhost:3000", "streethosting.com.br,strt.host,ruas.run")
}

func TestMiddleware_ApexOrigin(t *testing.T) {
	h := cors.Middleware(testPolicy())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://streethosting.com.br")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://streethosting.com.br" {
		t.Fatalf("Allow-Origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestMiddleware_SubdomainOrigins(t *testing.T) {
	p := testPolicy()
	for _, origin := range []string{
		"https://www.streethosting.com.br",
		"https://preview.streethosting.com.br",
		"https://app.strt.host",
		"https://cdn.ruas.run",
	} {
		if !p.Allowed(origin) {
			t.Fatalf("expected allowed: %s", origin)
		}
	}
}

func TestMiddleware_OptionsPreflight(t *testing.T) {
	h := cors.Middleware(testPolicy())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://api.strt.host")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("missing Allow-Methods on preflight")
	}
}

func TestMiddleware_DisallowedOrigin(t *testing.T) {
	p := testPolicy()
	for _, origin := range []string{
		"https://evil.example",
		"https://streethosting.com.br.evil.com",
		"https://notstrt.host",
	} {
		if p.Allowed(origin) {
			t.Fatalf("expected denied: %s", origin)
		}
	}

	h := cors.Middleware(p)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("should not reflect disallowed origin")
	}
}

func TestMiddleware_LocalhostExact(t *testing.T) {
	p := testPolicy()
	if !p.Allowed("http://localhost:3000") {
		t.Fatal("localhost dev origin should be allowed via exact list")
	}
}
