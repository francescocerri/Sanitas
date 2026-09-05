package authclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testIssuer is a stand-in for anagrafica: serves a JWKS document and signs
// tokens with the matching private key, so authclient can be tested without
// spinning up the real service (a different Go module, ADR-0003).
type testIssuer struct {
	server *httptest.Server
	priv   *rsa.PrivateKey
	kid    string
}

func newTestIssuer(t *testing.T) *testIssuer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	iss := &testIssuer{priv: priv, kid: "test-kid"}
	iss.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pub := priv.PublicKey
		body := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"use": "sig",
					"alg": "RS256",
					"kid": iss.kid,
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(bigEndianExponent(pub.E)),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(iss.server.Close)
	return iss
}

// bigEndianExponent mirrors anagrafica's user.jwt.go helper of the same
// name — same encoding, needed here only to build a realistic test JWKS.
func bigEndianExponent(e int) []byte {
	b := []byte{byte(e >> 16), byte(e >> 8), byte(e)}
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return b[i:]
}

func (iss *testIssuer) sign(t *testing.T, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = iss.kid
	signed, err := token.SignedString(iss.priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func validClaims() Claims {
	return Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Username: "mario",
	}
}

func TestVerify_AcceptsTokenSignedByCachedKey(t *testing.T) {
	iss := newTestIssuer(t)
	client := New(iss.server.URL)
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	token := iss.sign(t, validClaims())
	claims, err := client.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "user-1" || claims.Username != "mario" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerify_RefreshesOnUnknownKid(t *testing.T) {
	iss := newTestIssuer(t)
	client := New(iss.server.URL)
	// Deliberately no initial Refresh: Verify must fetch the JWKS itself on
	// the first unknown kid, so a token signed after startup (or before the
	// very first Refresh completes) still verifies.
	token := iss.sign(t, validClaims())

	if _, err := client.Verify(token); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerify_RejectsWrongSignature(t *testing.T) {
	iss := newTestIssuer(t)
	client := New(iss.server.URL)
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	otherPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims())
	token.Header["kid"] = iss.kid // claims a kid it wasn't actually signed with
	signed, err := token.SignedString(otherPriv)
	if err != nil {
		t.Fatalf("sign with other key: %v", err)
	}

	if _, err := client.Verify(signed); err == nil {
		t.Fatal("expected verification to fail for a token signed by a different key")
	}
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	iss := newTestIssuer(t)
	client := New(iss.server.URL)
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	expired := validClaims()
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	token := iss.sign(t, expired)

	if _, err := client.Verify(token); err == nil {
		t.Fatal("expected verification to fail for an expired token")
	}
}

func TestVerify_RejectsUnknownKidAfterRefresh(t *testing.T) {
	iss := newTestIssuer(t)
	client := New(iss.server.URL)
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims())
	token.Header["kid"] = "not-a-real-kid"
	signed, err := token.SignedString(iss.priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := client.Verify(signed); err == nil {
		t.Fatal("expected verification to fail for a kid the issuer never published")
	}
}
