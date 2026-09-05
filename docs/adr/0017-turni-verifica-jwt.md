# 0017. `turni` verifica i JWT emessi da `anagrafica` via JWKS locale

Status: Accettata

## Contesto

`turni` non verificava alcun token: tutti i suoi endpoint `/v1/*` erano raggiungibili senza autenticazione, mentre `anagrafica` protegge già i propri. La scelta di RS256 in [ADR-0013](0013-autenticazione-bcrypt-jwt.md) anticipava esplicitamente questo scenario: un servizio che detiene solo la chiave pubblica può verificare i token senza mai contattare `anagrafica` per ogni richiesta.

## Decisione

**Verifica locale via JWKS, non introspection sincrona.** `turni` recupera la chiave pubblica di `anagrafica` da `GET /.well-known/jwks.json` (nuovo pacchetto `internal/authclient`), la mantiene in cache in memoria, e verifica la firma RS256 di ogni token localmente — nessuna chiamata di rete su ogni richiesta protetta. L'alternativa (chiamare `anagrafica` in sincrono per validare ogni token, come una introspection OAuth2) avrebbe introdotto un accoppiamento e una latenza non necessari, vanificando il motivo stesso per cui si è scelto RS256 invece di HMAC.

**Claims duplicati, non condivisi**: `turni` e `anagrafica` sono moduli Go indipendenti senza codice condiviso ([ADR-0003](0003-stack-tecnico.md)) — `authclient.Claims` rispecchia a mano la forma JSON di `anagrafica/internal/user.Claims`. Oggi `turni` verifica solo l'autenticazione (firma + scadenza): `roles`/`is_admin` sono mappati ma non usati da alcuna logica — l'autorizzazione granulare per ruolo resta una voce di backlog separata.

**Rotazione della chiave gestita senza riavvio**: se il `kid` di un token non è nella cache, `authclient.Client` tenta un unico refetch della JWKS prima di rifiutare — così una rotazione della chiave lato `anagrafica` viene recepita automaticamente. Un guard temporale (non più di un refetch ogni 60s) evita che token con `kid` casuale causino un fetch per ogni richiesta.

**Avvio: fail-fast con retry**. `turni` recupera la JWKS prima di accettare traffico; qualche tentativo con una breve pausa tra uno e l'altro copre il caso normale in cui `anagrafica` non è ancora pronta quando `docker-compose` avvia i due servizi insieme (nessun healthcheck su `anagrafica` oggi). Esauriti i tentativi, il servizio non parte — senza la chiave non può autenticare nessuno in sicurezza, stesso principio già applicato a `DATABASE_URL`/`LoadKeyPair` in `anagrafica`.

## Conseguenze

- `turni` ha ora una dipendenza di avvio da `anagrafica` che prima non aveva (`depends_on` in `docker-compose.yml`, nuova variabile `AUTH_JWKS_URL` obbligatoria).
- Nuova dipendenza esterna in `turni`: `github.com/golang-jwt/jwt/v5` — già scelta/vettata in ADR-0013 per `anagrafica`, nessuna nuova valutazione di sicurezza necessaria.
- Solo autenticazione, non autorizzazione: chiunque abbia un token valido (qualunque ruolo) può oggi creare/leggere turni — permessi granulari per ruolo restano backlog separato (vedi anche ADR-0013).
- `volontario_id` resta nel body della richiesta, non derivato dai claims del token — cambiare questo contratto non è nello scope di questa decisione.
