package seattime

import (
	"testing"
	"time"

	"github.com/google/uuid"
	reposeattime "github.com/lextures/lextures/server/internal/repos/seattime"
)

func TestApplyHeartbeat_countsOneMinutePerGap(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	state := SessionState{}

	state, counted := ApplyHeartbeat(state, now, 0)
	if !counted || state.MinutesActive != 1 {
		t.Fatalf("first heartbeat: counted=%v minutes=%d", counted, state.MinutesActive)
	}

	state, counted = ApplyHeartbeat(state, now.Add(1*time.Second), 0)
	if counted || !state.AnomalyFlag {
		t.Fatalf("rapid heartbeat: counted=%v anomaly=%v", counted, state.AnomalyFlag)
	}
	if state.MinutesActive != 1 {
		t.Fatalf("minutes=%d want 1", state.MinutesActive)
	}

	state, counted = ApplyHeartbeat(state, now.Add(61*time.Second), 0)
	if !counted || state.MinutesActive != 2 {
		t.Fatalf("second minute: counted=%v minutes=%d", counted, state.MinutesActive)
	}
}

func TestApplyHeartbeat_dailyCap(t *testing.T) {
	now := time.Now().UTC()
	state := SessionState{
		LastCountedAt: now.Add(-61 * time.Second),
		MinutesActive: 5,
	}
	_, counted := ApplyHeartbeat(state, now, MaxDailyMinutesPerCourse)
	if counted {
		t.Fatal("expected daily cap to prevent counting")
	}
}

func TestComputeProgress_values(t *testing.T) {
	cfg := &reposeattime.CEUConfig{RequiredHours: 10, CEUCredit: 1, Enabled: true}
	partial := ComputeProgress(300, cfg, false)
	if partial.ProgressPct != 50 {
		t.Fatalf("progress pct=%v want 50", partial.ProgressPct)
	}
	if partial.CEUEarned < 0.49 || partial.CEUEarned > 0.51 {
		t.Fatalf("ceu earned=%v want ~0.5", partial.CEUEarned)
	}

	awarded := ComputeProgress(600, cfg, true)
	if !awarded.Awarded || awarded.CEUEarned != 1 {
		t.Fatalf("awarded=%+v", awarded)
	}
}

func TestSessionKey_unique(t *testing.T) {
	u1 := uuid.New()
	u2 := uuid.New()
	k1 := sessionKey(u1, u2, "tok")
	k2 := sessionKey(u1, u2, "tok2")
	if k1 == k2 {
		t.Fatal("expected distinct keys")
	}
}

func TestBuildTranscriptPDF_nonempty(t *testing.T) {
	pdf, err := BuildTranscriptPDF(TranscriptInput{
		InstitutionName: "Test U",
		LearnerName:     "Ada Lovelace",
		GeneratedAt:     time.Now().UTC(),
		Rows: []TranscriptRow{{
			CourseTitle:  "Nursing CE",
			CEUCredit:    1,
			ContactHours: 10,
			CompletedAt:  time.Now().UTC(),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 {
		t.Fatalf("pdf too small: %d bytes", len(pdf))
	}
}
