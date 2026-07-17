package diplomas

import (
	"encoding/json"
	"testing"
)

func TestMustMarshalCanonicalStable(t *testing.T) {
	raw, hash, err := MustMarshalCanonical(map[string]any{"a": 1, "b": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 || hash == "" {
		t.Fatal("expected canonical + hash")
	}
	var again map[string]any
	if err := json.Unmarshal(raw, &again); err != nil {
		t.Fatal(err)
	}
	if HashCanonical(raw) != hash {
		t.Fatal("hash mismatch")
	}
}

func TestVerifyContentHash(t *testing.T) {
	raw, hash, err := MustMarshalCanonical(map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	d := &Diploma{Canonical: raw, ContentHash: hash}
	if !VerifyContentHash(d) {
		t.Fatal("expected valid hash")
	}
	d.ContentHash = "deadbeef"
	if VerifyContentHash(d) {
		t.Fatal("expected invalid hash")
	}
}
