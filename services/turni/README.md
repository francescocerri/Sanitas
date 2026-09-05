# turni

Microservizio Go per la gestione turni. Modello dati volutamente scheletrico (vedi [docs/adr/0005](../../docs/adr/0005-database-postgres-self-hosted.md)): serve a validare la pipeline, non è la progettazione definitiva del dominio.

## Eseguire in locale (con Postgres via Docker Compose)

Il modo più semplice: usa lo stesso `docker-compose.yml` pensato per il deploy, che builda l'immagine e avvia anche Postgres.

```bash
cd ../../deploy
cp .env.example .env   # imposta almeno POSTGRES_PASSWORD
docker compose up -d --build postgres turni
```

Il servizio è raggiungibile su `http://localhost:8080` (o sulla porta impostata in `TURNI_HOST_PORT` in `.env`, se la 8080 è già occupata — se cambi anche `VITE_TURNI_API_URL` in `web/.env` di conseguenza):

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/turni
```

Postgres è raggiungibile da `localhost` (utile per ispezionarlo con un client come DBeaver) su `POSTGRES_HOST_PORT` (default `5432`, cambialo in `.env` se hai già un Postgres locale su quella porta): host `localhost`, database/utente/password come impostati in `.env`.

Documentazione API interattiva (Swagger UI): `http://localhost:8080/docs/` (stessa porta di cui sopra).

Per fermare tutto:

```bash
docker compose down
```

## Eseguire il binario Go direttamente (senza Docker per il servizio)

Utile per iterare rapidamente sul codice senza ricostruire l'immagine ad ogni modifica. Serve comunque un Postgres raggiungibile da `localhost`: avvialo come container standalone (non tramite `deploy/docker-compose.yml`, che espone Postgres solo sulla rete interna dei container) e applica lo schema:

```bash
docker run -d --name turni-postgres-dev -p 5432:5432 \
  -e POSTGRES_USER=sanitas -e POSTGRES_PASSWORD=devlocalpassword -e POSTGRES_DB=sanitas \
  -v "$(pwd)/migrations:/docker-entrypoint-initdb.d:ro" \
  postgres:16-alpine

cp .env.example .env   # valori già coerenti col container Postgres sopra; PORT=8081 se anche la 8080 sull'host è occupata
go run ./cmd/server
```

Il binario carica `.env` da solo se presente nella directory da cui viene lanciato (comodo per `go run`, ignorato in Docker/produzione dove le variabili vere sono già impostate) — non serve fare `source` a mano.

Per fermare/rimuovere il container di prova: `docker rm -f turni-postgres-dev`.

## Comandi di verifica (usati anche in CI)

```bash
go build ./...
go vet ./...
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
gofmt -l .   # deve stampare niente
```

## Rigenerare la documentazione Swagger

La spec in `api/` (`docs.go`, `swagger.json`, `swagger.yaml`) è **generata** dalle annotazioni sopra gli handler in `internal/httpapi/server.go` (vedi [ADR-0009](../../docs/adr/0009-swagger-generato-dal-codice.md)) — non va mai modificata a mano. Dopo aver aggiunto o cambiato un endpoint:

```bash
go install github.com/swaggo/swag/cmd/swag@latest   # una tantum
swag init -g cmd/server/main.go -d . -o api --parseInternal --parseDependency
```

Se dimentichi questo passo, la CI fallisce (rigenera e confronta con quanto committato in `api/`).

## Variabili d'ambiente

| Variabile             | Default                  | Descrizione                                                    |
|-----------------------|---------------------------|------------------------------------------------------------------|
| `PORT`                | `8080`                    | Porta HTTP del servizio                                          |
| `DATABASE_URL`        | *(obbligatoria)*          | DSN Postgres, es. `postgres://user:pass@host:5432/db?sslmode=disable` |
| `CORS_ALLOWED_ORIGIN` | `http://localhost:5173`  | Origine consentita per le richieste del frontend                 |
