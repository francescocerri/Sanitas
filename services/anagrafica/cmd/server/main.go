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

	"github.com/francescocerri/sanitas/services/anagrafica/internal/config"
	"github.com/francescocerri/sanitas/services/anagrafica/internal/httpapi"
	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

// @title			Sanitas — Anagrafica API
// @version		0.1.0
// @description	Users, roles, authentication. Phase A: admin-created users, activation via
// @description	token, login, password change. Sending real email (invites/forgot-password)
// @description	is a later phase — see docs/adr/0013.
//
// No @BasePath: Swagger 2.0 has no per-path basePath override, and this API
// mixes versioned (/v1/...) and unversioned (/healthz, /.well-known/...)
// routes — a global BasePath would make Swagger UI's "Try it out" call the
// wrong URL for the unversioned ones. Each @Router below spells out its
// real registered path instead.
func main() {
	// JSON to stdout: meant to be read by `docker logs`/a log aggregator,
	// not by a human on a screen (see turni's ADR-0010).
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	keys, err := user.LoadKeyPair(cfg.JWTPrivateKeyPath)
	if err != nil {
		logger.Error("load JWT key pair failed", "error", err)
		os.Exit(1)
	}

	repo := user.NewRepository(pool)

	if cfg.RolesSeedPath != "" {
		if err := user.SeedRoles(ctx, repo, cfg.RolesSeedPath); err != nil {
			logger.Error("seed roles failed", "error", err)
			os.Exit(1)
		}
	}

	if err := user.Bootstrap(ctx, repo, cfg.InitialAdminEmail, cfg.InitialAdminUsername, cfg.InitialAdminPassword); err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}

	server := httpapi.NewServer(repo, keys, cfg.CORSAllowedOrigin, cfg.InviteURLBase, logger)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.Routes(),
		// Beyond ReadHeaderTimeout (which only bounds reading the headers),
		// these are needed too so a slow client connection can't stay open
		// indefinitely (slowloris-style risk) — see turni's ADR-0010.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("anagrafica service listening", "port", cfg.Port)
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
