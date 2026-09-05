package turno

// Turno is intentionally skeletal: it validates the DB -> API -> CI -> deploy
// pipeline, it is not the final domain design (assignment, conflicts,
// volunteer availability).
//
// Data/OraInizio/OraFine are plain strings, not time.Time/a date type: the
// DB columns are TEXT (see migrations/0001_init.sql) precisely to sidestep
// pgx's date/time type-mapping for what is still a placeholder schema.
type Turno struct {
	ID           string `json:"id,omitempty"`
	VolontarioID string `json:"volontario_id"`
	Data         string `json:"data"`
	OraInizio    string `json:"ora_inizio"`
	OraFine      string `json:"ora_fine"`
	Stato        string `json:"stato,omitempty"`
}
