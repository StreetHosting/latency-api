package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/streethosting/latency-api/internal/clientip"
	"github.com/streethosting/latency-api/internal/mtr"
)

type mtrHandler struct {
	log     *slog.Logger
	enabled bool
	opt     mtr.Options
	limiter *mtr.Limiter
}

func (h *mtrHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		http.Error(w, `{"error":"mtr disabled"}`, http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	target := clientip.FromRequest(r)
	if target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "client IP unavailable (requires X-Real-IP from reverse proxy)",
		})
		return
	}

	if !h.limiter.Allow(target) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "rate limit: one MTR per client IP per minute",
		})
		return
	}

	report, err := mtr.Run(r.Context(), target, h.opt)
	if err != nil {
		h.log.Warn("mtr failed", "target", target, "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":  "mtr execution failed",
			"detail": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
