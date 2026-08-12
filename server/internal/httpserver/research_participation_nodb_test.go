package httpserver

import (
	"testing"
	"time"

	"github.com/google/uuid"
	repo "github.com/lextures/lextures/server/internal/repos/researchparticipation"
)

func TestResearchParticipationResponseUnresolvedIsFailClosed(t *testing.T) {
	out := researchParticipationResponse(uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil)
	if out.Resolved || out.Participation != nil {
		t.Fatalf("missing decision must remain unresolved: %#v", out)
	}
}

func TestResearchParticipationResponseResolved(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	out := researchParticipationResponse(uuid.Nil, &repo.Setting{Participation: repo.OptOut, UpdatedAt: now})
	if !out.Resolved || out.Participation == nil || *out.Participation != repo.OptOut || out.UpdatedAt == nil {
		t.Fatalf("unexpected response: %#v", out)
	}
}
