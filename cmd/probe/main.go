package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
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

	mtrTimeout := envOrDuration("MTR_TIMEOUT", 45*time.Second)

	cfg := server.Config{
		ListenAddr:            envOr("LISTEN_ADDR", "127.0.0.1:8080"),
		AllowedOrigins:        envOr("ALLOWED_ORIGINS", defaultOrigins),
		AllowedOriginSuffixes: envOr("ALLOWED_ORIGIN_SUFFIXES", defaultOriginSuffixes),
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          mtrTimeout + 15*time.Second,
		IdleTimeout:           30 * time.Second,
		ShutdownTimeout:       10 * time.Second,
		MTR_ENABLED:           envOrBool("MTR_ENABLED", true),
		MTR_BIN:               envOr("MTR_BIN", "/usr/bin/mtr"),
		MTR_CYCLES:            envOrInt("MTR_CYCLES", 10),
		MTR_TIMEOUT:           mtrTimeout,
		MTR_MIN_INTERVAL:      envOrDuration("MTR_MIN_INTERVAL", 60*time.Second),
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

func envOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envOrDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return fallback
}
