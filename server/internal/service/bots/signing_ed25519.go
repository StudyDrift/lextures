package bots

import (
	"crypto/ed25519"
	"encoding/hex"
)

func verifyEd25519(publicKeyHex string, message []byte, signatureHex string) bool {
	pubBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), message, sigBytes)
}
