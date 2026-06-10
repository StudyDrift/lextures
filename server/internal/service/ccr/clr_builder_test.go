package ccr

import (
	"testing"
	"time"

	"github.com/google/uuid"
	repo "github.com/lextures/lextures/server/internal/repos/ccr"
)

func TestBuildCLRContainsAllAchievements(t *testing.T) {
	achievements := []repo.Achievement{
		{ID: uuid.New(), AchievementType: repo.TypeCourseCompletion, Title: "Biology 101", IssuedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), AchievementType: repo.TypeBadge, Title: "Leadership Badge", IssuedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), AchievementType: repo.TypePortfolio, Title: "Capstone Milestone", IssuedAt: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)},
	}
	clr := BuildCLR(BuildCLRInput{
		DocumentID:      uuid.New(),
		InstitutionName: "Example University",
		StudentName:     "Alex Student",
		StudentDID:      "urn:uuid:11111111-1111-4111-8111-111111111111",
		IssuerDID:       "did:web:app.example.com",
		IssuedAt:        time.Now().UTC(),
		Achievements:    achievements,
	})
	assertions, ok := clr["assertions"].([]map[string]any)
	if !ok {
		t.Fatalf("assertions type %T", clr["assertions"])
	}
	if len(assertions) != 3 {
		t.Fatalf("got %d assertions", len(assertions))
	}
}
