package academicrecord

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// CanonicalJSON returns deterministic JSON bytes for hashing (sorted keys via encoding/json struct tags).
// Floating-point fields must already be rounded by ComputeCumulative / ComputeTermGPA.
func CanonicalJSON(rec *AcademicRecord) ([]byte, error) {
	if rec == nil {
		return []byte("null"), nil
	}
	return json.Marshal(rec)
}

// ContentHash returns the SHA-256 hex digest of the canonical JSON bytes.
func ContentHash(rec *AcademicRecord) (string, []byte, error) {
	raw, err := CanonicalJSON(rec)
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), raw, nil
}

// VerifyHash returns true when the stored hash matches the canonical JSON of rec.
func VerifyHash(rec *AcademicRecord, storedHash string) (bool, error) {
	h, _, err := ContentHash(rec)
	if err != nil {
		return false, err
	}
	return h == storedHash, nil
}

// ContentDocumentID returns a stable short document id derived from the content hash.
func (rec *AcademicRecord) ContentDocumentID() string {
	h, _, err := ContentHash(rec)
	if err != nil || len(h) < 16 {
		return rec.GeneratedAt
	}
	return h[:16]
}
