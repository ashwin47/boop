// Package auth generates and hashes secrets and extracts bearer tokens.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/chrisgregori/boop/server/internal/ids"
)

// Secret prefixes.
const (
	PrefixProjectKey = "boop_proj"
	PrefixDevice     = "boop_dev"
	PrefixPairing    = "pair"
)

// NewSecret returns a new random secret with the given prefix (160 bits of entropy).
func NewSecret(prefix string) string {
	return prefix + "_" + ids.Random(20)
}

// Hash returns the hex SHA-256 of a secret. Secrets are high-entropy random
// strings, so a plain hash (not a slow KDF) is appropriate for storage.
func Hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// Equal compares two hashes in constant time.
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Bearer extracts the bearer token from r's Authorization header, or "".
func Bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

// HasPrefix reports whether secret starts with prefix + "_".
func HasPrefix(secret, prefix string) bool {
	return strings.HasPrefix(secret, prefix+"_")
}
