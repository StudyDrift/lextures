package diplomas

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func sha256Sum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashBytes returns hex SHA-256 of arbitrary bytes.
func HashBytes(b []byte) string {
	return sha256Sum(b)
}

// MustMarshalCanonical marshals v as compact JSON for hashing.
func MustMarshalCanonical(v any) (json.RawMessage, string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	return json.RawMessage(raw), sha256Sum(raw), nil
}
