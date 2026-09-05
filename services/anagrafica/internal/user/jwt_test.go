package user

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// writeTestKeyPair generates a fresh RSA key and writes it as a PEM file,
// exercising the same LoadKeyPair code path production uses.
func writeTestKeyPair(t *testing.T) *KeyPair {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}

	path := filepath.Join(t.TempDir(), "jwt_private_key.pem")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer f.Close()
	if err := pem.Encode(f, block); err != nil {
		t.Fatalf("write key: %v", err)
	}

	kp, err := LoadKeyPair(path)
	if err != nil {
		t.Fatalf("LoadKeyPair: %v", err)
	}
	return kp
}

func TestIssueAndVerifyToken(t *testing.T) {
	kp := writeTestKeyPair(t)

	u := User{ID: "11111111-1111-1111-1111-111111111111", Username: "mario", Roles: []string{"president"}, IsAdmin: true}
	token, err := kp.IssueToken(u)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	claims, err := kp.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != u.ID || claims.Username != u.Username || !claims.IsAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "president" {
		t.Fatalf("unexpected roles in claims: %v", claims.Roles)
	}
}

func TestVerify_RejectsTokenFromDifferentKey(t *testing.T) {
	kp1 := writeTestKeyPair(t)
	kp2 := writeTestKeyPair(t)

	token, err := kp1.IssueToken(User{ID: "u1"})
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if _, err := kp2.Verify(token); err == nil {
		t.Fatal("expected verification with a different key pair to fail")
	}
}

func TestJWKS_ContainsPublicKey(t *testing.T) {
	kp := writeTestKeyPair(t)

	jwks := kp.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("expected exactly one key in JWKS, got: %v", jwks)
	}
	if keys[0]["kty"] != "RSA" || keys[0]["alg"] != "RS256" {
		t.Fatalf("unexpected JWK: %+v", keys[0])
	}
	if keys[0]["kid"] != kp.kid {
		t.Fatalf("expected kid %s, got %v", kp.kid, keys[0]["kid"])
	}
}
