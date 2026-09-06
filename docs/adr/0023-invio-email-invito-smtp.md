# 0023. Invio automatico dell'email di invito via SMTP

Status: Accettata

## Contesto

ADR-0013 ("Fase A") aveva scelto esplicitamente di restituire l'URL di invito direttamente nella risposta di `POST /v1/users`, non spedirlo via email, rimandando l'invio a un'attività separata: *"l'URL di invito è restituito direttamente nella risposta API, non spedito via email (attività separata)"*. Questa attività chiude quel gap: l'email parte automaticamente quando il servizio ha un SMTP configurato, mantenendo il comportamento precedente (link da copiare a mano) per chi non lo configura.

## Decisione

**Libreria**: [`github.com/wneessen/go-mail`](https://github.com/wneessen/go-mail), non `net/smtp` (stdlib, nessun supporto MIME/HTML, `Auth` pre-OAuth2/pre-Gmail-moderno) né `gopkg.in/gomail.v2` (non più mantenuta) — coerente con "stdlib quando basta, altrimenti una libreria piccola e mantenuta" già seguito per `golang-jwt/jwt` e `bcrypt`.

**Configurazione a due livelli**, stesso schema segreti-vs-dati-di-comitato già in uso nel progetto:
- **Segreti** (env var, mai committati): `SMTP_HOST`, `SMTP_PORT` (default `587`), `SMTP_USERNAME`, `SMTP_PASSWORD`. Tutti opzionali — **se `SMTP_HOST` è vuoto l'invio resta disabilitato**, comportamento identico a prima di questa attività. Nessun fork è costretto a configurare Gmail per continuare a funzionare.
- **Branding, non segreto, versionato nel fork**: nuovo file `config/<slug>/registry/email.json` (`from_name`, `from_address`), caricato al bootstrap esattamente come `roles.json` (ADR-0012) tramite un nuovo env var opzionale `EMAIL_CONFIG_PATH`, dentro il volume `/config` già montato — nessun nuovo volume in `docker-compose.yml`.

**Nessuna interfaccia/mock**: `internal/user.Mailer` è una struct concreta che incapsula un client `go-mail`; se `SMTP_HOST` è vuoto il campo `mailer` di `Server` resta `nil` e `handleCreateUser` salta l'invio. Stesso identico codice in produzione e nei test, cambia solo l'host SMTP configurato — coerente con "dipendenze reali, non mock" (ADR-0010/0011).

**Contratto API**: `createUserResponse` guadagna `email_sent bool`, `invite_url` resta **sempre** presente (ripiego se l'email non arriva, non solo se SMTP non è configurato). Un errore di invio è loggato (`slog.Error`) e mai esposto al client: l'utente è già stato creato con successo, l'invio è un effetto collaterale best-effort che non fa fallire la richiesta — niente nuovo status code, nessun precedente di risposta parziale/errore esisteva già in `internal/httpapi` e non serviva inventarne uno per questo caso.

**Test**: nessun mock — un server SMTP fittizio (`axllent/mailpit`) avviato via testcontainers-go (`internal/testmail`, stesso pattern di `internal/testdb` per Postgres), condiviso per tutta la durata dei test del package. Un test crea un utente con SMTP configurato e verifica sia `email_sent: true` sia, tramite l'API HTTP di mailpit, che il messaggio sia stato davvero recapitato; un altro verifica che senza mailer configurato `email_sent` resti `false` e il comportamento esistente non cambi.

## Conseguenze

- Nuova dipendenza esterna in `registry`: `github.com/wneessen/go-mail` (aggiunta a `CLAUDE.md`).
- `NewServer(...)` guadagna due parametri (`mailer *user.Mailer`, `emailBranding user.EmailBranding`) — tutte le chiamate esistenti (produzione e test) aggiornate.
- Nessuna rottura per deployment/fork esistenti: SMTP non configurato → comportamento identico a prima di questa attività.
- Il mittente reale del Comitato di Pavullo (`config/pavullo/registry/email.json`) è dato vero del comitato, non un placeholder — un nuovo fork lo sostituisce con il proprio, generando la propria app password Gmail (attività manuale, fuori dal codice).
- La UI Flutter (`app/`) si biforca sullo stesso schermo di successo già esistente in base a `email_sent`, senza introdurre una nuova schermata.
