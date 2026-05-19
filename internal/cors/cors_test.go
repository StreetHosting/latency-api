package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/streethosting/latency-api/internal/cors"
)

func TestMiddleware_AllowedOrigin(t *testing.T) {
	allowed := cors.ParseOrigins("https://streethosting.com.br")
	called := false
	h := cors.Middleware(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://streethosting.com.br")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://streethosting.com.br" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestMiddleware_OptionsPreflight(t *testing.T) {
	allowed := cors.ParseOrigins("https://streethosting.com.br")
	h := cors.Middleware(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://streethosting.com.br")
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
	allowed := cors.ParseOrigins("https://streethosting.com.br")
	h := cors.Middleware(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
