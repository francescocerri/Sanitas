# Deploy — Raspberry Pi

Target di produzione a costo ricorrente ~zero: Docker Compose sul Raspberry Pi, esposto via Cloudflare Tunnel, deploy continuo tramite un GitHub Actions self-hosted runner installato sul Pi stesso (build Docker nativa ARM, nessun accesso SSH da esporre verso casa).

## Setup una tantum sul Raspberry Pi

1. **Docker**: installare Docker Engine + plugin Compose (`curl -fsSL https://get.docker.com | sh`, poi aggiungere l'utente al gruppo `docker`).
2. **Cloudflare Tunnel**: nel dashboard Cloudflare Zero Trust → Networks → Tunnels, creare un tunnel, mappare il hostname pubblico (es. `turni.<tuo-dominio>`) al servizio locale `http://turni:8080`, copiare il token generato.
3. **File `.env`**: copiare `deploy/.env.example` in `deploy/.env` sul Pi e compilarlo (password Postgres, token del tunnel). **Non committare mai `.env`** (è già in `.gitignore`).
4. **GitHub Actions self-hosted runner**: da Settings → Actions → Runners del repo GitHub, seguire le istruzioni per registrare un runner Linux/ARM64 sul Raspberry Pi, installarlo come servizio (`./svc.sh install && ./svc.sh start`) così riparte da solo al riavvio.

## Deploy

Il workflow [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) gira automaticamente sul self-hosted runner ad ogni push su `main` che supera la CI: `git pull` + `docker compose --profile prod up -d --build`.

Per un primo avvio manuale (o per debug) dalla cartella `deploy/` sul Pi:

```bash
docker compose --profile prod up -d --build
```

## Note

- Il DB Postgres vive in un volume Docker locale sul Pi: non c'è ancora una strategia di backup automatico (da definire in una sessione futura, es. `pg_dump` periodico verso storage esterno).
- Essendo tutto containerizzato e configurato via env var, una futura migrazione ad AWS (EC2/ECS) richiede solo di ripuntare il job di deploy a una nuova macchina — nessuna modifica al codice dei servizi.
