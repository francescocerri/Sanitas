# Sanitas

Insieme di microservizi per la gestione operativa di un'associazione di soccorso sanitario (turni, anagrafica volontari, mezzi e magazzino, servizi ed emergenze). Sviluppato per il Comitato CRI di Pavullo, ma progettato per essere **forkato e riusato da altre associazioni** (CRI, Croce Verde, Croce Blu, ecc.).

Dettagli su architettura, convenzioni e contratto di forkabilità: vedi [CLAUDE.md](CLAUDE.md).

## Servizi

- [`services/shifts`](services/shifts) — gestione turni (Go)
- [`services/registry`](services/registry) — utenti, ruoli, autenticazione (Go)
- [`web`](web) — frontend React + TypeScript

Entrambi i servizi Go condividono un solo Postgres, uno schema per servizio, creato al primo avvio via GORM AutoMigrate (vedi [ADR-0019](docs/adr/0019-gorm-e-automigrate.md), [ADR-0020](docs/adr/0020-shifts-gorm.md)) — nessuno script SQL da applicare a mano.

## Sviluppo locale

```bash
cd deploy
cp .env.example .env   # compilare POSTGRES_PASSWORD; vedi services/registry/README.md per la chiave JWT; CLOUDFLARE_TUNNEL_TOKEN serve solo in prod
docker compose up -d --build postgres registry shifts

cd ../web
npm install
npm run dev
```

## Fork per altre associazioni

Il progetto segue il modello *fork + configurazione*: ogni associazione fa il fork del repository, personalizza branding e dati in `config/<slug>/` e deploya la propria istanza indipendente. Nessun dato specifico del Comitato di Pavullo è hardcoded nel codice sorgente dei servizi.
