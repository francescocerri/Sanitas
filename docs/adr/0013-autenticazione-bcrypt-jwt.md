# 0013. Autenticazione: bcrypt + JWT RS256 + bootstrap admin via env

Status: Accettata

## Contesto

`anagrafica` deve gestire login, sessioni e permessi per gli altri servizi. Nessuna auto-registrazione: gli account li crea un amministratore, il nuovo utente li attiva tramite un token di invito impostando la propria password.

## Decisione

- **Password hashing**: `bcrypt` (`golang.org/x/crypto/bcrypt`) — autocontenuto (cost factor e salt incorporati nell'hash restituito), non richiede di codificare a mano un formato come servirebbe con argon2id. Resta pienamente accettato (seconda scelta OWASP dopo argon2id, non deprecato).
- **Sessione**: JWT (`github.com/golang-jwt/jwt/v5`) firmato con **coppia di chiavi RSA (RS256)**, non HMAC a secret condiviso — richiesto esplicitamente: solo `anagrafica` detiene la chiave privata (genera una tantum, mai committata, path da `JWT_PRIVATE_KEY_PATH`), chiunque altro verifica con la sola chiave pubblica esposta su `GET /.well-known/jwks.json` (formato JWKS standard), senza mai poter firmare token per conto proprio. Claim: user id, username, ruoli (slug), `is_admin`, scadenza (24h).
- **Bootstrap del primo admin**: nessun endpoint pubblico di setup. Se la tabella `users` è vuota all'avvio, il servizio crea il primo utente da `INITIAL_ADMIN_EMAIL`/`INITIAL_ADMIN_USERNAME`/`INITIAL_ADMIN_PASSWORD` (env var, mai committate). Se `users` non è vuota, queste variabili vengono ignorate.
- **Registrazione**: solo tramite `POST /v1/utenti` (richiede `is_admin`), che crea un utente pending e un token di invito monouso (hash salvato, mai il token in chiaro). L'utente lo consuma su `POST /v1/utenti/attiva` per impostare la password. In questa fase l'URL di invito è restituito direttamente nella risposta API, non spedito via email (attività separata).
- **Permesso di amministrazione** (`is_admin`) è un campo di sistema distinto dai ruoli organizzativi (un "presidente" non è automaticamente abilitato a gestire account) — un vero modello di permessi granulari per ruolo (chi vede/fa cosa nell'interfaccia) resta da progettare quando esisteranno schermate/endpoint su cui applicarlo.

## Conseguenze

- Nuove dipendenze: `golang.org/x/crypto/bcrypt` (stesso ecosistema fidato di altre dipendenze già in uso) e `github.com/golang-jwt/jwt/v5` (standard de facto Go per JWT, mantenuto attivamente).
- Nessuna chiamata di rete tra servizi per validare un token: chi lo consuma verifica la firma localmente con la chiave pubblica, recuperata una volta da JWKS e messa in cache.
- Un token compromesso resta valido fino alla scadenza (24h) — non esiste ancora un meccanismo di revoca; da valutare se servirà (blocklist, rotazione della chiave).
- Il file della chiave privata e le credenziali del bootstrap admin richiedono la stessa disciplina operativa delle altre credenziali (mai committati, iniettati via volume/env var — vedi `deploy/docker-compose.yml`).
