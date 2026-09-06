package shift

// Shift is intentionally skeletal: it validates the DB -> API -> CI -> deploy
// pipeline, it is not the final domain design (assignment, conflicts,
// volunteer availability). Also the GORM model for the shifts table (see
// Migrate) — gorm tags added, no change to the JSON shape.
//
// Date/StartTime/EndTime are plain strings, not time.Time/a date type: the
// DB columns are TEXT precisely to sidestep date/time type-mapping for what
// is still a placeholder schema.
type Shift struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id,omitempty"`
	VolunteerID string `gorm:"column:volunteer_id;type:uuid;not null" json:"volunteer_id"`
	Date        string `gorm:"not null" json:"date"`
	StartTime   string `gorm:"column:start_time;not null" json:"start_time"`
	EndTime     string `gorm:"column:end_time;not null" json:"end_time"`
	Status      string `gorm:"not null;default:planned" json:"status,omitempty"`
}

func (Shift) TableName() string { return "shifts" }
