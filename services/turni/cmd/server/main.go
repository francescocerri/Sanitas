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
// @description	Gestione turni. Modello dati volutamente scheletrico (vedi docs/adr/0005 nel repository):
// @description	valida la pipeline end-to-end, non è la progettazione definitiva del dominio.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configurazione non valida", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connessione al database fallita", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := turno.NewRepository(pool)
	server := httpapi.NewServer(repo, cfg.CORSAllowedOrigin, logger)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("turni service in ascolto", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("errore http server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown in corso")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("errore durante shutdown", "error", err)
	}
}
