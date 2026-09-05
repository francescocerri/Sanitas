package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/turni/internal/config"
	"github.com/francescocerri/sanitas/services/turni/internal/httpapi"
	"github.com/francescocerri/sanitas/services/turni/internal/turno"
)

// @title			Sanitas — Turni API
// @version		0.1.0
// @description	Turno (shift) management. The data model is intentionally skeletal (see docs/adr/0005
// @description	in the repository): it validates the end-to-end pipeline, not the final domain design.
//
// No @BasePath: Swagger 2.0 has no per-path basePath override, and this API
// mixes versioned (/v1/...) and unversioned (/healthz) routes — a global
// BasePath would make Swagger UI's "Try it out" call the wrong URL for the
// unversioned ones (see docs/adr/0010). Each @Router below spells out its
// real registered path instead.
func main() {
	// JSON to stdout: meant to be read by `docker logs`/a log aggregator,
	// not by a human on a screen (see ADR-0010).
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// SIGTERM is what Docker/an orchestrator sends to ask for a clean
	// stop (docker stop, deploy, restart) before killing the process:
	// without catching it here, the server would be cut off mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := turno.NewRepository(pool)
	server := httpapi.NewServer(repo, cfg.CORSAllowedOrigin, logger)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.Routes(),
		// Beyond ReadHeaderTimeout (which only bounds reading the headers),
		// these are needed too so a slow client connection can't stay open
		// indefinitely (slowloris-style risk) — see ADR-0010.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("turni service listening", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	// A fresh context (not the one already Done() above): just gives the
	// clean stop a time budget, independent of whatever signal caused it.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
