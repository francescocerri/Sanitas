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
services/<nome-servizio>/   # turni, anagrafica (implementati); mezzi-magazzino, servizi-emergenze da fare
                             # ciascuno Go module a sé stante, con cmd/server, internal/, api/, Dockerfile
                             # api/ = spec Swagger generata dalle annotazioni sopra gli handler (swaggo/swag), servita su /docs/
                             # schema DB: anagrafica via GORM AutoMigrate, turni via SQL incorporato (internal/schema) — vedi ADR-0019
web/                         # app React (Vite + TS)
config/<committee-slug>/    # override per-comitato (es. config/pavullo/anagrafica/roles.json) — creata via via che servono
docs/
  backlog.md                # attività pianificate ma non ancora fatte — fonte di verità sul "cosa manca"
  adr/                       # una decisione architetturale per file, con contesto e conseguenze
  funzionale/                # cosa si può fare oggi sull'applicativo, per area, dal punto di vista di chi lo usa
  deploy-e-fork/             # percorso completo per far partire il progetto da zero (fork, segreti, deploy, verifica)
deploy/                     # docker-compose.yml, .env.example, README setup Raspberry Pi
.github/workflows/          # ci.yml (build/test/vet/govulncheck/npm audit), deploy.yml (self-hosted runner)
```

## Target di deploy e costo

Obiettivo: costo di esercizio molto basso, tendente a zero.

- **Produzione oggi**: Docker Compose su Raspberry Pi di casa, esposto via **Cloudflare Tunnel** (`cloudflared`) — niente porte aperte sul router, niente IP dinamico da gestire, TLS gratuito. Il deploy continuo gira su un **GitHub Actions self-hosted runner installato sul Raspberry Pi stesso**: la CI (build/test/scan) gira su runner GitHub-hosted gratuiti, il job di deploy gira sul runner self-hosted → build Docker nativa ARM, nessuna cross-compilazione, nessun bisogno di esporre SSH verso casa.
- **Database**: PostgreSQL self-hosted, un solo container nel `docker-compose` condiviso da tutti i servizi (un database, `sanitas`), con uno **schema Postgres nativo per servizio** (`anagrafica`, `turni`, ...) — non un container/database per servizio. Ogni servizio resta proprietario del proprio schema, creato dal proprio binario al primo avvio — non più da script SQL in `/docker-entrypoint-initdb.d/`: `anagrafica` usa **GORM AutoMigrate** (vedi [ADR-0019](docs/adr/0019-gorm-e-automigrate.md)), `turni` esegue il proprio SQL incorporato nel binario (`internal/schema`), con retry perché la sua FK dipende dallo schema di `anagrafica` già esistente. FK reali tra schemi di servizi diversi sono ammesse quando esprimono un vincolo di dominio reale (es. `turni.turni.volontario_id → anagrafica.users.id`). Vedi [ADR-0014](docs/adr/0014-database-condiviso-schema-separati.md).
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

Tre tipi di documentazione, tenuti sempre aggiornati (non scritti una volta e dimenticati):

- **Tecnica/decisionale**: questo file + [`docs/adr/`](docs/adr/) — architettura, vincoli, il "perché" delle scelte.
- **Funzionale**: [`docs/funzionale/`](docs/funzionale/) — cosa si può fare oggi sull'applicativo, un file per area, dal punto di vista di chi lo usa (non come è costruito).
- **Deploy e fork**: [`docs/deploy-e-fork/`](docs/deploy-e-fork/) — tutto il percorso per far partire il progetto da zero (fork, segreti da generare, deploy, verifica), pensato per chi forka.

- **Attività pianificate**: vivono in [`docs/backlog.md`](docs/backlog.md), non nel codice o solo in conversazione — prima di scrivere codice per una feature non banale, l'attività deve essere in backlog (o aggiunta lì contestualmente). Niente "vibe coding": si definiscono le attività, poi si passa al codice.
- **Decisioni architetturali**: ogni decisione rilevante (nuova o che ne cambia una precedente) va registrata come ADR in [`docs/adr/`](docs/adr/) — un file per decisione, con contesto e conseguenze. Le decisioni passate non si riscrivono: se cambiano, si aggiunge un nuovo ADR che supera il precedente.
- **Codice commentato**: i commenti spiegano il *perché* di scelte non ovvie (vincoli, workaround, trade-off), non ripetono cosa fa già dire il codice stesso.
- **Lingua**: codice, commenti (incluse le annotazioni Swagger), log strutturati (`log/slog` o equivalente) **e ora anche i messaggi nel body delle risposte HTTP** (es. `{"error": "invalid credentials"}`) sono in inglese — convenzione standard per codice destinato a essere letto/consultato anche da chi non parla italiano. Vedi [ADR-0015](docs/adr/0015-messaggi-errore-in-inglese.md) (supera la parte di [ADR-0010](docs/adr/0010-convenzioni-cross-cutting-servizi-go.md) che teneva questi messaggi in italiano). I segmenti del path delle route HTTP (es. `/v1/shifts`, `/v1/users`) sono anch'essi in inglese — vedi ADR-0010. **Restano invariati in italiano**, perché non sono messaggi ma dati/contratto: i nomi dei campi JSON nel body di richieste/risposte (es. `volontario_id`, `email`, `username`) e i valori di vocabolario di dominio (es. `stato: "pianificato"`) — un'inconsistenza nota e intenzionalmente aperta, non ancora estesa alla convenzione.
- **API documentate con Swagger generato dal codice**: annotazioni `swaggo/swag` sopra ogni handler, mai una spec scritta e mantenuta a mano — vedi [ADR-0009](docs/adr/0009-swagger-generato-dal-codice.md). Un check in CI garantisce che la spec generata combaci con quella committata.

## Convenzioni di lavoro con Claude Code

- Usare **Plan Mode** prima di iniziare ogni nuovo microservizio o feature non banale.
- Passare da `/code-review` prima di ogni merge.
- Aggiornare questo file ogni volta che cambia una decisione architetturale rilevante (in aggiunta all'ADR dedicato, se la decisione è abbastanza importante da meritarne uno).
- Commit: Conventional Commits. Branching: feature branch + PR.
- **Mai committare senza l'ok esplicito dell'utente**: preparare e verificare le modifiche, poi chiedere conferma prima di ogni `git commit` — non solo prima del push/PR.
- **Test**: integration test con `testcontainers-go` (Postgres usa-e-getta, `go test ./...` senza setup manuale) — vedi [ADR-0011](docs/adr/0011-test-automatici-testcontainers.md). **Ogni nuovo endpoint o modifica a un endpoint esistente richiede i test corrispondenti prima del merge**, su tutti i servizi.
- Seguire le linee guida standard per servizi Go a microservizi: logging strutturato (`log/slog`), error handling con wrapping ed errori di dominio, timeout su `http.Server` — vedi [ADR-0010](docs/adr/0010-convenzioni-cross-cutting-servizi-go.md). Struttura a layer non introdotta finché il dominio reale non è progettato (stessa ADR). Da replicare identico in ogni nuovo servizio.
- **Prima di ogni `git push`/apertura o aggiornamento di una PR**: rieseguire in locale l'intera suite di verifica (`go build`, `go vet`, `go test`, `gofmt -l`, `govulncheck`, rigenerazione Swagger con `swag init` e controllo che non ci sia drift) — non fidarsi solo della CI remota, verificarlo anche in locale prima del push.

## Nota per chi fa il fork

1. Fork del repository.
2. Creare `config/<nuovo-comitato>/` con branding e dati del proprio comitato.
3. Personalizzare eventuali override necessari.
4. Deploy indipendente (dettagli tecnici da definire quando si disegnerà `deploy/`).

## Licenza

MIT (vedi [LICENSE](LICENSE)). Permissiva: chi forka non è obbligato a ridare indietro le modifiche.
