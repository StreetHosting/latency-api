package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/streethosting/latency-api/internal/server"
)

const (
	defaultOrigins        = "http://localhost:3000"
	defaultOriginSuffixes = "streethosting.com.br,strt.host,ruas.run"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := server.Config{
		ListenAddr:            envOr("LISTEN_ADDR", "127.0.0.1:8080"),
		AllowedOrigins:        envOr("ALLOWED_ORIGINS", defaultOrigins),
		AllowedOriginSuffixes: envOr("ALLOWED_ORIGIN_SUFFIXES", defaultOriginSuffixes),
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}

	srv := server.New(cfg, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, srv, log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
