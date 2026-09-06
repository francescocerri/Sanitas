package authclient

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// minRefreshInterval bounds how often an unknown kid triggers a refetch of
// the JWKS — without it, a burst of requests carrying a bogus/unknown kid
// would each hammer registry.
const minRefreshInterval = 60 * time.Second

// Client verifies JWTs against registry's public key(s), fetched once and
// cached — see docs/adr/0017. Safe for concurrent use.
type Client struct {
	jwksURL    string
	httpClient *http.Client

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
}

func New(jwksURL string) *Client {
	return &Client{
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		keys:       map[string]*rsa.PublicKey{},
	}
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// Refresh fetches and replaces the cached key set. Called once at startup
// (see cmd/server/main.go) and again, lazily, whenever Verify sees a kid it
// doesn't recognize — that's how a key rotated on registry's side gets
// picked up here without restarting shifts.
func (c *Client) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("authclient: build jwks request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authclient: fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authclient: fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var parsed jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("authclient: decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(parsed.Keys))
	for _, k := range parsed.Keys {
		pub, err := parseRSAPublicKey(k)
		if err != nil {
			return fmt.Errorf("authclient: parse key %q: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}

	c.mu.Lock()
	c.keys = keys
	c.lastRefresh = time.Now()
	c.mu.Unlock()
	return nil
}

// parseRSAPublicKey is the inverse of registry's user.KeyPair.JWKS(): n/e
// are base64url, unpadded, big-endian — same encoding, decoded back.
func parseRSAPublicKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

// Verify checks the token's RS256 signature against the cached key
// matching its kid (rejecting any other signing method) and returns its
// claims. jwt.ParseWithClaims also enforces standard claims (expiry, etc.).
func (c *Client) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, c.keyFunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (c *Client) keyFunc(token *jwt.Token) (any, error) {
	kid, _ := token.Header["kid"].(string)
	if key, ok := c.lookup(kid); ok {
		return key, nil
	}
	if c.shouldRefresh() {
		if err := c.Refresh(context.Background()); err != nil {
			return nil, fmt.Errorf("authclient: refresh after unknown kid: %w", err)
		}
	}
	if key, ok := c.lookup(kid); ok {
		return key, nil
	}
	return nil, fmt.Errorf("authclient: unknown kid %q", kid)
}

func (c *Client) lookup(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.keys[kid]
	return key, ok
}

func (c *Client) shouldRefresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastRefresh) >= minRefreshInterval
}
