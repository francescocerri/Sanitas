# 0011. Test automatici: integration test con testcontainers-go

Status: Accettata

## Contesto

Il servizio `turni` non aveva alcun test. Il codice è quasi tutta orchestrazione HTTP↔DB, non logica isolabile: il livello che conta è l'integrazione contro un Postgres reale (query SQL, cast `id::text`, traduzione di `pgx.ErrNoRows` in `turno.ErrNotFound`, status code HTTP), non gli unit test su funzioni pure.

## Decisione

**`testcontainers-go`** (+ modulo `.../modules/postgres`) per avviare un Postgres usa-e-getta nei test, sia in locale che in CI — `go test ./...` funziona da solo, senza setup manuale.

- `internal/testdb/testdb.go`: helper condiviso (non un file `_test.go`) che avvia il container, applica lo stesso `migrations/0001_init.sql` già usato da `deploy/docker-compose.yml` (via `postgres.WithInitScripts`), e ritorna un pool pronto + funzione di cleanup.
- Ogni package di test (`internal/turno`, `internal/httpapi`) ha il proprio `TestMain` che chiama questo helper una volta sola per l'intero package (un container condiviso tra i test di quel package, non uno per test) — ogni test tronca la tabella `turni` a fine esecuzione per non sporcare gli altri.
- `internal/httpapi`: i test collegano un `turno.Repository` reale al Postgres di test, nessuna interfaccia/mock introdotta — coerente con [ADR-0010](0010-convenzioni-cross-cutting-servizi-go.md).
- **Requisito da qui in avanti**: ogni nuovo endpoint o modifica a un endpoint esistente richiede i test di integrazione corrispondenti prima del merge — vale per `turni` e per ogni servizio futuro (`anagrafica`, `mezzi-magazzino`, `servizi-emergenze`), stesso pattern da replicare identico.

## Conseguenze

- Nuova dipendenza Go reale ma **solo di test**: nessun pacchetto di produzione (`cmd/server`, `internal/httpapi`, `internal/turno`, `internal/config`) importa `internal/testdb` o `testcontainers-go` — verificato che `go build ./cmd/server` non li compila e il binario finale non li referenzia.
- `govulncheck ./...` analizza anche `internal/testdb` (è comunque codice del repository): due CVE reali in dipendenze transitive di testcontainers-go sono state trovate e risolte durante questa attività (`golang.org/x/crypto` → v0.56.0, `github.com/moby/go-archive` → v0.3.0) — a differenza di altre dipendenze indirette già viste in questo repo, qui il codice le richiama davvero (tramite `postgres.Run`), quindi non erano ignorabili.
- CI leggermente più lenta (qualche secondo per avviare i container Postgres ad ogni run di `go test`) — accettabile per la garanzia che dà.
- La regola "test per ogni nuovo/modificato endpoint" non è imposta meccanicamente da CI (nessun check di coverage minima) — resta responsabilità di code review, come da convenzione in `CLAUDE.md`.
