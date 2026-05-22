package xapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ActorHash returns a stable pseudonymous identifier for the actor (used in analytics.xapi_statements.actor_hash).
func ActorHash(email string, anonymize bool) string {
	norm := strings.ToLower(strings.TrimSpace(email))
	if anonymize {
		sum := sha256.Sum256([]byte(norm))
		return hex.EncodeToString(sum[:])
	}
	return norm
}

// ActorMbox returns the xAPI actor mbox value (mailto:...).
func ActorMbox(email string, anonymize bool) string {
	if anonymize {
		return fmt.Sprintf("mailto:%s@lextures.local", ActorHash(email, true))
	}
	return "mailto:" + strings.ToLower(strings.TrimSpace(email))
}
