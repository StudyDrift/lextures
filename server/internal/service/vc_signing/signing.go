// Package vc_signing signs and verifies W3C Verifiable Credentials for CLR documents (plan 14.13).
package vc_signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	vcContextURL  = "https://www.w3.org/2018/credentials/v1"
	clrContextURL = "https://purl.imsglobal.org/spec/clr/v2p0/context.json"
)

// KeyMaterial is an Ed25519 signing keypair for institutional VC issuance.
type KeyMaterial struct {
	IssuerDID    string
	PublicKeyJWK json.RawMessage
	PrivateKey   ed25519.PrivateKey
}

// SignedCredential wraps a CLR credential subject and its JWT proof.
type SignedCredential struct {
	Credential map[string]any `json:"credential"`
	Proof      map[string]any `json:"proof"`
	JWT        string         `json:"jwt"`
}

// GenerateKeyMaterial creates a new did:web issuer and Ed25519 keypair.
func GenerateKeyMaterial(webOrigin string) (*KeyMaterial, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	issuerDID, err := DIDWebFromOrigin(webOrigin)
	if err != nil {
		return nil, err
	}
	jwk, err := publicJWK(pub)
	if err != nil {
		return nil, err
	}
	return &KeyMaterial{
		IssuerDID:    issuerDID,
		PublicKeyJWK: jwk,
		PrivateKey:   priv,
	}, nil
}

// KeyMaterialFromPrivate restores signing material from raw private key bytes and issuer DID.
func KeyMaterialFromPrivate(issuerDID string, privateKey ed25519.PrivateKey, publicJWKBytes json.RawMessage) *KeyMaterial {
	return &KeyMaterial{
		IssuerDID:    issuerDID,
		PublicKeyJWK: publicJWKBytes,
		PrivateKey:   privateKey,
	}
}

// DIDWebFromOrigin converts a public web origin into a did:web identifier.
func DIDWebFromOrigin(origin string) (string, error) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "did:web:localhost", nil
	}
	u, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("vc_signing: invalid web origin: %w", err)
	}
	host := strings.TrimSuffix(u.Host, "/")
	if host == "" {
		return "", errors.New("vc_signing: web origin missing host")
	}
	host = strings.ReplaceAll(host, ":", "%3A")
	path := strings.Trim(u.Path, "/")
	if path != "" {
		return "did:web:" + host + ":" + strings.ReplaceAll(path, "/", ":"), nil
	}
	return "did:web:" + host, nil
}

// BuildDIDDocument returns a W3C DID document for the institutional issuer.
func BuildDIDDocument(km *KeyMaterial) (map[string]any, error) {
	if km == nil {
		return nil, errors.New("vc_signing: key material required")
	}
	var jwk map[string]any
	if err := json.Unmarshal(km.PublicKeyJWK, &jwk); err != nil {
		return nil, err
	}
	vmID := km.IssuerDID + "#key-1"
	return map[string]any{
		"@context": []string{"https://www.w3.org/ns/did/v1"},
		"id":       km.IssuerDID,
		"verificationMethod": []map[string]any{
			{
				"id":           vmID,
				"type":         "Ed25519VerificationKey2020",
				"controller":   km.IssuerDID,
				"publicKeyJwk": jwk,
			},
		},
		"authentication":  []string{vmID},
		"assertionMethod": []string{vmID},
	}, nil
}

// SignCLR wraps a CLR payload as a W3C Verifiable Credential with a JWT proof.
func SignCLR(km *KeyMaterial, clr map[string]any, subjectID string) (*SignedCredential, error) {
	if km == nil {
		return nil, errors.New("vc_signing: key material required")
	}
	now := time.Now().UTC()
	credential := map[string]any{
		"@context":     []string{vcContextURL, clrContextURL},
		"type":         []string{"VerifiableCredential", "ClrCredential"},
		"issuer":       km.IssuerDID,
		"issuanceDate": now.Format(time.RFC3339),
		"credentialSubject": map[string]any{
			"id":  subjectID,
			"clr": clr,
		},
	}

	payload, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}
	claims := jwt.MapClaims{
		"iss": km.IssuerDID,
		"sub": subjectID,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"vc":  json.RawMessage(payload),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(km.PrivateKey)
	if err != nil {
		return nil, err
	}
	proof := map[string]any{
		"type":               "JwtProof2020",
		"created":            now.Format(time.RFC3339),
		"verificationMethod": km.IssuerDID + "#key-1",
		"jwt":                signed,
	}
	credential["proof"] = proof
	return &SignedCredential{
		Credential: credential,
		Proof:      proof,
		JWT:        signed,
	}, nil
}

// VerifyJWT validates a JWT proof against the institutional public key.
func VerifyJWT(publicKey ed25519.PublicKey, tokenString string) (bool, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	_, err := parser.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// PublicKeyFromJWK extracts an Ed25519 public key from a JWK blob.
func PublicKeyFromJWK(jwkBytes json.RawMessage) (ed25519.PublicKey, error) {
	var jwk struct {
		X string `json:"x"`
	}
	if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
		return nil, err
	}
	raw, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("vc_signing: invalid ed25519 public key size")
	}
	return ed25519.PublicKey(raw), nil
}

func publicJWK(pub ed25519.PublicKey) (json.RawMessage, error) {
	jwk := map[string]string{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(pub),
	}
	return json.Marshal(jwk)
}
