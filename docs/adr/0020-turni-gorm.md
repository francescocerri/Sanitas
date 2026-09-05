# 0020. `turni` adotta GORM, completando la Fase 2

Status: Accettata

## Contesto

[ADR-0019](0019-gorm-e-automigrate.md) ha portato `anagrafica` su GORM+AutoMigrate, lasciando esplicitamente `turni` su `pgx` con la creazione del proprio schema spostata al proprio avvio (SQL incorporato via `//go:embed`, con retry per la dipendenza dalla FK verso `anagrafica.users`). Questa attività completa la Fase 2 applicando a `turni` lo stesso pattern.

## Decisione

Stesso schema di ADR-0019, sul dominio di `turni` (un solo modello):

- `turno.Turno` è anche il modello GORM per la tabella `turni` (stessi campi, tag `gorm:` aggiunti) — nessun cambio al contratto JSON.
- `turno.Migrate(db)` (nuovo, condiviso tra produzione e test come `user.Migrate` in anagrafica): crea lo schema, `AutoMigrate` sul modello, poi aggiunge via SQL grezzo idempotente la FK verso `anagrafica.users(id)` — cross-schema **e cross-servizio** (un modulo Go diverso), quindi AutoMigrate non ha alcuna associazione da cui derivarla, esattamente come le FK di `anagrafica` verso se stessa in ADR-0019.
- Il pacchetto `internal/schema` (SQL incorporato) è rimosso, sostituito dai tag sul modello.
- Il retry allo startup (`createSchemaWithRetry` in `cmd/server/main.go`) resta invariato nella forma — richiama `turno.Migrate` invece di eseguire lo script SQL.
- `Repository.Create` azzera esplicitamente `id`/`stato` prima di chiamare GORM: con l'INSERT grezzo di prima questi campi non venivano mai scritti (solo `volontario_id`/`data`/`ora_inizio`/`ora_fine`), quindi un client che li passasse veniva silenziosamente ignorato — con `db.Create(&t)` di GORM, un valore non-zero in quei campi verrebbe invece inserito. Comportamento preservato esplicitamente, non un effetto collaterale.

## Conseguenze

- Fase 2 completa: entrambi i servizi su GORM+AutoMigrate, nessuno dei due usa più script SQL applicati da `docker-entrypoint-initdb.d` o incorporati nel binario.
- Stesso ciclo di import da rompere già visto in anagrafica: i test di `internal/turno` (package `turno`) importano `internal/testdb`, quindi `testdb` non può importare `turno` — `testdb.StartPostgres` accetta `migrate func(*gorm.DB) error` come parametro.
- Nessun cambio al contratto pubblico del repository né alle API esposte.
