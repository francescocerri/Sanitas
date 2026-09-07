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
