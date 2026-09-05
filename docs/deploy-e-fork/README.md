# Deploy e fork: guida completa

Tutto quello che serve per far partire il progetto da zero, in ordine — sia per chi lavora su questo repo sia per un comitato che lo forka. Non duplica i dettagli già in `deploy/README.md` e nei `README.md` dei singoli servizi: li mette in sequenza e rimanda lì per il dettaglio tecnico.

## 1. Fork e personalizzazione

1. Fork del repository.
2. Creare `config/<nuovo-comitato>/` copiando la struttura di `config/pavullo/` e sostituendo i dati (es. `anagrafica/roles.json` con i ruoli reali della propria associazione — vedi [ADR-0012](../adr/0012-ruoli-come-dati-seed.md)).
3. Scegliere la licenza se diversa da MIT (vedi `LICENSE`).

## 2. Segreti da generare (mai committati)

Per ogni servizio: copiare il relativo `.env.example` in `.env` e compilarlo. In particolare:

- Password dei database (`deploy/.env`, vedi `deploy/README.md`).
- Chiave privata JWT per `anagrafica` (RS256) — vedi `services/anagrafica/README.md` per il comando di generazione.
- Credenziali del primo amministratore (`INITIAL_ADMIN_*` in `deploy/.env`) — usate solo al primo avvio.
- Token del tunnel Cloudflare (solo in produzione) — vedi `deploy/README.md`.

## 3. Avvio in locale (per sviluppo/verifica)

Vedi `deploy/README.md` per il comando `docker compose up` completo, e i `README.md` di ogni servizio (`services/turni`, `services/anagrafica`) per le variabili d'ambiente specifiche.

## 4. Deploy in produzione (Raspberry Pi)

Setup completo (Docker, Cloudflare Tunnel, GitHub Actions self-hosted runner) descritto in `deploy/README.md` — non ripetuto qui per evitare che i due documenti vadano fuori sincrono.

## 5. Verifica che tutto funzioni

- `curl` sugli endpoint `/healthz` di ogni servizio.
- Login con l'amministratore bootstrap (`anagrafica`), creazione di un utente di prova.
- Frontend (`web/`) raggiungibile e che mostra dati reali.

## Note per chi forka

- Il modello è **fork + configurazione**, non multi-tenant: ogni comitato ha una propria istanza indipendente (vedi [ADR-0002](../adr/0002-modello-forkabilita.md)).
- Non deve mai finire hardcoded nel codice sorgente nulla di specifico del proprio comitato — tutto in `config/<slug>/` o env var (vedi `CLAUDE.md`, contratto di forkabilità).
