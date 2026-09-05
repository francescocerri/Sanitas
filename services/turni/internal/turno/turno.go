package turno

// Turno is intentionally skeletal: it validates the DB -> API -> CI -> deploy
// pipeline, it is not the final domain design (assignment, conflicts,
// volunteer availability). Also the GORM model for the turni table (see
// Migrate) — gorm tags added, no change to the JSON shape.
//
// Data/OraInizio/OraFine are plain strings, not time.Time/a date type: the
// DB columns are TEXT precisely to sidestep date/time type-mapping for what
// is still a placeholder schema.
type Turno struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id,omitempty"`
	VolontarioID string `gorm:"column:volontario_id;type:uuid;not null" json:"volontario_id"`
	Data         string `gorm:"not null" json:"data"`
	OraInizio    string `gorm:"column:ora_inizio;not null" json:"ora_inizio"`
	OraFine      string `gorm:"column:ora_fine;not null" json:"ora_fine"`
	Stato        string `gorm:"not null;default:pianificato" json:"stato,omitempty"`
}

func (Turno) TableName() string { return "turni" }
