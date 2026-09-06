# 0015. Messaggi di errore nel body delle risposte HTTP in inglese

Status: Accettata

## Contesto

[ADR-0010](0010-convenzioni-cross-cutting-servizi-go.md) e `CLAUDE.md` documentavano una convenzione linguistica mista: codice, commenti e log in inglese, ma i messaggi nel body delle risposte HTTP (es. `{"error": "credenziali non valide"}`) restavano in italiano, per coerenza col vocabolario di dominio del progetto. Verificando manualmente le API durante il consolidamento del database, l'utente ha notato questi messaggi in italiano e ha chiesto di uniformare tutto all'inglese, invece di mantenere la distinzione.

## Decisione

Tutte le stringhe passate a `writeError(...)` (e i sentinel error di dominio il cui testo può comparire in log o risposte, es. `shift.ErrNotFound`, `user.ErrNotFound`, `user.ErrDuplicateUser`, `user.ErrInvalidToken`) sono ora in inglese, in entrambi i servizi (`shifts`, `registry`). Nessun cambio di status code o struttura JSON — solo il testo del messaggio.

**Restano invariati**, perché non sono "messaggi" rivolti all'utente ma dati/contratto dell'API:
- I nomi dei campi JSON nel body di richieste/risposte (`volontario_id`, `email`, `username`, ecc.) — inconsistenza già nota e aperta in ADR-0010.
- I valori di vocabolario di dominio salvati/restituiti come dati (es. `stato: "pianificato"`).

Le annotazioni Swagger (`@Failure`, ecc.) erano già in inglese (ADR-0009/0010) e non hanno richiesto modifiche.

## Conseguenze

- Nessun consumer reale dell'API oggi: il cambio di testo non rompe alcun client esistente.
- Se in futuro servirà servire messaggi d'errore in italiano a un utente finale non tecnico (es. un frontend rivolto a volontari non anglofoni), andrà introdotto un meccanismo di traduzione/i18n lato frontend — questo ADR non lo prevede, il body dell'API resta un contratto tecnico in inglese.
- Superata la parte di [ADR-0010](0010-convenzioni-cross-cutting-servizi-go.md) che teneva questi messaggi in italiano; il resto di quella decisione (versionamento, logging, timeout, ecc.) resta valido.
