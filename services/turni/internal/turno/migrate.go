package turno

import "gorm.io/gorm"

// Migrate creates the turni schema (if missing) and brings the table up to
// date via AutoMigrate, then adds the FK into anagrafica.users — a
// cross-schema, cross-service reference AutoMigrate can't derive on its
// own (no GORM association: anagrafica.users is a different Go module's
// model, see docs/adr/0019). Shared by production startup
// (cmd/server/main.go) and the test harness (internal/testdb) so schema
// provisioning can never drift between the two.
//
// Wrap this call in retry at the caller: anagrafica.users may not exist
// yet if anagrafica hasn't finished its own AutoMigrate — see
// cmd/server/main.go's createSchemaWithRetry.
func Migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS turni").Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&Turno{}); err != nil {
		return err
	}
	return db.Exec(`
		DO $$ BEGIN
			ALTER TABLE turni ADD CONSTRAINT fk_turni_volontario
				FOREIGN KEY (volontario_id) REFERENCES anagrafica.users(id);
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;`).Error
}
