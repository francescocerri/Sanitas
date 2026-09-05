# Backlog

Elenco vivo delle attività da fare. Si aggiorna ad ogni sessione: si spuntano le voci fatte, se ne aggiungono di nuove, si riformulano quelle che si scoprono sbagliate. Le decisioni già prese (il "perché" di scelte architetturali) stanno negli [ADR](adr/), non qui.

Quando un'attività è "una feature non banale" (per la convenzione in `CLAUDE.md`), va pianificata con Plan Mode prima di scrivere codice.

## Servizio `turni`

- [x] **Commentare il codice esistente** (`services/turni`): commenti sul *perché* delle scelte non ovvie (cast `id::text`, slice vuota vs nil per il JSON, CORS via env, timeout su `http.Server`, ecc.) in tutti i file. Log strutturati e commenti in inglese, messaggi rivolti al client/API restano in italiano (vedi `CLAUDE.md`).
- [x] **Documentazione API via Swagger**: generata dalle annotazioni sopra gli handler (`swaggo/swag`), consultabile su `/docs/` — un check in CI fallisce se qualcuno dimentica di rigenerarla dopo un cambio di endpoint. Vedi [ADR-0009](adr/0009-swagger-generato-dal-codice.md) (supera [ADR-0008](adr/0008-documentazione-api-openapi.md)).
- [x] **Allineamento alle linee guida standard per microservizi Go**: logging strutturato (`log/slog` + middleware di access log/panic recovery), error handling con wrapping ed errori di dominio (`turno.ErrNotFound`), timeout completi su `http.Server`. Vedi [ADR-0010](adr/0010-convenzioni-cross-cutting-servizi-go.md). Struttura a layer **non** introdotta ora, deliberatamente (vedi ADR) — da rivalutare con la progettazione del dominio reale.
- [x] **Test automatici**: integration test con `testcontainers-go` (Postgres usa-e-getta, nessun setup manuale) per `internal/turno` e `internal/httpapi`. Vedi [ADR-0011](adr/0011-test-automatici-testcontainers.md). Da qui in avanti requisito per ogni nuovo endpoint o modifica a un endpoint esistente, su tutti i servizi.
- [ ] **Progettazione del dominio reale** (assegnazione turni, conflitti, disponibilità volontari) — il modello attuale è volutamente scheletrico (vedi [ADR-0005](adr/0005-database-postgres-self-hosted.md)). Richiede una sessione di Plan Mode dedicata.
- [ ] **Introdurre uno strumento di migrazioni** quando lo schema dovrà evolvere su dati reali (non ancora necessario, vedi [ADR-0005](adr/0005-database-postgres-self-hosted.md)).

## Servizio `anagrafica`

- [x] **Fase A**: scaffold del servizio, schema (utenti, ruoli, ruoli-utente, token di invito), login, creazione utenti da admin + attivazione via token, cambio password. Ruoli come dati seed per-comitato (vedi [ADR-0012](adr/0012-ruoli-come-dati-seed.md)), autenticazione bcrypt + JWT RS256 con JWKS (vedi [ADR-0013](adr/0013-autenticazione-bcrypt-jwt.md)). Stesse convenzioni cross-cutting/test di `turni` (ADR-0009/0010/0011).
- [ ] **Fase B**: invio email reale (inviti + forgot-password) via Gmail SMTP — oggi l'URL di invito è solo restituito nella risposta API, non spedito.
- [x] **Permessi granulari per ruolo**: `is_admin` rimosso, sostituito da permessi (`users:manage`, `shifts:read`, `shifts:write`) assegnati per ruolo via `config/<slug>/anagrafica/roles.json`, applicati sia in `anagrafica` che in `turni`. Vedi [ADR-0018](adr/0018-permessi-granulari-per-ruolo.md).
- [ ] **Filtrare i dati per ruolo** (es. un volontario vede solo i propri turni) — fuori scope di ADR-0018: richiede prima progettare come `turni` rappresenta un "proprietario" della vista (vedi anche "Progettazione del dominio reale" più sopra). Lato FE, applicare la stessa granularità dei permessi già emessi nel token.
- [x] Integrazione `turni` ↔ `anagrafica`: `volontario_id` è ora una FK reale verso `anagrafica.users(id)`, database condiviso con schema separati per servizio (vedi [ADR-0014](adr/0014-database-condiviso-schema-separati.md)).
- [x] **Refresh token**: `POST /v1/login` ora restituisce anche un refresh token (30gg, rotante), più `POST /v1/refresh` e `POST /v1/logout` — riusa la tabella `tokens` (rinominata da `invite_tokens`, ora generica) esistente. Vedi [ADR-0016](adr/0016-refresh-token.md). Nessun rilevamento di riuso/furto (token family) né revoca dell'access token già emesso — limiti noti, non affrontati.
- [ ] **Pulizia dei token scaduti/usati**: righe di `tokens` (invite, refresh, in futuro password-reset) restano nel DB a tempo indeterminato dopo `used_at`/`expires_at` — introdurre un job periodico (o una query di cleanup all'avvio) che le cancella. Non compromette la correttezza oggi, solo la crescita nel tempo della tabella.
- [x] **`turni` verifica i JWT**: tutti gli endpoint `/v1/shifts*` richiedono un token valido, verificato localmente contro la JWKS di `anagrafica` (nessuna chiamata sincrona per ogni richiesta). Vedi [ADR-0017](adr/0017-turni-verifica-jwt.md). Solo autenticazione — l'autorizzazione granulare per ruolo resta la voce già separata qui sotto.

## Database / ORM

- [x] **Fase 1**: consolidamento su un unico Postgres condiviso tra `turni` e `anagrafica`, uno schema per servizio, FK reale `turni.turni.volontario_id → anagrafica.users.id` (vedi [ADR-0014](adr/0014-database-condiviso-schema-separati.md)). `pgx` invariato in entrambi i servizi.
- [x] **Fase 2 completa**: entrambi i servizi su **GORM** (query builder) + **AutoMigrate** al posto degli script SQL. `anagrafica` per prima (con la risoluzione dell'ordine di creazione schema con `turni`, la cui FK dipende da `anagrafica.users`), poi `turni` (stesso pattern, dominio più piccolo). Vedi [ADR-0019](adr/0019-gorm-e-automigrate.md) e [ADR-0020](adr/0020-turni-gorm.md) (superano parte di [ADR-0005](adr/0005-database-postgres-self-hosted.md)).

## Altri microservizi (non ancora iniziati)

- [ ] `mezzi-magazzino` — gestione mezzi e magazzino
- [ ] `servizi-emergenze` — servizi ed emergenze

Ognuno replicherà le stesse convenzioni (commenti, Swagger, linee guida Go, test) stabilite per `turni`/`anagrafica` — da tenere sincronizzate.

## Infrastruttura / cross-cutting

- [ ] Setup fisico del Raspberry Pi: Docker, token del tunnel Cloudflare, registrazione del GitHub Actions self-hosted runner (vedi [`deploy/README.md`](../deploy/README.md)) — attività dell'utente, fuori da Claude Code.
- [ ] Confermare la visibilità del repository GitHub (assunto pubblico, coerente con licenza MIT e obiettivo di forkabilità) e verificare il push.
- [ ] Strategia di backup del database (es. `pg_dump` periodico verso storage esterno) — vedi [ADR-0004](adr/0004-target-di-deploy.md) e [ADR-0005](adr/0005-database-postgres-self-hosted.md).
- [x] Creare la struttura `config/<slug>/` per gli override per-associazione — fatto per `anagrafica` (`config/pavullo/anagrafica/roles.json`, vedi ADR-0012); altri override (branding, dati, endpoint) da aggiungere man mano che servono.

## Processo

- [x] Definire dove tracciare le attività e le decisioni (questo file + `docs/adr/`).
- [ ] Tenere `docs/backlog.md` e `docs/adr/` aggiornati ad ogni sessione rilevante — responsabilità condivisa umano/Claude, richiamata in `CLAUDE.md`.
