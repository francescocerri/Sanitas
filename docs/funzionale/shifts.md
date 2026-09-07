# Turni

## Chi può fare cosa oggi

- **Chi ha il permesso `shifts:configure` tramite i propri ruoli** (nel Comitato di Pavullo, solo il ruolo "Responsabile turni") può creare un turno-template ricorrente (giorno della settimana + orario, es. "giovedì 20:00–08:00") e modificarne uno esistente, incluso disattivarlo.
- **Chiunque abbia un account attivo** (permesso `shifts:read`, che ogni ruolo ha) può consultare l'elenco dei turni-template esistenti — serve a un volontario per sapere quali slot esistono prima di potersi prenotare.
- **Non esiste ancora**: nessuna prenotazione, nessuna vista a calendario con la copertura reale (slot liberi/in attesa/confermati) — sono le prossime attività in `docs/backlog.md` ("Gestione turni", voci 3-7). Oggi un turno-template è solo la definizione dello slot, non ancora collegato a chi lo occupa.

## Turni-template configurati oggi (Comitato di Pavullo)

Seed iniziale da `config/pavullo/shifts/templates.json`, applicato una sola volta all'avvio (se non ci sono già turni-template): giovedì sera, sabato nelle 3 fasce classiche, domenica nelle prime due. Il gestore turni può aggiungerne altri o modificare questi da qui in avanti — un riavvio del servizio non li resetta.
