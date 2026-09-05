package user

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTTL = 24 * time.Hour

// KeyPair holds the RSA key anagrafica signs JWTs with. Only this service
// ever sees the private key; everyone else verifies with the public half
// exposed at GET /.well-known/jwks.json — asymmetric on purpose, so no
// other service can forge a token even if its own secrets leak (see
// docs/adr/0013).
type KeyPair struct {
	private *rsa.PrivateKey
	kid     string
}

// LoadKeyPair reads a PEM-encoded RSA private key from path. The key is
// generated once (a documented one-time step, see README.md) and never
// committed — same handling as the database password.
//
// Accepts both PKCS#1 ("RSA PRIVATE KEY", the classic `openssl genrsa`
// output) and PKCS#8 ("PRIVATE KEY", what OpenSSL 3.x's `genrsa` produces
// by default nowadays) — asking whoever generates the key to remember
// which one their OpenSSL version picked isn't worth the friction.
func LoadKeyPair(path string) (*KeyPair, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("jwt: read private key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("jwt: no PEM block found in %s", path)
	}

	key, err := parseRSAPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse private key: %w", err)
	}
	return &KeyPair{private: key, kid: keyID(&key.PublicKey)}, nil
}

func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA key")
	}
	return key, nil
}

// Claims carried by every token this service issues.
type Claims struct {
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	IsAdmin  bool     `json:"is_admin"`
}

func (k *KeyPair) IssueToken(u User) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Username: u.Username,
		Roles:    u.Roles,
		IsAdmin:  u.IsAdmin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = k.kid
	return token.SignedString(k.private)
}

func (k *KeyPair) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return &k.private.PublicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// JWKS returns the public key in JWKS format (RFC 7517), for GET
// /.well-known/jwks.json — the only way another service or the frontend
// should obtain the key to verify tokens.
func (k *KeyPair) JWKS() map[string]any {
	pub := k.private.PublicKey
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": k.kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bigEndianExponent(pub.E)),
			},
		},
	}
}

// bigEndianExponent encodes the public exponent the way JWK expects it:
// big-endian, no leading zero bytes (65537 -> 3 bytes: 0x01 0x00 0x01).
func bigEndianExponent(e int) []byte {
	b := []byte{byte(e >> 16), byte(e >> 8), byte(e)}
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return b[i:]
}

func keyID(pub *rsa.PublicKey) string {
	sum := sha256.Sum256(pub.N.Bytes())
	return hex.EncodeToString(sum[:8])
}
