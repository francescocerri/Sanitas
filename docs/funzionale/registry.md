# Registry: utenti, ruoli, login

## Chi può fare cosa oggi

- **Chiunque abbia un account attivo** può accedere (login), vedere il proprio profilo (email, username, ruoli) e cambiare la propria password — tramite l'app Sanitas (`app/`, vedi [ADR-0022](../adr/0022-frontend-flutter.md)), non solo da API.
- **Chi ha il permesso `users:manage` tramite i propri ruoli** può creare un nuovo account per un volontario/socio, assegnandogli uno o più ruoli. Se il comitato ha configurato l'invio email (SMTP), il link di attivazione parte automaticamente al nuovo utente; altrimenti resta da copiare e inoltrare a mano, come ripiego sempre disponibile anche quando l'email è configurata. L'operazione è disponibile nell'app tramite «Crea utente» nel profilo. Chi riceve il link di attivazione (via email o a mano) lo apre nell'app, che gli mostra il form per impostare la propria password al primo accesso.
- **Non esiste ancora l'auto-registrazione**: nessuno può crearsi da solo un account, deve farlo un amministratore.

## Attivazione: scelta della password

Il form mostra i requisiti prima dell'invio: almeno 8 caratteri. Il pulsante «Attiva account» resta disabilitato finché la password non rispetta i requisiti e la conferma non coincide; si disabilita anche durante l'invio.

## Cosa manca oggi (non ancora disponibile)

- **"Password dimenticata"**: non esiste ancora un modo per un utente di recuperare l'accesso da solo se dimentica la password — serve chiedere a un amministratore.
- **Gestione dei ruoli da interfaccia**: i ruoli disponibili oggi sono quelli elencati sotto, decisi in fase di configurazione. Cambiarli richiede una modifica di configurazione, non c'è ancora una schermata per farlo.
- **Differenziare cosa vede/può fare ciascun ruolo** nell'applicazione: oggi tutti i ruoli sono solo "etichette" — non cambiano ancora cosa un utente vede o può fare nell'app (a parte il permesso di amministrazione, separato dai ruoli).

## Ruoli disponibili (Comitato di Pavullo)

Presidente, Responsabile turni, Responsabile mezzi, Responsabile vestiario, Responsabile magazzino, Trainer MS, Trainer TSSA, Volontario base, Volontario sociali, Volontario emergenza.

Un utente può avere più ruoli contemporaneamente (es. Presidente **e** Responsabile turni).

## Catalogo ruoli per chi gestisce gli utenti

Chi possiede il permesso `users:manage` può consultare via API i ruoli disponibili (`GET /v1/roles`). Nell’app, il pulsante «Crea utente» nel profilo apre il form con email, username e ruoli selezionabili. L’invio resta disabilitato con dati non validi o durante il caricamento dei ruoli e la creazione. Dopo il salvataggio, se l'email è stata inviata l'app lo segnala; il link di attivazione (valido 7 giorni) resta comunque disponibile da copiare come ripiego, o si può creare subito un altro utente. Email e username già in uso vengono segnalati mantenendo i dati del form.
