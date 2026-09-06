// Package authclient verifies JWTs issued by the registry service,
// fetching its public key from GET /.well-known/jwks.json instead of
// calling back to registry on every request — see docs/adr/0017.
package authclient

import "github.com/golang-jwt/jwt/v5"

// Claims mirrors registry/internal/user.Claims's JSON shape. shifts and
// registry are independent Go modules with no shared code (ADR-0003), so
// this can't be imported — it's kept in sync by hand, not a duplicate
// meant to diverge. Permissions is what requirePermission checks (see
// docs/adr/0018) — roles are mapped but unused by any logic here.
type Claims struct {
	jwt.RegisteredClaims
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
