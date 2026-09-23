package setupglob

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
)

var ServerInSetup = false

// SetupToken must be provided to create the first admin, so whoever reaches
// a freshly deployed server first can't claim it. It is generated when setup
// routes are registered and printed to the server log.
var SetupToken = ""

// NewSetupToken generates, stores and returns a new setup token.
func NewSetupToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	SetupToken = hex.EncodeToString(b)
	return SetupToken, nil
}

// ValidSetupToken reports if t matches the current setup token.
func ValidSetupToken(t string) bool {
	if SetupToken == "" || t == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(t), []byte(SetupToken)) == 1
}
