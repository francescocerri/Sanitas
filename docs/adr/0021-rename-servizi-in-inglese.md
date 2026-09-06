# 0021. Rename dei servizi in inglese: `turni`→`shifts`, `anagrafica`→`registry`

Status: Accettata

## Contesto

Rotte, messaggi di errore, log e commenti erano già stati portati in inglese in sessioni precedenti ([ADR-0010](0010-convenzioni-cross-cutting-servizi-go.md), [ADR-0015](0015-messaggi-errore-in-inglese.md)), lasciando però un'eccezione esplicita: il nome stesso dei due servizi (`turni`, `anagrafica`) e, di conseguenza, i nomi dei campi JSON del dominio `turni` (`volontario_id`, `data`, `ora_inizio`, `ora_fine`, `stato`), rimasti in italiano e documentati come "inconsistenza nota, lasciata aperta intenzionalmente" (ADR-0010).

Su richiesta esplicita, il perimetro è stato esteso a tutto: non solo tabella e colonne di `turni`, ma anche i nomi dei due servizi stessi, chiudendo l'ultima eccezione rimasta alla convenzione.

## Decisione

**Tutto in inglese**: `turni` → `shifts`, `anagrafica` → `registry`. Il rename copre ogni livello, non solo la superficie:

- Directory dei moduli Go, module path (`go.mod`), pacchetto interno (`internal/turno` → `internal/shift`, tipo `Turno` → `Shift`); `internal/user` in registry non cambia, era già inglese.
- Contratto JSON di `shifts`: `volontario_id`→`volunteer_id`, `data`→`date`, `ora_inizio`→`start_time`, `ora_fine`→`end_time`, `stato`→`status`, valore di default `"pianificato"`→`"planned"`.
- Schema e tabelle Postgres (nativi per servizio, [ADR-0014](0014-database-condiviso-schema-separati.md)): schema `turni`→`shifts`, `anagrafica`→`registry`; tabella `turni.turni`→`shifts.shifts`; FK cross-schema aggiornata (`shifts.shifts.volunteer_id → registry.users.id`).
- Nomi dei servizi Docker Compose, hostname interni, variabili d'ambiente (`TURNI_HOST_PORT`→`SHIFTS_HOST_PORT`, `ANAGRAFICA_HOST_PORT`→`REGISTRY_HOST_PORT`, `VITE_TURNI_API_URL`→`VITE_SHIFTS_API_URL`, ecc.), nome dei binari nei Dockerfile.
- `config/pavullo/anagrafica/` → `config/pavullo/registry/` (contenuto di `roles.json` invariato).
- Job CI (`.github/workflows/ci.yml`), Swagger rigenerato in entrambi i servizi (i `$ref` generati incorporano il module path Go completo).
- Documentazione: `CLAUDE.md`, ADR esistenti (riferimenti al servizio aggiornati dove aiutano la navigabilità; il vocabolario di dominio italiano nella prosa e i filename storici che documentano uno stato passato — es. i nomi degli script `docker-entrypoint-initdb.d` citati in ADR-0014 — non sono stati toccati), `docs/backlog.md`, `docs/funzionale/`, `docs/deploy-e-fork/README.md`, tutti i `README.md`.

**Cosa resta invariato**: il vocabolario di dominio nella prosa italiana (es. "un volontario vede i propri turni", "assegnazione turni" nel backlog) e i nomi visualizzati dei ruoli per-comitato (es. `display_name: "Responsabile turni"` in `config/pavullo/*/roles.json` e nei relativi fixture di test) — questi usano "turni"/"anagrafica" come parole della lingua italiana, non come identificatore del servizio. Il nome del comitato (`pavullo`) e i suoi dati reali non cambiano.

## Conseguenze

- **Supera la frase in [ADR-0010](0010-convenzioni-cross-cutting-servizi-go.md)** sui nomi di campo JSON rimasti in italiano "finché non si deciderà di estendere anche a quelli la convenzione": per `shifts` è stato ora fatto; `registry` non aveva campi in italiano da convertire.
- **Nessuna migrazione dei dati esistenti**: come già accettato per altri rename di schema in questa stessa fase di sviluppo (es. `invite_tokens`→`tokens`), un deploy con dati reali già esistenti (sul Raspberry Pi) andrebbe resettato/rifatto a mano — nessun tooling di migrazione dedicato costruito per questo, il progetto è ancora in fase pre-produzione senza consumer reali.
- Rename simultaneo di directory, moduli Go, schema DB e infrastruttura: nessun periodo intermedio con nomi misti — ogni riferimento è stato verificato con build/vet/test/govulncheck puliti su entrambi i servizi dopo il rename.
- Pattern di naming ora coerente end-to-end (servizio, schema, tabella, colonne, rotte, messaggi) per ogni servizio futuro (`mezzi-magazzino`, `servizi-emergenze`) che replicherà lo stesso stack.
