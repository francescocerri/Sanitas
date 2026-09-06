# Registry: utenti, ruoli, login

## Chi può fare cosa oggi

- **Chiunque abbia un account attivo** può accedere (login), vedere il proprio profilo (email, username, ruoli) e cambiare la propria password — tramite l'app Sanitas (`app/`, vedi [ADR-0022](../adr/0022-frontend-flutter.md)), non solo da API.
- **Un amministratore** (permesso di sistema, non un ruolo organizzativo) può creare un nuovo account per un volontario/socio, assegnandogli uno o più ruoli, e ricevere il link di attivazione da inoltrare a mano al nuovo utente. **Oggi questa operazione si fa solo da API** (`POST /v1/users`, es. via curl/Postman): non esiste ancora un pannello admin nell'app (voce di backlog separata). Chi riceve il link di attivazione lo apre invece direttamente nell'app, che gli mostra il form per impostare la propria password al primo accesso.
- **Non esiste ancora l'auto-registrazione**: nessuno può crearsi da solo un account, deve farlo un amministratore.

## Attivazione: scelta della password

Il form mostra i requisiti prima dell'invio: almeno 8 caratteri. Il pulsante «Attiva account» resta disabilitato finché la password non rispetta i requisiti e la conferma non coincide; si disabilita anche durante l'invio.

## Cosa manca oggi (non ancora disponibile)

- **Invio automatico dell'email di invito**: oggi il link di attivazione viene mostrato solo a chi crea l'account (l'amministratore), che deve inoltrarlo a mano al volontario (WhatsApp, di persona, ecc.). L'invio automatico via email è previsto ma non ancora fatto.
- **"Password dimenticata"**: non esiste ancora un modo per un utente di recuperare l'accesso da solo se dimentica la password — serve chiedere a un amministratore.
- **Gestione dei ruoli da interfaccia**: i ruoli disponibili oggi sono quelli elencati sotto, decisi in fase di configurazione. Cambiarli richiede una modifica di configurazione, non c'è ancora una schermata per farlo.
- **Differenziare cosa vede/può fare ciascun ruolo** nell'applicazione: oggi tutti i ruoli sono solo "etichette" — non cambiano ancora cosa un utente vede o può fare nell'app (a parte il permesso di amministrazione, separato dai ruoli).

## Ruoli disponibili (Comitato di Pavullo)

Presidente, Responsabile turni, Responsabile mezzi, Responsabile vestiario, Responsabile magazzino, Trainer MS, Trainer TSSA, Volontario base, Volontario sociali, Volontario emergenza.

Un utente può avere più ruoli contemporaneamente (es. Presidente **e** Responsabile turni).
