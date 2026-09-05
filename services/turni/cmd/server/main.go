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

	"github.com/francescocerri/sanitas/services/turni/internal/authclient"
	"github.com/francescocerri/sanitas/services/turni/internal/config"
	"github.com/francescocerri/sanitas/services/turni/internal/httpapi"
	"github.com/francescocerri/sanitas/services/turni/internal/schema"
	"github.com/francescocerri/sanitas/services/turni/internal/turno"
)

// jwksFetchAttempts/jwksFetchInterval: anagrafica and turni start together
// in docker-compose with no ordering guarantee between them (anagrafica has
// no container healthcheck to depend_on), so the first fetch attempt(s)
// failing is the normal case, not an error — this bounds the wait instead
// of retrying forever.
//
// schemaCreateAttempts/schemaCreateInterval: same reasoning, for turni's
// own schema — anagrafica moved schema creation from docker-entrypoint-initdb.d
// (which ran before either binary started, in a guaranteed order) to its own
// startup (AutoMigrate, see docs/adr/0019), so turni's FK into
// anagrafica.users can no longer assume that table already exists when
// turni's own schema is created — it might not have run yet.
const (
	jwksFetchAttempts = 5
	jwksFetchInterval = 2 * time.Second

	schemaCreateAttempts = 5
	schemaCreateInterval = 2 * time.Second
)

// @title			Sanitas — Turni API
// @version		0.1.0
// @description	Turno (shift) management. The data model is intentionally skeletal (see docs/adr/0005
// @description	in the repository): it validates the end-to-end pipeline, not the final domain design.
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				turni issues no tokens itself — type "Bearer" followed by a space and
// @description				the access token obtained from anagrafica's POST /v1/login (e.g. "Bearer eyJhbGci...").
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

	if err := createSchemaWithRetry(ctx, pool, logger); err != nil {
		logger.Error("create schema failed", "error", err)
		os.Exit(1)
	}

	authClient := authclient.New(cfg.AuthJWKSURL)
	if err := fetchJWKSWithRetry(ctx, authClient, logger); err != nil {
		logger.Error("fetch JWKS from anagrafica failed", "error", err)
		os.Exit(1)
	}

	repo := turno.NewRepository(pool)
	server := httpapi.NewServer(repo, authClient, cfg.CORSAllowedOrigin, logger)

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

// createSchemaWithRetry applies schema.SQL, retrying on failure — turni's
// FK into anagrafica.users may briefly not resolve if anagrafica's own
// AutoMigrate hasn't run yet, same reasoning as fetchJWKSWithRetry below.
func createSchemaWithRetry(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	var err error
	for attempt := 1; attempt <= schemaCreateAttempts; attempt++ {
		if _, err = pool.Exec(ctx, schema.SQL); err == nil {
			return nil
		}
		logger.Info("create schema: attempt failed, retrying", "attempt", attempt, "error", err)
		select {
		case <-time.After(schemaCreateInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// fetchJWKSWithRetry keeps trying (a few times, a short pause apart) instead
// of failing on the first attempt — see the const doc comment above for why
// a transient failure here is expected, not exceptional.
func fetchJWKSWithRetry(ctx context.Context, client *authclient.Client, logger *slog.Logger) error {
	var err error
	for attempt := 1; attempt <= jwksFetchAttempts; attempt++ {
		if err = client.Refresh(ctx); err == nil {
			return nil
		}
		logger.Info("fetch JWKS: attempt failed, retrying", "attempt", attempt, "error", err)
		select {
		case <-time.After(jwksFetchInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
