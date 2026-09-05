# anagrafica

Microservizio Go per utenti, ruoli e autenticazione. Fase A (vedi [docs/adr/0013](../../docs/adr/0013-autenticazione-bcrypt-jwt.md)): creazione utenti da amministratore, attivazione via token, login, cambio password. L'invio email reale (inviti/forgot-password) è una fase successiva — l'URL di attivazione viene restituito direttamente nella risposta di `POST /v1/utenti`.

## Setup una tantum: chiave JWT

Il servizio firma i JWT con una coppia di chiavi RSA (RS256) — solo `anagrafica` detiene la chiave privata, chi deve verificare i token usa la chiave pubblica esposta su `GET /.well-known/jwks.json`. Generarla una volta (non va mai committata):

```bash
openssl genrsa -out jwt_private_key.pem 2048
```

## Eseguire in locale (con Postgres via Docker Compose)

```bash
cd ../../deploy
cp .env.example .env   # imposta almeno le password Postgres
docker compose up -d --build anagrafica-postgres anagrafica
```

```bash
curl http://localhost:8090/healthz
curl http://localhost:8090/.well-known/jwks.json
```

Documentazione API interattiva (Swagger UI): `http://localhost:8090/docs/`.

## Eseguire il binario Go direttamente (senza Docker per il servizio)

```bash
docker run -d --name anagrafica-postgres-dev -p 5433:5432 \
  -e POSTGRES_USER=anagrafica -e POSTGRES_PASSWORD=devlocalpassword -e POSTGRES_DB=anagrafica \
  -v "$(pwd)/migrations:/docker-entrypoint-initdb.d:ro" \
  postgres:16-alpine

openssl genrsa -out jwt_private_key.pem 2048
cp .env.example .env   # valori già coerenti col container sopra
go run ./cmd/server
```

Il binario carica `.env` da solo se presente (vedi `internal/config`) — non serve fare `source` a mano.

Al primo avvio, se la tabella `users` è vuota, viene creato l'amministratore da `INITIAL_ADMIN_EMAIL`/`INITIAL_ADMIN_USERNAME`/`INITIAL_ADMIN_PASSWORD`.

## Comandi di verifica (usati anche in CI)

```bash
go build ./...
go vet ./...
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
gofmt -l .   # deve stampare niente
```

## Rigenerare la documentazione Swagger

Come `services/turni` (vedi [ADR-0009](../../docs/adr/0009-swagger-generato-dal-codice.md)): la spec in `api/` è generata dalle annotazioni sopra gli handler, mai modificata a mano.

```bash
go install github.com/swaggo/swag/cmd/swag@latest   # una tantum
swag init -g cmd/server/main.go -d . -o api --parseInternal --parseDependency
```

## Variabili d'ambiente

| Variabile                 | Default                 | Descrizione                                                       |
|---------------------------|--------------------------|---------------------------------------------------------------------|
| `PORT`                    | `8080`                   | Porta HTTP del servizio                                             |
| `DATABASE_URL`            | *(obbligatoria)*         | DSN Postgres                                                        |
| `CORS_ALLOWED_ORIGIN`     | `http://localhost:5173` | Origine consentita per le richieste del frontend                    |
| `JWT_PRIVATE_KEY_PATH`    | *(obbligatoria)*         | Path della chiave privata RSA per firmare i JWT                      |
| `ROLES_SEED_PATH`         | *(nessuno)*              | File JSON dei ruoli da applicare all'avvio (vedi `config/<slug>/anagrafica/roles.json`) |
| `INVITE_URL_BASE`         | `http://localhost:5173/attiva` | Base dell'URL di attivazione restituito da `POST /v1/utenti`  |
| `INITIAL_ADMIN_EMAIL`     | *(nessuno)*              | Email del primo admin, usata solo se `users` è vuota                |
| `INITIAL_ADMIN_USERNAME`  | *(nessuno)*              | Username del primo admin                                            |
| `INITIAL_ADMIN_PASSWORD`  | *(nessuno)*              | Password del primo admin                                            |
