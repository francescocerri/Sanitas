# Turni

## Chi può fare cosa oggi

- **Chi ha il permesso `shifts:configure` tramite i propri ruoli** (nel Comitato di Pavullo, solo il ruolo "Responsabile turni") può creare un turno-template ricorrente (giorno della settimana + orario, es. "giovedì 20:00–08:00") e modificarne uno esistente, incluso disattivarlo.
- **Chiunque abbia un account attivo** (permesso `shifts:read`, che ogni ruolo ha) può consultare l'elenco dei turni-template esistenti — serve a un volontario per sapere quali slot esistono prima di potersi prenotare — e consultare, per un intervallo di date a scelta (fino a 90 giorni), quali occorrenze concrete di quei turni sono libere, in attesa di approvazione o già confermate.
- **Chi ha il permesso `shifts:request`** (nel Comitato di Pavullo, solo il ruolo "Volontario emergenza") può richiedere di prenotarsi su uno slot: la richiesta nasce sempre "in attesa" — non è ancora una conferma. Viene rifiutata se lo slot è già occupato da una prenotazione confermata, o se il volontario ha già una propria richiesta pending/confermata per lo stesso slot (niente doppioni; una richiesta rifiutata o annullata in precedenza non blocca un nuovo tentativo).
- **Chi ha il permesso `shifts:write`** può consultare quante richieste sono in attesa in totale (`GET /v1/shift-bookings/pending-count` — il numero dietro il futuro contatore/badge del gestore turni) e **approvare o rifiutare** una richiesta in attesa (`PATCH /v1/shift-bookings/{id}`). Una richiesta già decisa (o annullata) non può essere decisa una seconda volta. Se più volontari hanno richiesto lo stesso slot, confermarne uno non rifiuta automaticamente gli altri — vanno smaltiti singolarmente (limite noto, vedi ADR-0025).
- **Non esiste ancora**: nessun modo di prenotare direttamente un volontario senza passare da una richiesta sua — è la prossima attività in `docs/backlog.md` ("Gestione turni", voce 6).

## Turni-template configurati oggi (Comitato di Pavullo)

Seed iniziale da `config/pavullo/shifts/templates.json`, applicato una sola volta all'avvio (se non ci sono già turni-template): giovedì sera, sabato nelle 3 fasce classiche, domenica nelle prime due. Il gestore turni può aggiungerne altri o modificare questi da qui in avanti — un riavvio del servizio non li resetta.
