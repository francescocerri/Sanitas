# 0018. Permessi granulari per ruolo, `is_admin` rimosso

Status: Accettata

## Contesto

L'unica distinzione di autorizzazione era `is_admin`, un booleano di sistema separato dai ruoli organizzativi ([ADR-0013](0013-autenticazione-bcrypt-jwt.md)). Serviva un'autorizzazione più granulare, in entrambi i servizi, con una decisione esplicita: nessun bypass superuser, tutta l'autorizzazione passa attraverso i permessi assegnati per ruolo — `is_admin` viene rimosso, non affiancato.

**Scope**: solo azioni consentite/negate per permesso (es. *"serve `shifts:write` per creare un turno"*). Filtrare i **dati** restituiti per ruolo (es. un volontario che vede solo i propri turni) resta fuori: richiede prima progettare come `turni` rappresenta un "proprietario" della vista — non esiste ancora, voce di backlog separata.

## Decisione

**Permessi: vocabolario fisso nel codice, assegnazione per ruolo nella config.** `users:manage`, `shifts:read`, `shifts:write` corrispondono 1:1 a endpoint reali in entrambi i servizi — non variano da comitato a comitato, quindi sono costanti Go (`internal/user/permissions.go` in anagrafica), non dati seed come i ruoli ([ADR-0012](0012-ruoli-come-dati-seed.md)). Quale ruolo ottiene quale permesso *è* invece per-comitato: `config/<slug>/anagrafica/roles.json` guadagna un array `permissions` per voce. `SeedRoles` valida ogni slug contro il vocabolario noto — un permesso sconosciuto in config è quasi certamente un typo, fail-fast all'avvio invece di seedare silenziosamente un ruolo senza effetto.

**Schema**: `roles` guadagna `permissions TEXT[] NOT NULL DEFAULT '{}'` — nessuna tabella `role_permissions` a parte, l'insieme è piccolo e statico. `users` **perde** `is_admin`. I permessi di un utente si risolvono unendo tutti i ruoli assegnati e deduplicando (`Repository.GetPermissionsForUser`), esposti nel JWT come nuovo claim `permissions` (sostituisce `is_admin` sia in `anagrafica/internal/user.Claims` che nella copia a mano in `turni/internal/authclient.Claims`, stesso vincolo di moduli separati già in [ADR-0017](0017-turni-verifica-jwt.md)).

**Bootstrap senza flag speciale.** Il primo admin (creato da env var quando `users` è vuota) non ha più un flag: `Bootstrap` gli assegna un ruolo tecnico riservato (`bootstrap_admin`, creato da codice — non nel file di config del comitato, non un ruolo organizzativo) con tutti i permessi noti. L'utente di bootstrap è così un utente ordinario la cui unica particolarità è quale ruolo ha all'inizio — un solo meccanismo di autorizzazione (ruolo → permessi), nessun caso speciale nel codice.

**Middleware**: `requirePermission(permission, next)` sostituisce `requireAdmin` in anagrafica (`POST /v1/users` ora richiede `users:manage`) ed è la prima cosa del genere in `turni`, dove protegge `GET /v1/shifts`, `GET /v1/shifts/{id}` (`shifts:read`) e `POST /v1/shifts` (`shifts:write`). 403 su permesso mancante, distinto dal 401 di autenticazione.

## Conseguenze

- La voce aperta in ADR-0017 ("`roles`/`is_admin` mappati ma non usati da alcuna logica — l'autorizzazione granulare resta backlog separato") è chiusa da questa decisione.
- Assegnare un permesso potente (es. `users:manage`) a un ruolo organizzativo è ora una decisione del comitato in un file di config, non una modifica al codice — coerente con il contratto di forkabilità.
- Filtrare i dati per ruolo (visibilità sui singoli turni) resta fuori scope — richiede prima la progettazione del dominio reale di `turni`, voce di backlog già separata.
- Nessun rilevamento di riuso/revoca di sessioni attive se un ruolo viene modificato: un token già emesso resta valido con i permessi calcolati al momento dell'emissione fino alla sua scadenza (24h) o al prossimo refresh — stesso limite già noto per i cambi di ruolo in generale.
