# 0014. Database Postgres condiviso tra servizi, uno schema per servizio

Status: Accettata

## Contesto

`registry` (PR #6) era partito con un proprio container Postgres separato, seguendo lo stesso pattern "un database per servizio" di [ADR-0005](0005-database-postgres-self-hosted.md). `shifts` deve però referenziare gli utenti di `registry` (`volontario_id`) con una relazione reale, non solo un identificativo testuale scollegato — e mantenere due Postgres separati renderebbe impossibile una FK a livello di database tra le due tabelle.

## Decisione

**Un solo container Postgres, un solo database (`sanitas`), condiviso da tutti i servizi.** Ogni servizio resta isolato al suo interno tramite uno **schema Postgres nativo** (namespace), non un database separato: `registry.*`, `shifts.*`. Ogni servizio continua a possedere ed eseguire la propria migrazione (`services/<nome>/migrations/0001_init.sql`), che ora inizia con `CREATE SCHEMA IF NOT EXISTS <nome>; SET search_path TO <nome>;` — nessuno schema condiviso o file di migrazione comune da mantenere.

`shifts.shifts.volontario_id` è ora `UUID NOT NULL REFERENCES registry.users(id)` — una FK reale, non un placeholder testuale: Postgres rifiuta l'inserimento di un turno con un volontario inesistente.

**Ordine di applicazione**: poiché la FK di `shifts` dipende dalla tabella `registry.users`, lo schema `registry` deve esistere prima. In `deploy/docker-compose.yml` i due script vengono montati in `/docker-entrypoint-initdb.d/` con nomi che impongono l'ordine (`01-anagrafica-init.sql`, `02-turni-init.sql` — Postgres esegue gli script in ordine lessicografico sul path nel container, non sul nome del file sorgente).

**Connessione**: ogni servizio si connette allo stesso `DATABASE_URL` (stesso host/database), con un parametro `search_path` diverso (`search_path=shifts` o `search_path=registry`) nella connection string — così le query esistenti, scritte senza prefisso di schema (`FROM shifts`, `FROM users`), continuano a funzionare invariate.

## Conseguenze

- **Indipendenza dei servizi ridotta rispetto ad ADR-0005/PR #6**: i test di integrazione di `shifts` (testcontainers-go) devono ora applicare anche la migrazione di `registry` nel proprio container Postgres usa-e-getta, prima della propria (`postgres.WithOrderedInitScripts`, non `WithInitScripts` — i due file si chiamano entrambi `0001_init.sql`, quindi servono nomi distinti nel container). Un trade-off esplicito, accettato in cambio dell'integrità referenziale reale.
- `shifts` legge ora `registry.users(id)` a livello di schema DB (vincolo FK), ma nessuna query applicativa cross-schema è stata introdotta in questa fase (niente join per mostrare il nome del volontario — resta fuori scope, richiede prima l'adozione di un ORM/query layer, vedi voce "Fase 2: GORM" in `docs/backlog.md`).
- Un solo volume/container Postgres da backuppare, avviare e monitorare in produzione, invece di uno per servizio — coerente con l'obiettivo di costo/operatività minima (vedi CLAUDE.md, "Target di deploy e costo").
- Se in futuro un comitato forka il progetto e vuole isolare completamente i dati tra servizi (es. per requisiti di compliance più stringenti), questa decisione va rivista — non è il caso d'uso principale di oggi.
