package gradingagent

import (
	"fmt"
	"strings"
	"testing"
)

func TestUserFacingScoreError_OpenRouterAuth(t *testing.T) {
	msg := UserFacingScoreError(fmt.Errorf("openrouter: status 401: invalid key"))
	if !strings.Contains(msg, "API key") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestUserFacingScoreError_ModelNotConfigured(t *testing.T) {
	msg := UserFacingScoreError(fmt.Errorf("grader agent model not configured"))
	if !strings.Contains(msg, "Intelligence") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestUserFacingScoreError_InvalidJSON(t *testing.T) {
	msg := UserFacingScoreError(fmt.Errorf("invalid model JSON: unexpected end"))
	if !strings.Contains(msg, "unreadable") {
		t.Fatalf("msg=%q", msg)
	}
}