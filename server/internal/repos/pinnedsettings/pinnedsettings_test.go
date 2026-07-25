package pinnedsettings

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateKeys_OK(t *testing.T) {
	keys := []string{
		"quiz.presentation.lockdown-mode",
		"  Quiz.Scheduling.Due-Date  ",
	}
	out, reason, err := ValidateKeys(keys)
	if err != nil || reason != "" {
		t.Fatalf("unexpected err=%v reason=%q", err, reason)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0] != "quiz.presentation.lockdown-mode" {
		t.Fatalf("got %q", out[0])
	}
	if out[1] != "quiz.scheduling.due-date" {
		t.Fatalf("got %q", out[1])
	}
}

func TestValidateKeys_EmptyAccepted(t *testing.T) {
	out, reason, err := ValidateKeys(nil)
	if err != nil || reason != "" {
		t.Fatalf("err=%v reason=%q", err, reason)
	}
	if len(out) != 0 {
		t.Fatalf("want empty, got %v", out)
	}
	out, reason, err = ValidateKeys([]string{})
	if err != nil || reason != "" || len(out) != 0 {
		t.Fatalf("empty slice: out=%v err=%v reason=%q", out, err, reason)
	}
}

func TestValidateKeys_TooMany(t *testing.T) {
	keys := make([]string, MaxPins+1)
	for i := range keys {
		keys[i] = fmt.Sprintf("key.%d", i)
	}
	_, reason, err := ValidateKeys(keys)
	if err == nil || reason != ReasonTooMany {
		t.Fatalf("want too_many, got reason=%q err=%v", reason, err)
	}
}

func TestValidateKeys_Shape(t *testing.T) {
	cases := []string{
		"Quiz.Bad Key!",
		"",
		"has space",
		"UPPER_SNAKE",
		strings.Repeat("a", MaxKeyLen+1),
		".leading",
		"trailing.",
		"double..dot",
	}
	for _, c := range cases {
		_, reason, err := ValidateKeys([]string{c})
		if err == nil || reason != ReasonShape {
			t.Fatalf("key %q: want shape reject, got reason=%q err=%v", c, reason, err)
		}
	}
}

func TestValidateKeys_DuplicateAfterNorm(t *testing.T) {
	_, reason, err := ValidateKeys([]string{
		"quiz.presentation.lockdown-mode",
		"Quiz.Presentation.Lockdown-Mode",
	})
	if err == nil || reason != ReasonDuplicate {
		t.Fatalf("want duplicate, got reason=%q err=%v", reason, err)
	}
}

func TestValidSurface(t *testing.T) {
	if !ValidSurface(SurfaceAssignment) || !ValidSurface(SurfaceQuiz) {
		t.Fatal("expected assignment/quiz valid")
	}
	if ValidSurface("discussion") || ValidSurface("") {
		t.Fatal("expected discussion/empty invalid")
	}
}
