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

## Altri microservizi (non ancora iniziati)

- [ ] `anagrafica` — anagrafica volontari/soci
- [ ] `mezzi-magazzino` — gestione mezzi e magazzino
- [ ] `servizi-emergenze` — servizi ed emergenze

Ognuno replicherà, una volta definite, le stesse convenzioni (commenti, Swagger, linee guida Go, test) stabilite per `turni` — da tenere sincronizzate.

## Infrastruttura / cross-cutting

- [ ] Setup fisico del Raspberry Pi: Docker, token del tunnel Cloudflare, registrazione del GitHub Actions self-hosted runner (vedi [`deploy/README.md`](../deploy/README.md)) — attività dell'utente, fuori da Claude Code.
- [ ] Confermare la visibilità del repository GitHub (assunto pubblico, coerente con licenza MIT e obiettivo di forkabilità) e verificare il push.
- [ ] Strategia di backup del database (es. `pg_dump` periodico verso storage esterno) — vedi [ADR-0004](adr/0004-target-di-deploy.md) e [ADR-0005](adr/0005-database-postgres-self-hosted.md).
- [ ] Creare la struttura `config/<slug>/` per gli override per-associazione (branding, dati, endpoint) — prevista in `CLAUDE.md`, non ancora creata.

## Processo

- [x] Definire dove tracciare le attività e le decisioni (questo file + `docs/adr/`).
- [ ] Tenere `docs/backlog.md` e `docs/adr/` aggiornati ad ogni sessione rilevante — responsabilità condivisa umano/Claude, richiamata in `CLAUDE.md`.
