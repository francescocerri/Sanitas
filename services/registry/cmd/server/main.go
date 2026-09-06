package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/francescocerri/sanitas/services/registry/internal/config"
	"github.com/francescocerri/sanitas/services/registry/internal/httpapi"
	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

// @title			Sanitas — Registry API
// @version		0.1.0
// @description	Users, roles, authentication. Admin-created users, activation via token,
// @description	login, password change. Invite emails are sent via SMTP when configured
// @description	(see docs/adr/0023), otherwise the invite link is returned in the API
// @description	response for the admin to forward by hand.
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
	// not by a human on a screen (see shifts' ADR-0010).
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
	// shifts still creates its own schema/table at its own startup, with
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

	emailBranding, err := user.LoadEmailBranding(cfg.EmailConfigPath)
	if err != nil {
		logger.Error("load email branding failed", "error", err)
		os.Exit(1)
	}

	// mailer resta nil se SMTPHost non è configurato: l'invio dell'email di
	// invito è opzionale, un fork che non lo configura mantiene il
	// comportamento di sempre (link restituito nella risposta API) — vedi
	// docs/adr/0023-invio-email-invito-smtp.md.
	var mailer *user.Mailer
	if cfg.SMTPHost != "" {
		port, err := strconv.Atoi(cfg.SMTPPort)
		if err != nil {
			logger.Error("invalid SMTP_PORT", "value", cfg.SMTPPort, "error", err)
			os.Exit(1)
		}
		mailer, err = user.NewMailer(cfg.SMTPHost, port, cfg.SMTPUsername, cfg.SMTPPassword, emailBranding)
		if err != nil {
			logger.Error("configure SMTP mailer failed", "error", err)
			os.Exit(1)
		}
	}

	server := httpapi.NewServer(repo, keys, cfg.CORSAllowedOrigin, cfg.InviteURLBase, mailer, emailBranding, logger)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.Routes(),
		// Beyond ReadHeaderTimeout (which only bounds reading the headers),
		// these are needed too so a slow client connection can't stay open
		// indefinitely (slowloris-style risk) — see shifts' ADR-0010.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("registry service listening", "port", cfg.Port)
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
