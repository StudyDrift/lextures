package vcsigning

import (
	"time"
)

// SignTranscriptCredential wraps an official transcript subject in a W3C VC with Ed25519 proof (T08).
func SignTranscriptCredential(subject map[string]any, issuerName string, key KeyMaterial, issuedAt time.Time) (map[string]any, error) {
	unsigned := map[string]any{
		"@context": []string{vcContext},
		"type":     []string{"VerifiableCredential", "OfficialTranscriptCredential"},
		"issuer": map[string]any{
			"id":   key.DID,
			"name": issuerName,
		},
		"issuanceDate":      issuedAt.UTC().Format(time.RFC3339),
		"credentialSubject": subject,
	}
	return signUnsigned(unsigned, key, issuedAt)
}
