# Turni

## Le 4 figure di un turno

Ogni turno-template (es. "giovedì 20:00–08:00") è composto da **4 figure indipendenti**, fino a un massimo di 4 persone: **autista**, **leader** (capo equipaggio), **soccorritore**, **osservatore**. Ogni prenotazione — richiesta del volontario o diretta del gestore — indica sempre per quale figura, mai per "il turno" nel suo complesso; una stessa persona tiene al più una figura per occorrenza (non può prenotarsi due volte sullo stesso turno+data, indipendentemente dalla figura). Le 4 figure sono un dato fisso, uguale per ogni comitato che forka il progetto — un equipaggio CRI, non una scelta specifica di Pavullo.

Un turno si considera **"al completo"** quando le 3 figure operative — autista, leader, soccorritore — sono confermate; l'osservatore è facoltativo e la sua assenza non impedisce lo stato "completo".

## Chi può fare cosa oggi

- **Chi ha il permesso `shifts:configure` tramite i propri ruoli** (nel Comitato di Pavullo, solo il ruolo "Responsabile turni") può creare un turno-template ricorrente (giorno della settimana + orario, es. "giovedì 20:00–08:00") e modificarne uno esistente, incluso disattivarlo.
- **Chiunque abbia un account attivo** (permesso `shifts:read`, che ogni ruolo ha) può consultare l'elenco dei turni-template esistenti e, per un intervallo di date a scelta (fino a 90 giorni), la copertura delle 4 figure di ogni occorrenza concreta di quei turni — per ciascuna figura: libera, in attesa di approvazione o già confermata, più `my_booking_status` (`pending`/`confirmed`/assente), lo stato della PROPRIA eventuale prenotazione su quella figura — distinto dallo stato aggregato, perché una figura "in attesa" può esserlo per una richiesta di qualcun altro.
- **Chi ha il permesso `shifts:request`** (nel Comitato di Pavullo, solo il ruolo "Volontario emergenza") può richiedere di prenotarsi su una figura libera di uno slot: la richiesta nasce sempre "in attesa" — non è ancora una conferma. Viene rifiutata se quella figura è già occupata da una prenotazione confermata, o se il volontario ha già una propria richiesta pending/confermata per lo stesso turno+data (qualunque figura — niente doppioni; una richiesta rifiutata o annullata in precedenza non blocca un nuovo tentativo). Può anche richiedere **più figure in un'unica chiamata** (`POST /v1/shift-bookings/bulk`, lista di `{template_id, date, role}`): o vengono create tutte, o — se anche un solo elemento non è valido (data passata, figura già occupata, doppione sullo stesso turno+data, ecc.) — non ne viene creata nessuna; comodo per prenotarsi su più giorni in un colpo solo invece di ripetere la chiamata singola una volta per figura.
- **Chi ha il permesso `shifts:write`** può consultare quante richieste sono in attesa in totale (`GET /v1/shift-bookings/pending-count`) o con il dettaglio completo (`GET /v1/shift-bookings/pending` — volontario, turno, data, figura), **approvare o rifiutare** una richiesta in attesa (`PATCH /v1/shift-bookings/{id}`) e **prenotare direttamente un volontario** su una figura (`POST /v1/shift-bookings/direct`, corpo `{template_id, volunteer_id, date, role}`) — nessuna approvazione necessaria, la prenotazione nasce già confermata e `decided_by` è il gestore stesso. Una richiesta già decisa (o annullata) non può essere decisa una seconda volta. Se più volontari hanno richiesto la stessa figura, confermarne uno non rifiuta automaticamente gli altri — vanno smaltiti singolarmente (limite noto, vedi ADR-0025). La prenotazione diretta viene rifiutata con lo stesso tipo di controlli della richiesta del volontario: figura già confermata, o il volontario scelto ha già una propria richiesta pending/confermata per quel turno+data (in quel caso va approvata con `PATCH`, non duplicata).

## Schermata Turni del volontario (app Flutter)

Raggiungibile dalla home (card "Turni", visibile a chiunque abbia `shifts:read`) alla rotta `/shifts`. Quattro viste selezionabili in cima alla schermata:

- **Mese**: calendario mensile con un pallino colorato aggregato per turno sotto ogni giorno con turni (verde = almeno una figura libera, ambra = nessuna libera ma almeno una in attesa, grigio = al completo, colore del comitato pieno/ad anello = una propria figura confermata/in attesa); toccare un giorno apre sotto il dettaglio dei turni di quel giorno.
- **Settimana**: i 7 giorni della settimana corrente (lunedì-domenica), ciascuno con le proprie occorrenze o "Nessun turno".
- **Giorno**: un giorno alla volta, navigabile avanti/indietro.
- **Lista**: agenda scorrevole sui prossimi 30 giorni (solo i giorni con turni) — comoda per selezionare più figure su settimane diverse senza cambiare vista.

Un filtro Tutti/Liberi/Miei si applica a tutte e 4 le viste. Ogni turno è una card espandibile: chiusa mostra un riepilogo compatto ("N/4 libero", "Al completo" appena autista/leader/soccorritore sono confermati, o "Tua richiesta in attesa"/"Confermato per te" se il chiamante ha già una figura lì); aperta rivela le 4 righe-figura (icona, stato colorato, checkbox o badge). Chi ha anche `shifts:request` (il ruolo "Volontario emergenza" nel Comitato di Pavullo) vede una checkbox su ogni figura ancora prenotabile (libera, o in attesa ma non ancora confermata — mai su una figura già confermata o su una propria richiesta esistente); selezionarne una blocca (senza nasconderle) le checkbox delle altre 3 figure della stessa card, perché una persona tiene al più una figura per occorrenza. Una barra in fondo mostra il conteggio delle figure selezionate e invia la richiesta multipla (`POST /v1/shift-bookings/bulk`). Chi ha solo `shifts:read` vede lo stesso calendario in sola lettura, senza checkbox.

## Schermata Gestione turni (app Flutter)

Raggiungibile dalla home (card "Gestione turni", visibile a chi ha `shifts:write` e/o `shifts:configure`) alla rotta `/shifts/manage`. Tre tab:

- **Copertura**: le stesse 4 viste calendario del volontario, ma senza selezione multipla — ogni figura non ancora confermata mostra un bottone "Assegna" al posto della checkbox. Toccandolo si apre un selettore volontario con ricerca (stesso stile di ricerca di "Gestisci utenti"); scegliendo un nome si prenota direttamente quella figura per lui (`POST /v1/shift-bookings/direct`), già confermata. Rifiutato con un messaggio se il volontario scelto ha già una propria richiesta pending/confermata sullo stesso turno+data.
- **Richieste**: elenco di dettaglio delle richieste in sospeso (`GET /v1/shift-bookings/pending`), un badge sulla tab ne mostra il conteggio. Ogni riga mostra volontario, turno, data e figura, con bottoni Approva/Rifiuta (`PATCH /v1/shift-bookings/{id}`).
- **Turni-template**: elenco dei turni-template (quelli inattivi appaiono attenuati), righe che si espandono al tocco in un form di modifica (giorno della settimana, orario inizio/fine, nome, attivo/inattivo — sempre l'intero insieme di campi, sostituzione completa) e un pulsante "+" per crearne uno nuovo.

## Turni-template configurati oggi (Comitato di Pavullo)

Seed iniziale da `config/pavullo/shifts/templates.json`, applicato una sola volta all'avvio (se non ci sono già turni-template): giovedì sera, sabato nelle 3 fasce classiche, domenica nelle prime due. Il gestore turni può aggiungerne altri o modificare questi da qui in avanti — un riavvio del servizio non li resetta.
