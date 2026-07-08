package httpserver

import (
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestProfileAdaptEnabled_RequiresMasterFlag(t *testing.T) {
	d := Deps{Config: config.Config{
		LearnerProfileEnabled:           false,
		LpAdaptRecommendationsEnabled:   true,
	}}
	if d.profileAdaptEnabled("recommendations") {
		t.Fatal("expected disabled without learner profile master flag")
	}
}

func TestRationaleToJSON_Nil(t *testing.T) {
	if rationaleToJSON(nil) != nil {
		t.Fatal("expected nil")
	}
}