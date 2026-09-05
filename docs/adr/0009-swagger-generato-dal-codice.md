# 0009. Documentazione API: Swagger generato dalle annotazioni nel codice

Status: Accettata — supera [ADR-0008](0008-documentazione-api-openapi.md)

## Contesto

ADR-0008 aveva scelto una spec OpenAPI scritta a mano, separata dal codice. Feedback dell'utente dopo averla vista: una spec a mano si scollega facilmente dal codice reale (nessuno garantisce che resti aggiornata quando un handler cambia), e va riscritta manualmente ad ogni modifica — l'opposto di quello che serve in un progetto dove più servizi ripeteranno lo stesso pattern.

## Decisione

La documentazione Swagger si genera dalle **annotazioni nei commenti sopra ogni handler** (standard [swaggo/swag](https://github.com/swaggo/swag), lo strumento più diffuso e mantenuto nell'ecosistema Go per questo). Le annotazioni vivono incollate al codice che descrivono (in `internal/httpapi/server.go`, sopra ogni funzione handler, più un blocco generale sopra `main()`); `swag init` genera `api/docs.go`, `api/swagger.json`, `api/swagger.yaml` — questi file **si committano** (non si generano solo in CI) così il servizio non richiede `swag` per buildare o girare in produzione.

La UI è servita da `github.com/swaggo/http-swagger` su `GET /docs/` — include gli asset di Swagger UI incorporati nella libreria stessa (nessun CDN esterno, a differenza della soluzione in ADR-0008).

**Garanzia anti-disallineamento**: un passo dedicato in CI (`.github/workflows/ci.yml`) rigenera la spec dalle annotazioni e fa fallire la build se il risultato differisce dai file committati (`git diff --exit-code -- api/`) — chi dimentica di rigenerare dopo aver cambiato un endpoint se ne accorge alla PR, non a runtime.

## Conseguenze

- Nuove dipendenze Go: `github.com/swaggo/swag` (libreria di supporto usata dal file generato) e `github.com/swaggo/http-swagger` (UI) — entrambe ampiamente adottate e mantenute nell'ecosistema Go, giustificate secondo la policy in [ADR-0006](0006-policy-sicurezza-dipendenze.md).
- Il CLI `swag` va installato solo da chi rigenera la spec in locale (`go install github.com/swaggo/swag/cmd/swag@latest`) e in CI (via `go run`, pinnato a una versione precisa per riproducibilità) — non è una dipendenza di build del servizio.
- Formato generato: Swagger 2.0 (non OpenAPI 3.x) — è il default di swag; sufficiente per l'obiettivo (documentazione navigabile, sempre allineata al codice).
- Pattern replicabile identico per ogni servizio futuro: stesse annotazioni sopra gli handler, stesso passo di generazione, stesso controllo in CI.
- File generati rimossi da questo ADR: `api/openapi.yaml`, `api/api.go` (sostituiti da `api/docs.go`, `api/swagger.json`, `api/swagger.yaml`); rimossa anche `internal/httpapi/swagger-ui.html` (la UI non serve più asset propri).
