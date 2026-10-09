package httpserver

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
)

func TestAttemptsRemainingAfterStart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cap, attempt, want int32
	}{
		{cap: 1, attempt: 1, want: 0},
		{cap: 3, attempt: 1, want: 2},
		{cap: 3, attempt: 2, want: 1},
		{cap: 3, attempt: 3, want: 0},
		{cap: 1, attempt: 2, want: 0},
	}
	for _, tc := range cases {
		if got := attemptsRemainingAfterStart(tc.cap, tc.attempt); got != tc.want {
			t.Fatalf("cap=%d attempt=%d: got %d want %d", tc.cap, tc.attempt, got, tc.want)
		}
	}
}

func TestEffectiveQuizAttemptCap(t *testing.T) {
	t.Parallel()
	if _, limited := effectiveQuizAttemptCap(true, 1, 4); limited {
		t.Fatal("unlimited quizzes are not capped")
	}
	cap, limited := effectiveQuizAttemptCap(false, 1, 2)
	if !limited || cap != 3 {
		t.Fatalf("cap=%d limited=%v", cap, limited)
	}
	cap, limited = effectiveQuizAttemptCap(false, 0, 0)
	if !limited || cap != 1 {
		t.Fatalf("floor cap=%d limited=%v", cap, limited)
	}
}

func TestWriteQuizStartResponseCountsAttemptJustStarted(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	row := &coursemodulequizzes.CourseItemQuizRow{
		UnlimitedAttempts:   false,
		MaxAttempts:         1,
		GradeAttemptPolicy:  "latest",
		AllowBackNavigation: true,
		LockdownMode:        "standard",
	}
	attempt := &quizattempts.QuizAttemptRow{
		ID:            uuid.New(),
		AttemptNumber: 1,
		StartedAt:     time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC),
	}
	Deps{}.writeQuizStartResponse(rec, row, &course.CourseQuizMeta{}, attempt, acsvc.Effective{TimeMultiplier: 1})
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["attemptNumber"].(float64) != 1 {
		t.Fatalf("attemptNumber %#v", body["attemptNumber"])
	}
	if body["maxAttempts"].(float64) != 1 {
		t.Fatalf("maxAttempts %#v", body["maxAttempts"])
	}
	if body["remainingAttempts"].(float64) != 0 {
		t.Fatalf("remainingAttempts %#v", body["remainingAttempts"])
	}

	rec = httptest.NewRecorder()
	row.MaxAttempts = 1
	Deps{}.writeQuizStartResponse(rec, row, &course.CourseQuizMeta{}, attempt, acsvc.Effective{TimeMultiplier: 1, ExtraAttempts: 1})
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["maxAttempts"].(float64) != 2 || body["remainingAttempts"].(float64) != 1 {
		t.Fatalf("extra attempt budget: max=%v remaining=%v", body["maxAttempts"], body["remainingAttempts"])
	}
}
