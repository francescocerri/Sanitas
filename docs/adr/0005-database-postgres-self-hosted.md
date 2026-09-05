# 0005. Database: PostgreSQL self-hosted, schema via init script

Status: Accettata

## Contesto

Serve un database relazionale per servizi come `turni`, adatto a un dominio con relazioni tra entità che cresceranno con `anagrafica` e `mezzi-magazzino`.

## Decisione

**PostgreSQL self-hosted**, un container nel `docker-compose` (vedi [ADR-0004](0004-target-di-deploy.md)), dati su volume Docker. Schema applicato tramite script SQL montati in `/docker-entrypoint-initdb.d/` (feature nativa dell'immagine ufficiale Postgres) — nessun tool di migrazione per questa prima fase.

## Conseguenze

- Zero dipendenze/tool di migrazione aggiuntivi finché lo schema non deve evolvere in modo incrementale su dati già in produzione. Gli script in `/docker-entrypoint-initdb.d/` girano **solo alla prima creazione del volume dati** — non sono un meccanismo di migrazione, sono adatti solo a un progetto ancora senza dati reali in produzione.
- Quando servirà evolvere lo schema senza perdere dati (prima release vera in produzione, o più servizi/più ambienti), va introdotto uno strumento di migrazioni dedicato — tracciato in `docs/backlog.md`.
- Nessuna strategia di backup automatico del DB ancora implementata (es. `pg_dump` periodico verso storage esterno) — nota aperta, tracciata in `docs/backlog.md`.
