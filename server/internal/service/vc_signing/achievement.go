package vcsigning

import (
	"time"
)

const obContext = "https://purl.imsglobal.org/spec/ob/v3p0/context.json"

// SignAchievementCredential wraps an Open Badges 3.0 subject in a W3C VC with Ed25519 proof.
func SignAchievementCredential(subject map[string]any, issuerName string, key KeyMaterial, issuedAt time.Time) (map[string]any, error) {
	unsigned := map[string]any{
		"@context": []string{vcContext, obContext},
		"type":     []string{"VerifiableCredential", "OpenBadgeCredential"},
		"issuer": map[string]any{
			"id":   key.DID,
			"name": issuerName,
			"type": []string{"Profile"},
		},
		"issuanceDate":      issuedAt.UTC().Format(time.RFC3339),
		"credentialSubject": subject,
	}
	return signUnsigned(unsigned, key, issuedAt)
}