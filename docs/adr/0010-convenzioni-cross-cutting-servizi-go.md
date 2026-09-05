# 0010. Convenzioni cross-cutting per i servizi Go: logging, error handling, timeout

Status: Accettata

## Contesto

I servizi Go devono seguire linee guida standard per microservizi. `turni` (primo servizio) aveva gap concreti: nessun logging degli errori applicativi (finivano solo nella risposta HTTP al client, `err.Error()` incluso — driver Postgres compreso), nessun panic recovery, nessun timeout su `http.Server` oltre `ReadHeaderTimeout`, e il layer HTTP importava `pgx` solo per riconoscere "riga non trovata".

## Decisione

- **Logging strutturato**: `log/slog` (standard library, JSON su stdout). Un unico middleware (`withLogging`) fa da access log (metodo, path, status, durata) e da panic recovery insieme — pattern standard in Go, evita che un panic in un handler chiuda la connessione senza risposta né traccia.
- **Error handling**: gli errori interni si loggano via `slog` con contesto (`operazione: %w` wrapping nel repository) e non si espongono mai al client — le risposte di errore sono sempre `{"error": "messaggio generico"}` in JSON. Il repository traduce gli errori del driver in errori di dominio (`turno.ErrNotFound`), così i layer superiori non conoscono `pgx`.
- **Timeout**: `http.Server` ha `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` tutti espliciti. La context propagation verso il database era già corretta (ogni handler passa `r.Context()` fino a pgx) e non è stata toccata.
- **Struttura a layer**: **non introdotto** un layer di servizio/business logic tra HTTP e repository — il modello dati è ancora un placeholder (vedi [ADR-0005](0005-database-postgres-self-hosted.md)), aggiungerlo ora sarebbe un'astrazione prematura. Da rivalutare quando si disegnerà il dominio reale.
- **Versionamento API**: prefisso `/v1` sugli endpoint che espongono risorse (`/shifts`, `/shifts/{id}` in `turni`; `/users`, `/users/activate`, `/password/change` in `anagrafica`). `/healthz`, `/docs/` e altri endpoint operativi/meta (es. `/.well-known/jwks.json` in `anagrafica`) restano **non** versionati — non fanno parte del contratto che evolve. **Niente `@BasePath` globale nelle annotazioni swag**: Swagger 2.0 non supporta un basePath diverso per singolo path, quindi con rotte miste versionate/non versionate farebbe chiamare a "Try it out" l'URL sbagliato per quelle non versionate (bug reale, trovato verificando `anagrafica` a mano nel browser — corretto anche in `turni`, dove era passato inosservato). Ogni `@Router` scrive il path reale per intero (`/v1/shifts` oppure `/healthz`), niente ambiguità.
- **Segmenti del path in inglese**: i nomi delle risorse nei path delle route sono in inglese, non solo commenti/log/annotazioni swag — es. `turni`→`shifts`, `utenti`→`users`, `attiva`→`activate`, `cambia`→`change` (fix applicata retroattivamente sia a `turni` che ad `anagrafica`). I nomi dei campi JSON nel body di richieste/risposte restano invece invariati in italiano (vocabolario di dominio, es. `volontario_id`, `email`, `username`) finché non si deciderà di estendere anche a quelli la convenzione — inconsistenza nota, lasciata aperta intenzionalmente.
- **Log del body delle richieste con body** (oggi solo `POST`): loggato nella stessa riga dell'access log, con i valori dei campi PII (denylist locale per servizio, es. `volontario_id`) sostituiti da `"[redacted]"` prima di scrivere il log. Se il body non è JSON valido, si logga un placeholder fisso invece dei byte grezzi — mai rischiare di loggare PII per un parse fallito.

## Conseguenze

- Zero nuove dipendenze esterne: tutto quanto sopra è standard library.
- Pattern replicabile identico per ogni servizio futuro (`anagrafica`, `mezzi-magazzino`, `servizi-emergenze`): stesso middleware di logging/recovery, stessa convenzione di wrapping errori e traduzione in errori di dominio, stessi timeout su `http.Server`, stesso prefisso `/v1`, stessa denylist locale di campi PII da mascherare nei log (ogni servizio manterrà la propria, dato che sono moduli Go indipendenti senza codice condiviso — vedi [ADR-0003](0003-stack-tecnico.md)).
- Chi aggiunge un campo che identifica una persona a un servizio esistente deve ricordarsi di aggiungerlo alla denylist locale di quel servizio — non è imposto meccanicamente, resta responsabilità di code review.
- Le risposte di errore cambiano forma (da testo semplice a `{"error": "..."}` JSON) — è un cambio visibile per chiunque già consumi l'API, accettabile perché il servizio non ha ancora consumer reali.
- La decisione di non introdurre un layer di servizio resta aperta: quando la sessione di progettazione del dominio reale (backlog) definirà la logica di business, questo ADR andrà probabilmente superato da uno nuovo.
