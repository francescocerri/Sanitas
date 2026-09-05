# Backlog

Elenco vivo delle attività da fare. Si aggiorna ad ogni sessione: si spuntano le voci fatte, se ne aggiungono di nuove, si riformulano quelle che si scoprono sbagliate. Le decisioni già prese (il "perché" di scelte architetturali) stanno negli [ADR](adr/), non qui.

Quando un'attività è "una feature non banale" (per la convenzione in `CLAUDE.md`), va pianificata con Plan Mode prima di scrivere codice.

## Servizio `turni`

- [ ] **Commentare il codice esistente** (`services/turni`): spiegare il *perché* delle scelte non ovvie (es. cast `id::text` nelle query, CORS configurabile via env), non ripetere l'ovvio.
- [x] **Documentazione API via OpenAPI/Swagger**: spec in `services/turni/api/openapi.yaml`, servita su `/openapi.yaml` e consultabile su `/docs` (Swagger UI). Vedi [ADR-0008](adr/0008-documentazione-api-openapi.md). Da tenere aggiornata ad ogni cambio di endpoint (nessuna generazione automatica).
- [ ] **Allineamento alle linee guida standard per microservizi Go**: da definire insieme quali esattamente (candidati da discutere prima di implementare: logging strutturato con `log/slog`, error handling con errori tipizzati/wrapping consistente, struttura a layer espliciti, timeout/context propagation su tutte le chiamate esterne — quanto di questo è già presente va verificato, non riscritto a caso).
- [ ] **Test automatici**: unit test per la logica applicativa, integration test contro un Postgres reale (es. via `testcontainers-go` o docker-compose in CI).
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
