package shift

import "gorm.io/gorm"

// Migrate creates the shifts schema (if missing), brings shift_templates and
// bookings up to date via AutoMigrate, drops the old placeholder shifts
// table (never held real data — see docs/adr/0025-modello-dati-turni.md),
// then adds the FKs and the partial unique index AutoMigrate can't derive
// on its own. Shared by production startup (cmd/server/main.go) and the
// test harness (internal/testdb) so schema provisioning can never drift
// between the two.
//
// Wrap this call in retry at the caller: registry.users may not exist
// yet if registry hasn't finished its own AutoMigrate — see
// cmd/server/main.go's createSchemaWithRetry.
func Migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS shifts").Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&ShiftTemplate{}, &Booking{}); err != nil {
		return err
	}
	// Placeholder table from before the real domain model (ADR-0025) — the
	// service was explicitly skeletal and never carried real data, so this
	// is a one-time drop, not a data migration.
	if err := db.Exec("DROP TABLE IF EXISTS shifts").Error; err != nil {
		return err
	}
	// template_id: same schema, but via raw SQL like the volunteer FK below
	// for one consistent style — no GORM association needed, nothing here
	// eager-loads a template through a booking.
	if err := db.Exec(`
		DO $$ BEGIN
			ALTER TABLE bookings ADD CONSTRAINT fk_bookings_template
				FOREIGN KEY (template_id) REFERENCES shift_templates(id);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;`).Error; err != nil {
		return err
	}
	// volunteer_id: cross-schema, cross-service reference AutoMigrate can't
	// derive on its own (no GORM association: registry.users is a different
	// Go module's model, see docs/adr/0019).
	if err := db.Exec(`
		DO $$ BEGIN
			ALTER TABLE bookings ADD CONSTRAINT fk_bookings_volunteer
				FOREIGN KEY (volunteer_id) REFERENCES registry.users(id);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;`).Error; err != nil {
		return err
	}
	// Only one confirmed booking per template+date+role: multiple pending
	// requests for the same slot+role stay allowed until one is confirmed
	// (deciding which others to reject at that point is application logic,
	// not the schema's job). Was (template_id, date) before roles existed
	// (see docs/adr/0025-modello-dati-turni.md "Aggiornamento") — drop and
	// recreate rather than an ALTER, no real deployment has ever carried
	// data under the old shape.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_bookings_confirmed_slot`).Error; err != nil {
		return err
	}
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_confirmed_slot
			ON bookings (template_id, date, role) WHERE status = 'confirmed'`).Error
}
