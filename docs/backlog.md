# Backlog

Elenco vivo delle attività da fare. Si aggiorna ad ogni sessione: si spuntano le voci fatte, se ne aggiungono di nuove, si riformulano quelle che si scoprono sbagliate. Le decisioni già prese (il "perché" di scelte architetturali) stanno negli [ADR](adr/), non qui.

Quando un'attività è "una feature non banale" (per la convenzione in `CLAUDE.md`), va pianificata con Plan Mode prima di scrivere codice.

## Servizio `shifts`

- [x] **Commentare il codice esistente** (`services/shifts`): commenti sul *perché* delle scelte non ovvie (cast `id::text`, slice vuota vs nil per il JSON, CORS via env, timeout su `http.Server`, ecc.) in tutti i file. Log strutturati e commenti in inglese, messaggi rivolti al client/API restano in italiano (vedi `CLAUDE.md`).
- [x] **Documentazione API via Swagger**: generata dalle annotazioni sopra gli handler (`swaggo/swag`), consultabile su `/docs/` — un check in CI fallisce se qualcuno dimentica di rigenerarla dopo un cambio di endpoint. Vedi [ADR-0009](adr/0009-swagger-generato-dal-codice.md) (supera [ADR-0008](adr/0008-documentazione-api-openapi.md)).
- [x] **Allineamento alle linee guida standard per microservizi Go**: logging strutturato (`log/slog` + middleware di access log/panic recovery), error handling con wrapping ed errori di dominio (`shift.ErrNotFound`), timeout completi su `http.Server`. Vedi [ADR-0010](adr/0010-convenzioni-cross-cutting-servizi-go.md). Struttura a layer **non** introdotta ora, deliberatamente (vedi ADR) — da rivalutare con la progettazione del dominio reale.
- [x] **Test automatici**: integration test con `testcontainers-go` (Postgres usa-e-getta, nessun setup manuale) per `internal/shift` e `internal/httpapi`. Vedi [ADR-0011](adr/0011-test-automatici-testcontainers.md). Da qui in avanti requisito per ogni nuovo endpoint o modifica a un endpoint esistente, su tutti i servizi.
- [ ] **Progettazione del dominio reale** (assegnazione turni, conflitti, disponibilità volontari) — il modello attuale è volutamente scheletrico (vedi [ADR-0005](adr/0005-database-postgres-self-hosted.md)). Richiede una sessione di Plan Mode dedicata.
- [ ] **Introdurre uno strumento di migrazioni** quando lo schema dovrà evolvere su dati reali (non ancora necessario, vedi [ADR-0005](adr/0005-database-postgres-self-hosted.md)).

## Servizio `registry`

- [x] **Fase A**: scaffold del servizio, schema (utenti, ruoli, ruoli-utente, token di invito), login, creazione utenti da admin + attivazione via token, cambio password. Ruoli come dati seed per-comitato (vedi [ADR-0012](adr/0012-ruoli-come-dati-seed.md)), autenticazione bcrypt + JWT RS256 con JWKS (vedi [ADR-0013](adr/0013-autenticazione-bcrypt-jwt.md)). Stesse convenzioni cross-cutting/test di `shifts` (ADR-0009/0010/0011).
- [ ] **Fase B**: invio email reale (inviti + forgot-password) via Gmail SMTP — oggi l'URL di invito è solo restituito nella risposta API, non spedito.
- [x] **Permessi granulari per ruolo**: `is_admin` rimosso, sostituito da permessi (`users:manage`, `shifts:read`, `shifts:write`) assegnati per ruolo via `config/<slug>/registry/roles.json`, applicati sia in `registry` che in `shifts`. Vedi [ADR-0018](adr/0018-permessi-granulari-per-ruolo.md).
- [ ] **Filtrare i dati per ruolo** (es. un volontario vede solo i propri turni) — fuori scope di ADR-0018: richiede prima progettare come `shifts` rappresenta un "proprietario" della vista (vedi anche "Progettazione del dominio reale" più sopra). Lato FE, applicare la stessa granularità dei permessi già emessi nel token.
- [x] Integrazione `shifts` ↔ `registry`: `volunteer_id` è ora una FK reale verso `registry.users(id)`, database condiviso con schema separati per servizio (vedi [ADR-0014](adr/0014-database-condiviso-schema-separati.md)).
- [x] **Refresh token**: `POST /v1/login` ora restituisce anche un refresh token (30gg, rotante), più `POST /v1/refresh` e `POST /v1/logout` — riusa la tabella `tokens` (rinominata da `invite_tokens`, ora generica) esistente. Vedi [ADR-0016](adr/0016-refresh-token.md). Nessun rilevamento di riuso/furto (token family) né revoca dell'access token già emesso — limiti noti, non affrontati.
- [ ] **Pulizia dei token scaduti/usati**: righe di `tokens` (invite, refresh, in futuro password-reset) restano nel DB a tempo indeterminato dopo `used_at`/`expires_at` — introdurre un job periodico (o una query di cleanup all'avvio) che le cancella. Non compromette la correttezza oggi, solo la crescita nel tempo della tabella.
- [x] **`shifts` verifica i JWT**: tutti gli endpoint `/v1/shifts*` richiedono un token valido, verificato localmente contro la JWKS di `registry` (nessuna chiamata sincrona per ogni richiesta). Vedi [ADR-0017](adr/0017-shifts-verifica-jwt.md). Solo autenticazione — l'autorizzazione granulare per ruolo resta la voce già separata qui sotto.

## Frontend (`app/`, Flutter)

- [ ] **Scaffold del progetto Flutter** (`app/`, sostituisce il precedente scaffold React in `web/`): routing (`go_router`), stato/sessione (`flutter_riverpod`), client HTTP con interceptor di refresh (`dio`), i18n italiano/inglese (`easy_localization`), tema per-comitato caricato da `config/<slug>/app/theme.json`. Vedi [ADR-0022](adr/0022-frontend-flutter.md).
- [ ] **Login**: form identifier+password contro `POST /v1/login` di `registry`, gestione errori localizzata, sessione con access token in memoria + refresh token in `flutter_secure_storage`.
- [ ] **Attivazione account** da link di invito (`POST /v1/users/activate`) — oggi il link lo inoltra a mano un admin (vedi `docs/funzionale/registry.md`), serve solo la schermata che lo consuma.
- [ ] **Profilo self-service**: `GET /v1/me`, cambio password (`POST /v1/password/change`), logout (`POST /v1/logout`).
- [ ] **Refresh automatico**: interceptor `dio` su 401 + refresh proattivo allo startup dell'app.
- [ ] **Test**: unit test su decode JWT e sul controller di sessione, widget test minimi sulle 3 schermate.
- [ ] **Pannello admin utenti/ruoli** (creazione utenti, assegnazione ruoli, lista utenti) — fuori perimetro della prima iterazione, richiede anche un nuovo endpoint backend (`GET /v1/roles`, oggi non esiste) per popolare un selettore ruoli. Da pianificare a parte.
- [ ] **Strategia di hosting/distribuzione**: come si serve la build web in produzione (oggi `deploy/docker-compose.yml` non ha alcun servizio frontend) e come si arriva a una pubblicazione reale su App Store/Google Play (account sviluppatore, firma, CI di release) — non affrontato nella prima iterazione.

## Database / ORM

- [x] **Fase 1**: consolidamento su un unico Postgres condiviso tra `shifts` e `registry`, uno schema per servizio, FK reale `shifts.shifts.volunteer_id → registry.users.id` (vedi [ADR-0014](adr/0014-database-condiviso-schema-separati.md)). `pgx` invariato in entrambi i servizi.
- [x] **Fase 2 completa**: entrambi i servizi su **GORM** (query builder) + **AutoMigrate** al posto degli script SQL. `registry` per prima (con la risoluzione dell'ordine di creazione schema con `shifts`, la cui FK dipende da `registry.users`), poi `shifts` (stesso pattern, dominio più piccolo). Vedi [ADR-0019](adr/0019-gorm-e-automigrate.md) e [ADR-0020](adr/0020-shifts-gorm.md) (superano parte di [ADR-0005](adr/0005-database-postgres-self-hosted.md)).

## Altri microservizi (non ancora iniziati)

- [ ] `mezzi-magazzino` — gestione mezzi e magazzino
- [ ] `servizi-emergenze` — servizi ed emergenze

Ognuno replicherà le stesse convenzioni (commenti, Swagger, linee guida Go, test) stabilite per `shifts`/`registry` — da tenere sincronizzate.

## Infrastruttura / cross-cutting

- [ ] Setup fisico del Raspberry Pi: Docker, token del tunnel Cloudflare, registrazione del GitHub Actions self-hosted runner (vedi [`deploy/README.md`](../deploy/README.md)) — attività dell'utente, fuori da Claude Code.
- [ ] Confermare la visibilità del repository GitHub (assunto pubblico, coerente con licenza MIT e obiettivo di forkabilità) e verificare il push.
- [ ] Strategia di backup del database (es. `pg_dump` periodico verso storage esterno) — vedi [ADR-0004](adr/0004-target-di-deploy.md) e [ADR-0005](adr/0005-database-postgres-self-hosted.md).
- [x] Creare la struttura `config/<slug>/` per gli override per-associazione — fatto per `registry` (`config/pavullo/registry/roles.json`, vedi ADR-0012); altri override (branding, dati, endpoint) da aggiungere man mano che servono.

## Processo

- [x] Definire dove tracciare le attività e le decisioni (questo file + `docs/adr/`).
- [ ] Tenere `docs/backlog.md` e `docs/adr/` aggiornati ad ogni sessione rilevante — responsabilità condivisa umano/Claude, richiamata in `CLAUDE.md`.
