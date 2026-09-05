package user

import "golang.org/x/crypto/bcrypt"

// HashPassword and VerifyPassword use bcrypt: self-contained (cost factor
// and salt are encoded into the returned hash, nothing to manage
// separately) and still fully accepted (OWASP's second choice after
// argon2id, not deprecated) — proportionate to this project's scale
// instead of hand-rolling argon2id's PHC string encoding. See docs/adr/0013.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
