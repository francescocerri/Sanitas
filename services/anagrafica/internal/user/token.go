package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateToken returns a random URL-safe token and its hash. Only the hash
// is ever persisted (invite_tokens.token_hash) — the raw token exists only
// in the invite URL handed back to the caller, so a DB read alone can never
// reveal a valid token.
func GenerateToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw = hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
