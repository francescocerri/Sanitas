# 0008. Documentazione API: OpenAPI scritto a mano + Swagger UI via CDN

Status: Accettata

## Contesto

Requisito: le API REST devono essere documentate con OpenAPI/Swagger (vedi sezione "Documentazione" in `CLAUDE.md`). Opzioni valutate per generare/servire la spec in un servizio Go:

1. Annotazioni nei commenti degli handler + tool di generazione (es. `swaggo/swag`) — comodo ma aggiunge un tool di build esterno e un modo di scrivere commenti non idiomatico, con rischio di disallineamento se il tool non gira in CI.
2. Spec OpenAPI scritta a mano, versionata come contratto esplicito.

Per servire la Swagger UI: vendorizzare `swagger-ui-dist` (alcuni MB di JS/CSS) nel repo, oppure caricarla da CDN.

## Decisione

- Spec **scritta a mano** in `services/turni/api/openapi.yaml`, seguendo la convenzione standard dei progetti Go (`/api` per i contratti — vedi golang-standards/project-layout) — cartella da replicare identica in ogni servizio futuro.
- Il file è embeddato nel binario Go (`//go:embed`, in un package `api` dedicato) e servito staticamente su `GET /openapi.yaml` — nessuna dipendenza Go aggiuntiva.
- La UI Swagger (`GET /docs`) è una pagina HTML statica, anch'essa embeddata, che carica `swagger-ui-dist` da CDN (jsdelivr) invece di vendorizzarlo nel repo.

## Conseguenze

- Zero nuove dipendenze Go: `embed` è standard library.
- La spec può disallinearsi dal codice se non ci si ricorda di aggiornarla insieme agli handler — non c'è generazione automatica che lo impedisca. Mitigazione: è un item esplicito di code review (`/code-review` prima del merge, per convenzione in `CLAUDE.md`).
- `GET /docs` richiede accesso a internet per caricare gli asset da CDN — accettabile perché è solo la UI di consultazione, non l'API stessa (che resta pienamente self-hosted e funzionante offline).
- Pattern replicabile identico per ogni servizio futuro (`anagrafica`, `mezzi-magazzino`, `servizi-emergenze`): stessa struttura `api/openapi.yaml` + `api/api.go`, stesso `internal/httpapi/docs.go`.
