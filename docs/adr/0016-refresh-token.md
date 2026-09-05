# 0016. Refresh token, riusando la tabella `invite_tokens`

Status: Accettata

## Contesto

L'access token JWT emesso da `POST /v1/login` dura 24h senza alcun modo di rinnovarlo — nota già come limite aperto in [ADR-0013](0013-autenticazione-bcrypt-jwt.md). Serve un refresh token per permettere sessioni più lunghe senza dover ripetere email/password ogni 24h, e un modo di invalidarlo (altrimenti un refresh token rubato varrebbe quanto una password che non scade mai).

## Decisione

**Nessuna nuova tabella.** La tabella `invite_tokens` (già usata per i token di attivazione account) ha `purpose TEXT`, `token_hash`, `expires_at`, `used_at` — generica per design, il suo stesso commento anticipava questo riuso. I refresh token sono righe con `purpose = 'refresh'`, creati e consumati con gli stessi `Repository.CreateInviteToken`/`ConsumeInviteToken` già esistenti, nessun metodo nuovo nel repository.

**Rotazione a ogni refresh**: `ConsumeInviteToken` marca il token usato in una singola query atomica (vedi ADR originale) — riusarlo per il refresh significa che ogni `POST /v1/refresh` consuma il token presentato e ne crea uno nuovo, gratis. Un token già usato non è più valido: ripresentarlo restituisce 401.

**Endpoint**:
- `POST /v1/login` ora restituisce sia l'access token che un refresh token (struct `authTokens`, non più una mappa anonima).
- `POST /v1/refresh` (non autenticato — il refresh token nel body è la credenziale, stesso pattern di `POST /v1/users/activate`): consuma il refresh token, riemette entrambi.
- `POST /v1/logout` (non autenticato, stesso pattern): consuma il refresh token senza riemetterne uno nuovo.

**Autenticazione classica, solo `Bearer <token>`**: `requireAuth` richiede rigorosamente il prefisso "Bearer " (nessuna eccezione per un token nudo) — coerente con lo standard. Lo Swagger UI (schema `apiKey`, l'unica opzione per un bearer token sotto Swagger 2.0 — non esiste un tipo "http bearer" con prefisso automatico) non può anteporlo da solo nel campo "Authorize": va scritto a mano insieme al token ogni volta, limite noto e accettato dello strumento, non del servizio.

**Refresh token opaco, non un JWT**: scelta deliberata, non un dettaglio implementativo lasciato al caso. Un JWT è stateless — una volta firmato resta valido fino a scadenza, senza modo di revocarlo prima. Per supportare il logout (invalidare il refresh token su richiesta) servirebbe comunque uno stato lato server (una blocklist di JWT revocati, o una tabella di sessioni), che duplicherebbe esattamente quello che `invite_tokens` già offre. Coerente con lo standard OAuth2 (RFC 6749): il refresh token è definito opaco dal punto di vista del client, l'access token resta l'unico stateless/autocontenuto.

## Conseguenze

- TTL scelto per il refresh token: 30 giorni (costante `refreshTokenTTL`, facilmente cambiabile).
- **Nessun rilevamento di riuso/furto** (pattern più avanzato "token family": se un token rubato viene usato prima del legittimo proprietario, non c'è modo di accorgersene né di revocare l'intera sessione) — limite noto, non affrontato in questa fase.
- **Nessuna revoca dell'access token già emesso**: il logout invalida solo il refresh token; un access token rubato resta valido fino alla sua scadenza naturale (24h) — stesso limite già in ADR-0013, invariato.
- `turni` non verifica ancora alcun JWT (voce di backlog separata) — il refresh token oggi ha effetto solo su `anagrafica`.
