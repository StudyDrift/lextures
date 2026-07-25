package adaptivepath

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestServiceHealth(t *testing.T) {
	s := New()
	if s.Name != "adaptivepath" {
		t.Fatalf("name: %q", s.Name)
	}
	got, err := s.Health(context.Background())
	if err != nil || got != "adaptivepath:ok" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestValidateRuleType(t *testing.T) {
	for _, rt := range []string{"skip_if_mastered", "required_if_not_mastered", "unlock_after", "remediation_insert"} {
		if err := ValidateRuleType(rt); err != nil {
			t.Fatalf("%s: %v", rt, err)
		}
	}
	if err := ValidateRuleType("nope"); !errors.Is(err, ErrInvalidRuleType) {
		t.Fatalf("want ErrInvalidRuleType got %v", err)
	}
}

func TestRequireTargetForRuleType(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if err := RequireTargetForRuleType("skip_if_mastered", nil); err != nil {
		t.Fatalf("skip allows nil target: %v", err)
	}
	if err := RequireTargetForRuleType("unlock_after", nil); !errors.Is(err, ErrTargetRequired) {
		t.Fatalf("unlock requires target: %v", err)
	}
	if err := RequireTargetForRuleType("unlock_after", &id); err != nil {
		t.Fatalf("unlock with target: %v", err)
	}
}

func TestValidateThreshold(t *testing.T) {
	if err := ValidateThreshold(0.5); err != nil {
		t.Fatal(err)
	}
	if err := ValidateThreshold(-0.1); !errors.Is(err, ErrBadThreshold) {
		t.Fatalf("got %v", err)
	}
	if err := ValidateThreshold(1.1); !errors.Is(err, ErrBadThreshold) {
		t.Fatalf("got %v", err)
	}
}
