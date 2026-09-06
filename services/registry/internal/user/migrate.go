package user

import "gorm.io/gorm"

// Migrate creates the registry schema (if missing) and brings the tables
// up to date via AutoMigrate — replaces the SQL init script this service
// used to rely on (docker-entrypoint-initdb.d only ever ran once, at first
// container creation — see docs/adr/0019, superseding part of docs/adr/0005).
// Shared by production startup (cmd/server/main.go) and the test harness
// (internal/testdb) so schema provisioning can never drift between the two.
func Migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS registry").Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &UserRole{}, &Token{}); err != nil {
		return err
	}
	return ensureForeignKeys(db)
}

// ensureForeignKeys adds the FK constraints AutoMigrate doesn't create on
// its own: UserRole/Token reference other rows by plain id fields, not
// GORM associations, so AutoMigrate has no relation to derive a constraint
// from. Idempotent: ADD CONSTRAINT has no IF NOT EXISTS in Postgres, so
// each is wrapped to tolerate already existing — safe to run on every
// startup, same as AutoMigrate itself.
func ensureForeignKeys(db *gorm.DB) error {
	constraints := []string{
		`ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`,
		`ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE`,
		`ALTER TABLE tokens ADD CONSTRAINT fk_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`,
	}
	for _, stmt := range constraints {
		guarded := `DO $$ BEGIN ` + stmt + `; EXCEPTION WHEN duplicate_object THEN NULL; END $$;`
		if err := db.Exec(guarded).Error; err != nil {
			return err
		}
	}
	return nil
}
