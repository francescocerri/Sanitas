# 0012. Ruoli come dati seed per-comitato, non enum nel codice

Status: Accettata

## Contesto

Il servizio `registry` deve gestire ruoli organizzativi (responsabile turni, presidente, trainer TSSA, ecc.). Questi nomi sono specifici del Comitato di Pavullo/CRI — "trainer TSSA" non ha senso per una Croce Verde — e vanno trattati secondo il contratto di forkabilità già in `CLAUDE.md`.

## Decisione

I ruoli sono **dati seed**, non un enum Go: vivono in `config/<committee-slug>/registry/roles.json` (slug inglese, nome visualizzato in italiano), applicati (upsert idempotente) da `registry` ad ogni avvio, path indicato da `ROLES_SEED_PATH`. Questa è la prima occasione concreta in cui `config/<committee-slug>/` (finora solo descritta in `CLAUDE.md`) viene effettivamente creata e usata: `config/pavullo/registry/roles.json`.

## Conseguenze

- Chi forka il progetto sostituisce solo quel file per avere i propri ruoli — nessuna modifica al codice Go.
- L'upsert è idempotente e gira ad ogni avvio (non solo alla creazione del DB, a differenza dello schema in `/docker-entrypoint-initdb.d/`): un fork può aggiornare i propri ruoli e vederli applicati con un semplice riavvio del servizio.
- Nessuna interfaccia di amministrazione per i ruoli in questa fase — modificarli significa editare il file e riavviare, accettabile alla scala attuale (backlog per un'eventuale gestione via API).
- Ruoli di Pavullo seedati oggi: `president`, `shift_manager`, `vehicle_manager`, `uniform_manager`, `warehouse_manager`, `trainer_ms`, `trainer_tssa`, `base_volunteer`, `social_services_volunteer`, `emergency_volunteer`.
