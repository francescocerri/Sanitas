package main

import (
	"context"
	"log"
	"net/http"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configurazione non valida: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connessione al database fallita: %v", err)
	}
	defer pool.Close()

	repo := turno.NewRepository(pool)
	server := httpapi.NewServer(repo, cfg.CORSAllowedOrigin)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("turni service in ascolto sulla porta %s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("errore http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown in corso...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("errore durante shutdown: %v", err)
	}
}
