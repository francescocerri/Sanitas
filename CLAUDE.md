# CLAUDE.md

Questo file guida chiunque (umano o Claude) lavori su questo repository, incluso chi lo forka per un altro comitato. È la fonte di verità condivisa per architettura, vincoli e convenzioni: va aggiornato ogni volta che una decisione rilevante cambia.

## Overview

**Sanitas** è una piattaforma a microservizi per la gestione operativa di un'associazione di soccorso sanitario (turni, anagrafica volontari/soci, mezzi e magazzino, servizi ed emergenze). Sviluppata per il Comitato CRI di Pavullo, progettata esplicitamente per essere forkata e riusata da altre associazioni (CRI, Croce Verde, Croce Blu, ecc.).

Scope MVP (in ordine di priorità):
1. Gestione turni
2. Anagrafica volontari/soci
3. Gestione mezzi e magazzino
4. Servizi ed emergenze

## Contratto di forkabilità (regola non negoziabile)

**Nessun dato o identificativo specifico del Comitato di Pavullo deve mai finire hardcoded nel codice sorgente.** Nome del comitato, loghi, colori, contatti, indirizzi, elenco mezzi, utenti, qualunque dato reale del deployment: tutto vive in configurazione esterna al codice, mai in constant, default, o fixture di test che assomiglino a dati reali.

Convenzione di configurazione: `config/<committee-slug>/` contiene gli override per-comitato (branding, dati anagrafici del comitato, endpoint). Il codice dei servizi legge sempre da lì o da variabili d'ambiente, mai da valori inline.

Modello di fork: **fork + configurazione**, non multi-tenant single-deployment. Ogni comitato fa il fork del repo, personalizza `config/<nuovo-comitato>/`, branding e dati, e deploya una propria istanza indipendente. Non progettare per l'isolamento dati multi-tenant in un unico deployment condiviso — non è l'obiettivo di questo progetto.

Prima di scrivere codice per una nuova feature, chiediti: "questo assumerebbe qualcosa di specifico di Pavullo?" Se sì, va parametrizzato.

## Stack tecnico

- **Backend**: Go. Ogni microservizio è un modulo Go indipendente (`module github.com/francescocerri/sanitas/services/<nome>`).
- **Frontend**: React + TypeScript (Vite).

## Struttura repo

```
services/<nome-servizio>/   # es. turni (implementato), anagrafica, mezzi-magazzino, servizi-emergenze
                             # ciascuno Go module a sé stante, con cmd/server, internal/, api/, migrations/, Dockerfile
                             # api/ = spec Swagger generata dalle annotazioni sopra gli handler (swaggo/swag), servita su /docs/
web/                         # app React (Vite + TS)
config/<committee-slug>/    # override per-comitato (branding, dati, endpoint) — non ancora creata
docs/
  backlog.md                # attività pianificate ma non ancora fatte — fonte di verità sul "cosa manca"
  adr/                       # una decisione architetturale per file, con contesto e conseguenze
deploy/                     # docker-compose.yml, .env.example, README setup Raspberry Pi
.github/workflows/          # ci.yml (build/test/vet/govulncheck/npm audit), deploy.yml (self-hosted runner)
```

## Target di deploy e costo

Obiettivo: costo di esercizio molto basso, tendente a zero.

- **Produzione oggi**: Docker Compose su Raspberry Pi di casa, esposto via **Cloudflare Tunnel** (`cloudflared`) — niente porte aperte sul router, niente IP dinamico da gestire, TLS gratuito. Il deploy continuo gira su un **GitHub Actions self-hosted runner installato sul Raspberry Pi stesso**: la CI (build/test/scan) gira su runner GitHub-hosted gratuiti, il job di deploy gira sul runner self-hosted → build Docker nativa ARM, nessuna cross-compilazione, nessun bisogno di esporre SSH verso casa.
- **Database**: PostgreSQL self-hosted in un container nel `docker-compose`, dati su volume Docker. Schema applicato via script SQL montati in `/docker-entrypoint-initdb.d/` (feature nativa dell'immagine ufficiale Postgres) — nessun tool di migrazione finché non serve evolvere lo schema in modo incrementale su più release.
- **Portabilità**: tutto containerizzato e configurato via env var, per permettere una futura migrazione ad AWS (EC2/ECS) — l'utente è già skillato su AWS — se il comitato/l'associazione cresce e serve più affidabilità di quanta ne dia una connessione domestica. Il free tier AWS su EC2/RDS scade dopo 12 mesi: non è il target di partenza proprio per questo.
- Backup del database: non ancora implementato (nota aperta, da definire quando servirà davvero — es. `pg_dump` periodico verso storage esterno).

## Policy di sicurezza e dipendenze

**Ogni servizio deve usare librerie mantenute attivamente e prive di vulnerabilità note.** Prima di aggiungere una dipendenza esterna, valutare: manutenzione attiva (commit recenti, non archiviata), adozione/popolarità, storico CVE. Preferire sempre la standard library quando è sufficiente (es. routing HTTP dei servizi Go: `net/http` di libreria standard, nessun router esterno).

Dipendenze esterne già scelte e perché:
- `github.com/jackc/pgx/v5` — driver Postgres per Go, standard de facto, attivamente mantenuto (preferito a `lib/pq`, che è in sola manutenzione).

CI (obbligatoria prima del merge):
- Go: `go build`, `go vet`, `go test`, `govulncheck` (tool ufficiale del team Go) — nessuna vulnerabilità nota deve passare.
- Frontend: `npm run build`, `npm run lint`, `npm audit --audit-level=high`.

Aggiornamento dipendenze: **Renovate** (`renovate.json`, preset `config:recommended`) apre PR automatiche per Go modules, npm, immagini Docker base e versioni delle GitHub Actions. Le PR di sicurezza vanno prioritizzate rispetto alle altre.

## Documentazione

- **Attività pianificate**: vivono in [`docs/backlog.md`](docs/backlog.md), non nel codice o solo in conversazione — prima di scrivere codice per una feature non banale, l'attività deve essere in backlog (o aggiunta lì contestualmente). Niente "vibe coding": si definiscono le attività, poi si passa al codice.
- **Decisioni architetturali**: ogni decisione rilevante (nuova o che ne cambia una precedente) va registrata come ADR in [`docs/adr/`](docs/adr/) — un file per decisione, con contesto e conseguenze. Le decisioni passate non si riscrivono: se cambiano, si aggiunge un nuovo ADR che supera il precedente.
- **Codice commentato**: i commenti spiegano il *perché* di scelte non ovvie (vincoli, workaround, trade-off), non ripetono cosa fa già dire il codice stesso.
- **Lingua**: codice, commenti (incluse le annotazioni Swagger) e log strutturati (`log/slog` o equivalente) in inglese — convenzione standard per codice destinato a essere letto/consultato anche da chi non parla italiano. Solo i messaggi nel body delle risposte HTTP (rivolti a chi consuma l'API) restano in italiano, coerentemente con la documentazione di progetto (`CLAUDE.md`, ADR, backlog) e il vocabolario di dominio (`turno`, `volontario`, ecc.), che restano invariati.
- **API documentate con Swagger generato dal codice**: annotazioni `swaggo/swag` sopra ogni handler, mai una spec scritta e mantenuta a mano — vedi [ADR-0009](docs/adr/0009-swagger-generato-dal-codice.md). Un check in CI garantisce che la spec generata combaci con quella committata.

## Convenzioni di lavoro con Claude Code

- Usare **Plan Mode** prima di iniziare ogni nuovo microservizio o feature non banale.
- Passare da `/code-review` prima di ogni merge.
- Aggiornare questo file ogni volta che cambia una decisione architetturale rilevante (in aggiunta all'ADR dedicato, se la decisione è abbastanza importante da meritarne uno).
- Commit: Conventional Commits. Branching: feature branch + PR.
- Test: unit test per servizio Go; test di integrazione via docker-compose quando più servizi comunicheranno tra loro.
- Seguire le linee guida standard per servizi Go a microservizi (struttura a layer, logging strutturato, error handling consistente, ecc.) — le specifiche concrete si definiscono e si tracciano in `docs/backlog.md`/ADR mano a mano che si applicano, per evitare di riscrivere codice esistente "a sensazione".

## Nota per chi fa il fork

1. Fork del repository.
2. Creare `config/<nuovo-comitato>/` con branding e dati del proprio comitato.
3. Personalizzare eventuali override necessari.
4. Deploy indipendente (dettagli tecnici da definire quando si disegnerà `deploy/`).

## Licenza

MIT (vedi [LICENSE](LICENSE)). Permissiva: chi forka non è obbligato a ridare indietro le modifiche.
