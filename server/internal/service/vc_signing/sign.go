package vcsigning

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	proofType           = "Ed25519Signature2020"
	proofPurpose        = "assertionMethod"
	vcContext           = "https://www.w3.org/2018/credentials/v1"
	clrContext          = "https://purl.imsglobal.org/spec/clr/v2p0/context.json"
	verificationMethodID = "#key-1"
)

// KeyMaterial holds an Ed25519 signing key and derived did:web identifier.
type KeyMaterial struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	DID        string
}

// GenerateKey creates a new Ed25519 key pair and did:web identifier from the API public origin.
func GenerateKey(apiOrigin string) (KeyMaterial, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyMaterial{}, err
	}
	did, err := DIDWebFromOrigin(apiOrigin)
	if err != nil {
		return KeyMaterial{}, err
	}
	return KeyMaterial{PrivateKey: priv, PublicKey: pub, DID: did}, nil
}

// KeyFromPrivateSeed loads a 32-byte seed from base64 and derives the key pair.
func KeyFromPrivateSeed(seedB64, apiOrigin string) (KeyMaterial, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(seedB64))
	if err != nil {
		return KeyMaterial{}, fmt.Errorf("decode signing seed: %w", err)
	}
	if len(raw) != ed25519.SeedSize {
		return KeyMaterial{}, fmt.Errorf("signing seed must be %d bytes", ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(raw)
	pub := priv.Public().(ed25519.PublicKey)
	did, err := DIDWebFromOrigin(apiOrigin)
	if err != nil {
		return KeyMaterial{}, err
	}
	return KeyMaterial{PrivateKey: priv, PublicKey: pub, DID: did}, nil
}

// DIDWebFromOrigin builds did:web from an http(s) origin host (percent-encoded when needed).
func DIDWebFromOrigin(origin string) (string, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(origin), "/"))
	if err != nil {
		return "", err
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("invalid origin for did:web: %q", origin)
	}
	return "did:web:" + strings.ReplaceAll(host, ":", "%3A"), nil
}

// DIDDocument returns a resolvable DID document for the institution signing key.
func (k KeyMaterial) DIDDocument() map[string]any {
	return map[string]any{
		"@context": "https://www.w3.org/ns/did/v1",
		"id":       k.DID,
		"verificationMethod": []map[string]any{
			{
				"id":           k.DID + verificationMethodID,
				"type":         "Ed25519VerificationKey2020",
				"controller":   k.DID,
				"publicKeyBase58": base64.RawURLEncoding.EncodeToString(k.PublicKey),
			},
		},
		"assertionMethod": []string{k.DID + verificationMethodID},
	}
}

// SignCredential wraps clrSubject in a W3C VC and attaches an Ed25519 proof.
func SignCredential(clrSubject map[string]any, issuerName string, key KeyMaterial, issuedAt time.Time) (map[string]any, error) {
	unsigned := map[string]any{
		"@context": []string{vcContext, clrContext},
		"type":     []string{"VerifiableCredential", "ClrCredential"},
		"issuer": map[string]any{
			"id":   key.DID,
			"name": issuerName,
		},
		"issuanceDate": issuedAt.UTC().Format(time.RFC3339),
		"credentialSubject": clrSubject,
	}
	canonical, err := canonicalJSON(unsigned)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(canonical)
	sig := ed25519.Sign(key.PrivateKey, digest[:])
	unsigned["proof"] = map[string]any{
		"type":               proofType,
		"created":            issuedAt.UTC().Format(time.RFC3339),
		"verificationMethod": key.DID + verificationMethodID,
		"proofPurpose":       proofPurpose,
		"proofValue":         base64.StdEncoding.EncodeToString(sig),
	}
	return unsigned, nil
}

// VerifyCredential checks the Ed25519 proof on a signed VC using the institution public key.
func VerifyCredential(vc map[string]any, pub ed25519.PublicKey) (bool, error) {
	proofRaw, ok := vc["proof"]
	if !ok {
		return false, fmt.Errorf("missing proof")
	}
	proof, ok := proofRaw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("invalid proof")
	}
	proofValue, _ := proof["proofValue"].(string)
	if strings.TrimSpace(proofValue) == "" {
		return false, fmt.Errorf("missing proofValue")
	}
	sig, err := base64.StdEncoding.DecodeString(proofValue)
	if err != nil {
		return false, err
	}
	unsigned := cloneWithoutProof(vc)
	canonical, err := canonicalJSON(unsigned)
	if err != nil {
		return false, err
	}
	digest := sha256.Sum256(canonical)
	if !ed25519.Verify(pub, digest[:], sig) {
		return false, nil
	}
	return true, nil
}

func cloneWithoutProof(vc map[string]any) map[string]any {
	out := make(map[string]any, len(vc)-1)
	for k, v := range vc {
		if k == "proof" {
			continue
		}
		out[k] = v
	}
	return out
}

func canonicalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
