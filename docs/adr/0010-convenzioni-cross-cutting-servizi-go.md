# 0010. Convenzioni cross-cutting per i servizi Go: logging, error handling, timeout

Status: Accettata

## Contesto

I servizi Go devono seguire linee guida standard per microservizi. `turni` (primo servizio) aveva gap concreti: nessun logging degli errori applicativi (finivano solo nella risposta HTTP al client, `err.Error()` incluso — driver Postgres compreso), nessun panic recovery, nessun timeout su `http.Server` oltre `ReadHeaderTimeout`, e il layer HTTP importava `pgx` solo per riconoscere "riga non trovata".

## Decisione

- **Logging strutturato**: `log/slog` (standard library, JSON su stdout). Un unico middleware (`withLogging`) fa da access log (metodo, path, status, durata) e da panic recovery insieme — pattern standard in Go, evita che un panic in un handler chiuda la connessione senza risposta né traccia.
- **Error handling**: gli errori interni si loggano via `slog` con contesto (`operazione: %w` wrapping nel repository) e non si espongono mai al client — le risposte di errore sono sempre `{"error": "messaggio generico"}` in JSON. Il repository traduce gli errori del driver in errori di dominio (`turno.ErrNotFound`), così i layer superiori non conoscono `pgx`.
- **Timeout**: `http.Server` ha `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` tutti espliciti. La context propagation verso il database era già corretta (ogni handler passa `r.Context()` fino a pgx) e non è stata toccata.
- **Struttura a layer**: **non introdotto** un layer di servizio/business logic tra HTTP e repository — il modello dati è ancora un placeholder (vedi [ADR-0005](0005-database-postgres-self-hosted.md)), aggiungerlo ora sarebbe un'astrazione prematura. Da rivalutare quando si disegnerà il dominio reale.

## Conseguenze

- Zero nuove dipendenze esterne: tutto quanto sopra è standard library.
- Pattern replicabile identico per ogni servizio futuro (`anagrafica`, `mezzi-magazzino`, `servizi-emergenze`): stesso middleware di logging/recovery, stessa convenzione di wrapping errori e traduzione in errori di dominio, stessi timeout su `http.Server`.
- Le risposte di errore cambiano forma (da testo semplice a `{"error": "..."}` JSON) — è un cambio visibile per chiunque già consumi l'API, accettabile perché il servizio non ha ancora consumer reali.
- La decisione di non introdurre un layer di servizio resta aperta: quando la sessione di progettazione del dominio reale (backlog) definirà la logica di business, questo ADR andrà probabilmente superato da uno nuovo.
