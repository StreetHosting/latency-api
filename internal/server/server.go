package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/streethosting/latency-api/internal/cors"
	"github.com/streethosting/latency-api/internal/mtr"
)

// Config holds probe server settings.
type Config struct {
	ListenAddr             string
	AllowedOrigins         string
	AllowedOriginSuffixes  string
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	ShutdownTimeout        time.Duration
	MTR_ENABLED            bool
	MTR_BIN                string
	MTR_CYCLES             int
	MTR_TIMEOUT            time.Duration
	MTR_MIN_INTERVAL       time.Duration
}

// New builds the HTTP handler and server.
func New(cfg Config, log *slog.Logger) *http.Server {
	policy := cors.NewPolicy(cfg.AllowedOrigins, cfg.AllowedOriginSuffixes)

	mux := http.NewServeMux()
	mux.Handle("/ping", cors.Middleware(policy)(http.HandlerFunc(pingHandler)))

	mtrH := &mtrHandler{
		log:     log,
		enabled: cfg.MTR_ENABLED,
		opt: mtr.Options{
			Binary:  cfg.MTR_BIN,
			Cycles:  cfg.MTR_CYCLES,
			Timeout: cfg.MTR_TIMEOUT,
		},
		limiter: mtr.NewLimiter(cfg.MTR_MIN_INTERVAL),
	}
	mux.Handle("/mtr", cors.Middleware(policy)(http.HandlerFunc(mtrH.serveHTTP)))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "")
		mux.ServeHTTP(w, r)
	})

	return &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// Run starts the server and blocks until ctx is cancelled, then shuts down gracefully.
func Run(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("probe listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Info("shutting down")
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
