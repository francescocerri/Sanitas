# Sanitas

Insieme di microservizi per la gestione operativa di un'associazione di soccorso sanitario (turni, anagrafica volontari, mezzi e magazzino, servizi ed emergenze). Sviluppato per il Comitato CRI di Pavullo, ma progettato per essere **forkato e riusato da altre associazioni** (CRI, Croce Verde, Croce Blu, ecc.).

Dettagli su architettura, convenzioni e contratto di forkabilità: vedi [CLAUDE.md](CLAUDE.md).

## Servizi

- [`services/turni`](services/turni) — gestione turni (Go)
- [`web`](web) — frontend React + TypeScript

## Sviluppo locale

```bash
cd deploy
cp .env.example .env   # compilare POSTGRES_PASSWORD; CLOUDFLARE_TUNNEL_TOKEN serve solo in prod
docker compose up -d --build postgres turni

cd ../web
npm install
npm run dev
```

## Fork per altre associazioni

Il progetto segue il modello *fork + configurazione*: ogni associazione fa il fork del repository, personalizza branding e dati in `config/<slug>/` e deploya la propria istanza indipendente. Nessun dato specifico del Comitato di Pavullo è hardcoded nel codice sorgente dei servizi.
