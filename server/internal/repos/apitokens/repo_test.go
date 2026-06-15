package apitokens

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateSecretFormat(t *testing.T) {
	t.Parallel()
	secret, hash, prefix, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) < 20 || secret[:4] != "ltk_" {
		t.Fatalf("unexpected secret format: %q", secret)
	}
	if hash == "" || prefix != secret[:8] {
		t.Fatalf("hash/prefix mismatch: hash=%q prefix=%q", hash, prefix)
	}
	if hashSecret(secret) != hash {
		t.Fatal("hashSecret mismatch")
	}
}

func TestMaskedDisplay(t *testing.T) {
	t.Parallel()
	if got := MaskedDisplay("ltk_abcd"); got != "ltk_abcd…" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeCourseIDs(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	out, err := NormalizeCourseIDs([]string{" " + id.String() + " ", id.String()})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != id {
		t.Fatalf("got %v", out)
	}
	if _, err := NormalizeCourseIDs([]string{"not-a-uuid"}); err == nil {
		t.Fatal("expected error")
	}
}
