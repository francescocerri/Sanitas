// Package user holds the anagrafica domain: users, roles, invite tokens.
package user

import "time"

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role is seed data, not a fixed enum: role names are committee-specific
// (see docs/adr/0012), loaded from a config file at startup.
type Role struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}
