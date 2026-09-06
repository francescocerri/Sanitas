# 0024. Recupero password (forgot-password) via email

Status: Accettata

## Contesto

`docs/backlog.md` elencava ancora aperta "Email di recupero password: completare il flusso forgot-password e il relativo invio via Gmail SMTP". Il servizio aveva già un **cambio password autenticato** (`POST /v1/password/change`, richiede la password attuale + un JWT valido), ma nessun modo per un utente che ha *dimenticato* la password di reimpostarla da solo. L'infrastruttura email (mailer SMTP, branding per-comitato) è già stata costruita per l'invito-utente (ADR-0023) e viene riusata quasi identica.

## Decisione

**Nessuna nuova tabella/migrazione**: la tabella `tokens` ha già una colonna `purpose` libera (oggi `"invite"` e `"refresh"`) — un terzo valore, `"password_reset"`, riusa `CreateToken`/`ConsumeToken` così come sono. TTL dedicato e breve, **1 ora**, molto più corto dei 7 giorni dell'invito: un reset è un'operazione sensibile e ripetibile su un account già esistente, non un primo accesso.

**Due nuovi endpoint pubblici**, in `internal/httpapi/password_reset.go` (non nello stesso file del cambio-password autenticato, che resta un flusso distinto):
- `POST /v1/password/reset/request` `{identifier}` → **sempre 204**, che l'account esista o no. Stesso principio anti-enumerazione già applicato a `handleLogin` e `handleActivateUser` (errori collassati in una risposta generica): mai rivelare tramite lo status code se un identifier corrisponde a un account reale. Se trovato, crea il token e — se un mailer è configurato — invia l'email best-effort con lo stesso pattern di timeout indipendente (6s) già introdotto per l'invito, per la stessa ragione (un SMTP lento non deve mai poter ritardare la risposta oltre il `WriteTimeout` del server).
- `POST /v1/password/reset/confirm` `{token, password}` → rispecchia quasi verbatim `handleActivateUser`: consuma il token con purpose `"password_reset"`, imposta la nuova password, `204`. `401` generico su token invalido/scaduto/già usato.

**Limite noto e accettato**: il ramo "identifier trovato" impiega fino a 6s in più per la chiamata SMTP rispetto al ramo "non trovato", quindi un attaccante che misura i tempi di risposta potrebbe comunque inferire l'esistenza di un account (side-channel temporale). Chiuderlo del tutto richiederebbe simulare lo stesso lavoro anche sul ramo "non trovato" — complessità non giustificata per un'associazione di volontariato che non è un bersaglio ad alto valore. Lo registriamo qui come limite consapevole, non lo ignoriamo silenziosamente.

**Nessuna revoca delle sessioni esistenti** dopo un reset riuscito: `handleChangePassword`, già in produzione, non lo fa nemmeno oggi — introdurlo solo qui creerebbe un'incoerenza tra i due modi di cambiare password. Se in futuro si vuole quella garanzia, va aggiunta a entrambi insieme.

**Mailer**: `SendPasswordResetEmail` + `passwordResetEmailHTML` in `internal/user/mailer.go`, copia strutturale di `SendInviteEmail`/`inviteEmailHTML` (stesso branding, stesso stile HTML a tabella con CSS inline) — cambiano solo oggetto, testo del bottone e la nota di validità (1 ora invece di 7 giorni).

**Config**: nuovo `PASSWORD_RESET_URL_BASE` (default `http://localhost:5173/#/reset-password`), stesso pattern di `INVITE_URL_BASE`.

## Conseguenze

- `NewServer(...)` guadagna un parametro (`passwordResetURLBase string`) — tutte le chiamate esistenti (produzione e test) aggiornate.
- Nessuna nuova infrastruttura di rate-limiting: non ne esisteva già nessuna nel servizio (nemmeno su `/v1/login`) e introdurla qui da sola sarebbe incoerente col resto — resta una nota per un'eventuale attività futura, non bloccante per questa.
- La UI Flutter (`app/`) guadagna due nuove schermate (`/forgot-password`, `/reset-password`), non un'estensione di schermate esistenti.
