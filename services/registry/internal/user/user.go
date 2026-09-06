// Package user holds the registry domain: users, roles, invite tokens.
package user

import "time"

// User is both the GORM model for the users table and the API-facing
// shape (see createUserResponse/handleMe) — Roles/Permissions aren't
// columns (gorm:"-"): they're resolved by joining user_roles/roles, see
// Repository.populateRolesAndPermissions.
type User struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash *string   `gorm:"column:password_hash" json:"-"`
	Roles        []string  `gorm:"-" json:"roles"`
	Permissions  []string  `gorm:"-" json:"permissions"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (User) TableName() string { return "users" }

// Role is seed data, not a fixed enum: role names are committee-specific
// (see docs/adr/0012), loaded from a config file at startup. Permissions is
// a fixed, code-known vocabulary (see permissions.go) — which roles get
// which is the per-committee part, not the permission names — docs/adr/0018.
type Role struct {
	ID          string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Slug        string      `gorm:"uniqueIndex;not null" json:"slug"`
	DisplayName string      `gorm:"column:display_name;not null" json:"display_name"`
	Permissions StringArray `gorm:"type:text[];not null;default:'{}'" json:"-"`
}

func (Role) TableName() string { return "roles" }

// UserRole is the users<->roles join table. Modeled explicitly (not via
// GORM's many2many) so AssignRoles/GetRolesForUser keep the same explicit
// queries they had under pgx — see docs/adr/0019.
type UserRole struct {
	UserID string `gorm:"type:uuid;primaryKey;column:user_id"`
	RoleID string `gorm:"type:uuid;primaryKey;column:role_id"`
}

func (UserRole) TableName() string { return "user_roles" }

// Token is a generic single-use token row: purpose discriminates ("invite",
// "refresh", ... — see docs/adr/0016) instead of duplicating a
// near-identical table per use case.
type Token struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string     `gorm:"type:uuid;not null;column:user_id"`
	Purpose   string     `gorm:"not null;default:invite"`
	TokenHash string     `gorm:"uniqueIndex;not null;column:token_hash"`
	ExpiresAt time.Time  `gorm:"not null;column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
}

func (Token) TableName() string { return "tokens" }
