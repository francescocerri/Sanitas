// Package authclient verifies JWTs issued by the anagrafica service,
// fetching its public key from GET /.well-known/jwks.json instead of
// calling back to anagrafica on every request — see docs/adr/0017.
package authclient

import "github.com/golang-jwt/jwt/v5"

// Claims mirrors anagrafica/internal/user.Claims's JSON shape. turni and
// anagrafica are independent Go modules with no shared code (ADR-0003), so
// this can't be imported — it's kept in sync by hand, not a duplicate
// meant to diverge. roles/IsAdmin aren't used by any logic here yet (this
// package only authenticates, it doesn't authorize — see docs/adr/0017),
// but are mapped now so a future per-role check doesn't need this struct
// touched again.
type Claims struct {
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	IsAdmin  bool     `json:"is_admin"`
}
