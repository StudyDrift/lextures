package introcourse

import (
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCompletionPoints(t *testing.T) {
	if got := completionPoints(nil); got != 1 {
		t.Fatalf("nil points: got %v want 1", got)
	}
	ten := 10
	if got := completionPoints(&ten); got != 10 {
		t.Fatalf("ten points: got %v want 10", got)
	}
	zero := 0
	if got := completionPoints(&zero); got != 1 {
		t.Fatalf("zero points: got %v want 1", got)
	}
}

func TestGraderAgentFeedbackEnabled(t *testing.T) {
	cfg := configWithGrader(true, true)
	if !graderAgentFeedbackEnabled(cfg) {
		t.Fatal("expected enabled when both flags on")
	}
	cfg = configWithGrader(false, true)
	if graderAgentFeedbackEnabled(cfg) {
		t.Fatal("expected disabled when master flag off")
	}
	cfg = configWithGrader(true, false)
	if graderAgentFeedbackEnabled(cfg) {
		t.Fatal("expected disabled when text entry flag off")
	}
}

func configWithGrader(master, text bool) config.Config {
	return config.Config{
		GraderAgentEnabled:                 master,
		GraderAgentTextEntryGradingEnabled: text,
	}
}