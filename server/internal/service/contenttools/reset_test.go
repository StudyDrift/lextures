package contenttools

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

func TestAsyncResetThresholdDefault(t *testing.T) {
	t.Setenv(EnvAsyncResetThreshold, "")
	if AsyncResetThreshold() != DefaultAsyncResetThreshold {
		t.Fatalf("got %d", AsyncResetThreshold())
	}
	t.Setenv(EnvAsyncResetThreshold, "50")
	if AsyncResetThreshold() != 50 {
		t.Fatalf("got %d", AsyncResetThreshold())
	}
	t.Setenv(EnvAsyncResetThreshold, "bogus")
	if AsyncResetThreshold() != DefaultAsyncResetThreshold {
		t.Fatalf("invalid should fall back, got %d", AsyncResetThreshold())
	}
	_ = os.Unsetenv(EnvAsyncResetThreshold)
}

func TestValidResetScopes(t *testing.T) {
	for _, s := range []string{
		ctrepo.ResetScopeInstanceEnrollment,
		ctrepo.ResetScopeInstanceAll,
		ctrepo.ResetScopeItemEnrollment,
		ctrepo.ResetScopeItemAll,
		ctrepo.ResetScopeCourseEnrollment,
	} {
		if _, ok := ValidResetScopes[s]; !ok {
			t.Fatalf("missing scope %s", s)
		}
	}
}

func TestInitialStateForTool(t *testing.T) {
	raw := InitialStateForTool(nil)
	if string(raw) != "{}" {
		t.Fatalf("got %s", raw)
	}
}

func TestSummarizeStateNoopProbe(t *testing.T) {
	state := json.RawMessage(`{"response":"hello world","attempts":2}`)
	score := 1.0
	max := 1.0
	s := SummarizeState("noop_probe", state, "completed", &score, &max)
	if s == "" || !containsAll(s, "status=completed", "score=1/1", "response=") {
		t.Fatalf("unexpected summary: %s", s)
	}
}

func TestClassifyGradeEffectStub(t *testing.T) {
	id := uuid.New()
	g := ClassifyGradeEffect(id, "auto", true)
	if g.Action != "unchanged" || g.EnrollmentID != id {
		t.Fatalf("unexpected %+v", g)
	}
}

func TestIntersectUUIDs(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	out := intersectUUIDs([]uuid.UUID{a, b}, []uuid.UUID{b, c})
	if len(out) != 1 || out[0] != b {
		t.Fatalf("got %v", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsStr(s, p) {
			return false
		}
	}
	return true
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
