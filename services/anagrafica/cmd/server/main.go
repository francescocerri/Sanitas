package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

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
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and the access token from POST /v1/login (e.g. "Bearer eyJhbGci...").
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

	// TranslateError: lets repository code check gorm.ErrDuplicatedKey
	// instead of reaching into a Postgres-specific error type — see
	// user.isUniqueViolation, docs/adr/0019. Silent logger: GORM's default
	// writes plain text to stdout, inconsistent with the JSON structured
	// logging used everywhere else (ADR-0010) — every error already gets
	// wrapped and logged through *slog.Logger at the call site, so this
	// isn't losing any real diagnostic signal.
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Replaces the SQL init script this service used to rely on
	// (docker-entrypoint-initdb.d only ever ran once, at first container
	// creation — see docs/adr/0019, superseding part of docs/adr/0005).
	// turni still creates its own schema/table at its own startup, with
	// retry, since it can no longer count on this script having already
	// run at Postgres container init time either.
	if err := user.Migrate(db); err != nil {
		logger.Error("migrate failed", "error", err)
		os.Exit(1)
	}

	keys, err := user.LoadKeyPair(cfg.JWTPrivateKeyPath)
	if err != nil {
		logger.Error("load JWT key pair failed", "error", err)
		os.Exit(1)
	}

	repo := user.NewRepository(db)

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
