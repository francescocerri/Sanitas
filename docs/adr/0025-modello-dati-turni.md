# 0025. Modello dati dei turni: turni-template ricorrenti + prenotazioni

Status: Accettata

## Contesto

Il servizio `shifts` aveva finora un'unica tabella piatta (`shifts`: `id`, `volunteer_id`, `date`/`start_time`/`end_time` come semplice testo, `status` libero) esplicitamente commentata come placeholder — serviva solo a validare la pipeline DB → API → CI → deploy, non il dominio reale (`docs/backlog.md`, voce "Progettazione del dominio reale"). Il servizio non ha mai avuto dati reali in produzione.

Il dominio richiesto: il gestore turni imposta turni ricorrenti (giorno della settimana + orario, es. "giovedì 20:00–08:00", "sabato 08:00–14:00/14:00–20:00/20:00–08:00"), anche parziali (non tutti i giorni devono avere turni configurati). I volontari di emergenza possono richiedere di prenotarsi su uno slot libero; la richiesta va approvata dal gestore turni. Il gestore può anche prenotare direttamente un volontario, senza passare da un'approvazione. Un volontario può prenotarsi su più giorni in un colpo solo (attività separata, voce 7 del backlog — bulk atomico).

## Decisione

**Due tabelle, non una**: `shift_templates` (turno ricorrente: `weekday`, `start_time`, `end_time`, `label` cosmetico, `active`) e `bookings` (una prenotazione: `template_id`, `volunteer_id`, `date`, `start_time`/`end_time` **copiati dal template al momento della richiesta**, `status`, `decided_by`/`decided_at`). La copia degli orari nel booking (non un riferimento live al template) è deliberata: una modifica futura al template non deve alterare prenotazioni già fatte.

**`shift_templates` è dati applicativi (DB), non config statica**: a differenza di `roles.json`/`email.json` (config per-comitato in `config/<slug>/`, versionata nel repo), i turni-template si impostano **da app** — è un'azione del gestore turni, non una modifica di configurazione che richiede un redeploy. Coerente con `AutoMigrate`/GORM già in uso (ADR-0019/0020).

**`Weekday` segue `time.Weekday`** di Go (0=domenica...6=sabato): riusa la conversione stdlib per generare le occorrenze concrete su un intervallo di date (prossima voce di backlog), niente schema di numerazione inventato.

**`Date` è un tipo DATE reale** (`time.Time`, colonna `date`), non più testo come nel vecchio modello: il vecchio placeholder usava TEXT deliberatamente per rimandare il problema del type-mapping — ora che il dominio è reale, non c'è più motivo di rimandarlo. `start_time`/`end_time` restano invece stringhe `"HH:MM"`: sono orari del giorno, non timestamp, e `time.Time` costringerebbe a una data fittizia/gestione fuso orario che non serve qui.

**Vocabolario di dominio in inglese**: i valori di `status` (`pending`/`confirmed`/`rejected`/`cancelled`) sono in inglese, non italiano — chiude un'inconsistenza che `CLAUDE.md` lasciava esplicitamente aperta ("valori di vocabolario di dominio... un'inconsistenza nota e intenzionalmente aperta"): d'ora in poi **tutto** il vocabolario di dominio (nomi di campo JSON inclusi, già `volunteer_id` e non `volontario_id`) è in inglese, coerente con codice/commenti/messaggi d'errore già in inglese. Resta in italiano solo la documentazione di progetto (questo file, `docs/backlog.md`, `CLAUDE.md`) e i dati di configurazione scritti dal comitato (es. `display_name` in `roles.json`), che non sono vocabolario di dominio.

**Vincoli via SQL raw dopo `AutoMigrate`**, stesso stile già in uso per la FK verso `registry.users`:
- FK `bookings.template_id → shift_templates.id` — stesso schema `shifts`, ma via SQL raw per coerenza con l'unico altro FK già presente (niente associazione GORM: non serve eager-load di un template attraverso un booking).
- FK `bookings.volunteer_id → registry.users.id` — cross-schema, invariata rispetto a prima.
- Indice unico parziale `idx_bookings_confirmed_slot` su `(template_id, date)` filtrato a `status = 'confirmed'`: un solo turno confermato per slot+data. Più richieste `pending` sullo stesso slot restano permesse finché una non viene confermata — quali altre rifiutare a quel punto è logica applicativa (voce di backlog "approvazione/rifiuto"), non dello schema.

**Nuovo permesso `shifts:configure`**, distinto da `shifts:write`: decide giorni/orari dei turni-template. Assegnato in `config/pavullo/registry/roles.json` **solo** al ruolo `shift_manager` — non a `president`, a differenza di `shifts:write` che oggi entrambi hanno. Scelta esplicita del comitato, non un default. `shifts:write` resta per approvare/rifiutare prenotazioni e per la prenotazione diretta del gestore. Per "chi può richiedere una prenotazione" (i volontari di emergenza) non si introduce un permesso dedicato: si verifica `claims.Roles` (già presente nel JWT, finora inutilizzato lato `shifts`) per il ruolo `emergency_volunteer`.

**`Active` senza default DB**: per un campo booleano, GORM omette dall'INSERT un valore Go zero (`false`) e lascerebbe comunque applicare un eventuale default DB — un chiamante che volesse esplicitamente `false` verrebbe ignorato. Il repository impone `Active = true` alla creazione (stesso principio già usato per `Booking.Status`), la disattivazione di un template esistente è compito della prossima voce di backlog (endpoint di update).

## Conseguenze

- La vecchia tabella `shifts` viene droppata in `Migrate` (`DROP TABLE IF EXISTS shifts`): non ha mai avuto dati reali, non serve una migrazione dati.
- Le 3 route esistenti (`GET/POST /v1/shifts`, `GET /v1/shifts/{id}`) sono rimosse insieme ai relativi handler: fino alle prossime voci di backlog (gestione turni-template, calendario/copertura, richiesta/approvazione/prenotazione diretta), `shifts` espone solo `/healthz` e `/docs/`. Non è una regressione: quelle route erano anch'esse placeholder, mai usate da un client reale.
- `internal/shift.Repository` guadagna `CreateTemplate`/`GetTemplate`/`ListTemplates` e `CreateBooking`/`GetBooking`/`ListBookings` — stesso mirror di Create/Get/List che il vecchio modello aveva per un'unica entità, ora per due. Update/transizioni di stato non ancora presenti: arrivano con gli endpoint che li useranno.
- `registry`: nuova costante `PermShiftsConfigure`, aggiunta a `AllPermissions`/`knownPermissions` — il bootstrap admin la eredita come tutte le altre.

## Aggiornamento (voce di backlog 4, "richiesta di prenotazione")

Sopra si era ipotizzato di verificare "chi può richiedere una prenotazione" leggendo `claims.Roles` per il ruolo `emergency_volunteer`, senza un permesso dedicato. Sostituito in fase di implementazione: nuovo permesso `PermShiftsRequest = "shifts:request"`, assegnato solo a `emergency_volunteer` — stesso meccanismo di autorizzazione (permessi, non ruoli) usato ovunque nel sistema, scelta esplicita del comitato per non introdurre un secondo asse di controllo. `shifts` continua quindi a non guardare mai `claims.Roles`.

## Aggiornamento (voce di backlog 10, dimensione "figura")

Emerso solo durante la verifica della schermata Turni del volontario: un turno non è un blocco libero/occupato unico, ma composto da **4 figure indipendenti** — autista, leader (capo equipaggio), soccorritore, osservatore — ciascuna prenotabile a sé, per un massimo di 4 persone per turno. Sia la richiesta del volontario sia la prenotazione diretta del gestore devono quindi specificare **per quale figura**.

Le 4 figure sono un **enum fisso nel codice**, non configurabile per comitato (scelta esplicita, confermata con l'utente): composizione di un equipaggio CRI, un dato di dominio comune, non uno specifico di Pavullo — a differenza dei turni-template (orari/giorni), che restano per-comitato.

`Booking` guadagna `Role BookingRole` (`driver`/`leader`/`rescuer`/`observer`, colonna NOT NULL, nessun default — sempre esplicita come `VolunteerID`). L'indice unico parziale sui confermati passa da `(template_id, date)` a `(template_id, date, role)`: più figure diverse sullo stesso slot possono ora essere confermate indipendentemente. `Occurrence` perde lo `Status`/`MyBookingStatus` a livello radice, guadagna `Roles []RoleCoverage` (sempre 4 elementi, uno per figura) — modifica di risposta non retrocompatibile, accettabile perché il servizio non ha mai avuto dati reali in produzione (stesso ragionamento già seguito per gli `AutoMigrate` precedenti).

**Vincolo di dominio**: una persona tiene al più una figura per occorrenza — non può prenotarsi due volte sullo stesso template+data, indipendentemente dalla figura. Applicato a 3 livelli: `HasBookingForVolunteer` resta deliberatamente senza filtro su `role` (controlla qualunque prenotazione pending/confermata del volontario su quel template+data); il controllo duplicati nel payload bulk è sulla coppia `(template_id, date)`, non sulla tripla con `role` — altrimenti due voci con figure diverse sullo stesso slot nella stessa richiesta bulk passerebbero la validazione singola (ciascuna guarda solo il DB, non le altre voci della stessa richiesta) e verrebbero entrambe inserite, violando il vincolo; lato UI, selezionare una figura su un'occorrenza disabilita (non nasconde) le checkbox delle altre 3 sulla stessa card.

**Bug di classe generica scoperto qui**: un endpoint bulk che "valida tutto contro il DB, poi inserisce tutto in una transazione" non vede gli elementi in-flight della stessa richiesta durante la validazione per-elemento — footgun da tenere a mente per qualunque futuro endpoint bulk in questo codebase.

## Aggiornamento (voce di backlog 11, "al completo" senza l'osservatore)

Emerso durante la verifica della schermata del gestore turni: un turno si considera "al completo" quando le **3 figure operative** — autista, leader, soccorritore — sono confermate; l'osservatore è facoltativo e la sua assenza non impedisce lo stato "completo" (`ShiftOccurrence.isComplete` lato Flutter, `requiredRolesForCompletion = {driver, leader, rescuer}`). Puramente una regola di presentazione lato client (il pallino del calendario mese e il badge della card usano questa soglia al posto di "tutte e 4 confermate"): non tocca lo schema né gli endpoint, `RoleCoverage`/`Occurrence` restano quelli descritti sopra, con tutte e 4 le figure sempre esposte. Lo stesso `isComplete` guida anche il filtro "Liberi" della schermata del volontario (voce 10): un turno già al completo non compare più lì, nemmeno con l'osservatore ancora libero.

## Aggiornamento (voce di backlog 10, visibilità di chi è confermato)

Richiesto esplicitamente dall'utente durante la verifica: chiunque abbia `shifts:read` (non solo il gestore turni) deve poter vedere il nome del volontario confermato su ciascuna figura — una richiesta ancora in attesa resta invece anonima come oggi, solo con l'indicazione "in attesa", finché non viene decisa.

`RoleCoverage` guadagna `VolunteerID *string` (`omitempty`), popolato in `Repository.ListOccurrences` SOLO quando quella figura ha una prenotazione confermata — mai per una pending, indipendentemente da chi chiama (un volontario non vede il nome di chi ha fatto una richiesta pending diversa dalla propria, nemmeno se il turno poi risulta al completo). Nessun nuovo permesso: la visibilità segue semplicemente `shifts:read`, già richiesto per leggere l'endpoint.

Il servizio `shifts` non conosce username/email (dominio di `registry`): il campo esposto è un id, risolto **lato client** con lo stesso `GET /v1/users` già usato da "Gestisci utenti" e dalla tab Richieste del gestore — `OccurrenceCard` (riusata identica da volontario e gestore) osserva `usersProvider` e mostra il nome se risolvibile, altrimenti ricade sul testo generico "Completo" (mai un id grezzo a schermo).

## Aggiornamento (stato aggregato del turno, non solo per figura)

Bug segnalato dall'utente in verifica: un turno con 3 figure confermate e una in attesa poteva comparire ancora come "libero" sul calendario, perché lo stato aggregato non era calcolato in un unico punto — `occurrenceCardColor` (pallino Mese, in `shift_status_style.dart`) e il badge della card chiusa (calcolato ad-hoc dentro `occurrence_card.dart`, `isComplete`/`openCount`) usavano due logiche indipendenti che potevano divergere, e nessuna delle due segnalava mai esplicitamente "in attesa" a livello di turno.

**Deciso esplicitamente con l'utente**: un'unica funzione (`occurrenceStatus`/equivalente in `shift_status_style.dart`) calcola lo stato aggregato di un'occorrenza, riusata identica da pallino Mese e badge di Settimana/Giorno/Lista/Mese. Priorità, dalla più alta:

1. **Propria figura** (in attesa o confermata) — vince sempre sullo stato reale del turno, anche se lo nasconde agli altri. Scelta esplicita: "è il mio turno" resta il segnale più utile per chi guarda; lo stato oggettivo (completo/in attesa/libero) resta comunque disponibile aprendo la card, niente secondo indicatore separato per ora.
2. **Completo** — autista/leader/soccorritore tutti confermati (invariato, `isComplete`).
3. **In attesa** — non completo, e almeno una delle 3 figure operative è in attesa. Vince **sempre** su "libero", anche con un'altra figura operativa o l'osservatore ancora liberi — è la correzione del bug: prima "libero" vinceva se restava anche un solo posto aperto altrove, nascondendo che c'era una decisione da prendere.
4. **Libero** — nessuno dei casi sopra.

Nessuna modifica allo schema o alla risposta di `GET /v1/shift-occurrences`: puro calcolo di presentazione lato client sugli stessi campi già esposti (`status`/`my_booking_status` per figura), stesso principio già seguito per `isComplete`.

**Rifinitura immediata, stessa sessione**: il riepilogo testuale della card (non il pallino, che resta un colore senza testo) non deve nascondere quante figure restano libere solo perché lo stato aggregato è "in attesa" o "mio" (confermato/in attesa per me) — richiesto esplicitamente dall'utente. Il conteggio ("N/3 libero") compare quindi anche in quei due casi, non solo in `free`, e conta **solo le 3 figure operative** (mai l'osservatore, coerente con `isComplete`/`requiredRolesForCompletion`) — un "3/3 libero" o "1/3 libero" con l'osservatore ancora libero o meno non deve cambiare, richiesto esplicitamente dall'utente. Quando il conteggio operativo è 0 (nessuna figura operativa libera, solo eventualmente l'osservatore) si ricade sul testo semplice ("In attesa"/"Confermato"), niente "0/3 libero".

## Aggiornamento (bug: checkbox/"Assegna" offerti anche su un'occorrenza passata)

Bug segnalato dall'utente in verifica ("un volontario di emergenza non riesce a proporsi per un turno"): navigando indietro nel tempo nelle viste Giorno/Settimana/Mese (che, a differenza di Lista, non sono limitate ai prossimi 30 giorni), un'occorrenza già passata mostrava comunque checkbox/bottone "Assegna" su ogni figura ancora libera. Il backend rifiuta sempre una prenotazione su una data passata (`parseAndValidateBookingDate`, sia per la richiesta del volontario sia per l'assegnazione diretta del gestore), ma nessuna delle due azioni lo controllava lato client prima di offrirsi: selezionare e inviare produceva solo il messaggio generico di fallita richiesta, senza spiegare perché.

`OccurrenceCard` calcola ora `isPast` (data dell'occorrenza prima di oggi, confronto a mezzanotte locale con `dateOnly`, coerente con `date.Before(today)` lato backend) e lo propaga a `_RoleRow`: checkbox e bottone "Assegna" non compaiono mai su un'occorrenza passata, indipendentemente da `isBookable`/permessi. Nessuna modifica al backend (la validazione era già corretta) né allo schema — puro allineamento lato client a un vincolo che esisteva già.

## Aggiornamento (stato operativo: completo/ridotto/chiuso)

Richiesta esplicita dell'utente: un turno passato deve registrare un **esito operativo** — si è svolto o no, e con quale equipaggio — pensato soprattutto per statistiche future, distinto dallo stato aggregato libero/in attesa/completo (che ha senso solo finché il turno è ancora prenotabile).

**Regola di dominio** (confermata dall'utente): autista e leader devono essere **entrambi** confermati perché il turno si sia svolto — il soccorritore da solo non supplisce mai a un autista o un leader mancante, distingue solo "completo" da "ridotto".
- `complete`: autista, leader, soccorritore tutti confermati.
- `reduced`: solo autista e leader confermati (soccorritore no) — etichetta/colore proprio, distinto sia da "completo" sia da "chiuso" (scelta esplicita dell'utente, non un riuso di "completo").
- `closed`: qualunque altra combinazione (manca l'autista o il leader) — il turno non si è svolto.

**Calcolato lato backend** (`shifts`), non lato client — scelta esplicita dell'utente, coerente con l'obiettivo statistico futuro: riusabile da qualunque consumatore senza duplicare altrove "cos'è passato". `Occurrence` guadagna `OperationalStatus *OperationalStatus` (`json:"operational_status,omitempty"`), popolato da `Repository.ListOccurrences` SOLO quando `date` è prima di "oggi" (stesso confine `date.Before(today)` già usato da `parseAndValidateBookingDate`, replicato in `internal/shift` perché pacchetto diverso da `internal/httpapi`) — per un'occorrenza di oggi o futura resta `nil`, lo stato aggregato esistente rimane l'unico segnale valido. Nuova funzione pura `operationalStatusFor(roles []RoleCoverage) OperationalStatus`, chiamata solo quando serve.

Lato Flutter, `occurrence.operationalStatus` (quando non nullo) ha priorità assoluta su "mio"/stato aggregato sia nel colore del pallino Mese (`occurrenceCardColor`) sia nel badge di riepilogo (`occurrenceSummaryBadge`, nuova funzione che centralizza sfondo/testo/icona del badge — sostituisce l'uso diretto di `occurrenceSummaryLabel` dentro `occurrence_card.dart`): l'esito oggettivo di un turno concluso conta più di chi ci fosse. "Chiuso" è l'unico caso con un'icona (una X) — sul pallino Mese sostituisce interamente il cerchio colorato (non lo sovrappone: un cerchio 6px più un'icona sopra risultava illeggibile, corretto durante la verifica del mockup), sul badge della card compare come icona accanto al testo.

**Limite noto, non affrontato ora**: `ListOccurrences` genera occorrenze solo dai turni-template **attualmente attivi** (`ListActiveTemplates`, `WHERE active`) — se un template viene disattivato, la sua storia passata (incluso l'esito operativo) sparisce dal calendario insieme a lui. Non sollevato dall'utente per questa richiesta; da rivedere quando si costruiranno davvero le statistiche.
