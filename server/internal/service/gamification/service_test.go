package gamification

import (
	"testing"
	"time"
)

func TestLevelFromXP(t *testing.T) {
	cases := []struct {
		xp    int
		level int
	}{
		{0, 0},
		{9, 0},
		{10, 1},
		{39, 1},
		{40, 2},
		{100, 3},
		{1000, 10},
	}
	for _, tc := range cases {
		if got := LevelFromXP(tc.xp); got != tc.level {
			t.Errorf("LevelFromXP(%d) = %d, want %d", tc.xp, got, tc.level)
		}
	}
}

func TestComputeStreakAfterActivity(t *testing.T) {
	today := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	t.Run("first activity", func(t *testing.T) {
		streak, longest, used := ComputeStreakAfterActivity(nil, 0, 0, nil, today)
		if streak != 1 || longest != 1 || used {
			t.Fatalf("got streak=%d longest=%d used=%v", streak, longest, used)
		}
	})

	t.Run("same day", func(t *testing.T) {
		streak, longest, used := ComputeStreakAfterActivity(&today, 5, 5, nil, today)
		if streak != 5 || longest != 5 || used {
			t.Fatalf("got streak=%d longest=%d used=%v", streak, longest, used)
		}
	})

	t.Run("consecutive day", func(t *testing.T) {
		streak, longest, used := ComputeStreakAfterActivity(&yesterday, 5, 5, nil, today)
		if streak != 6 || longest != 6 || used {
			t.Fatalf("got streak=%d longest=%d used=%v", streak, longest, used)
		}
	})

	t.Run("gap resets", func(t *testing.T) {
		old := today.AddDate(0, 0, -3)
		streak, longest, used := ComputeStreakAfterActivity(&old, 10, 10, nil, today)
		if streak != 1 || longest != 10 || used {
			t.Fatalf("got streak=%d longest=%d used=%v", streak, longest, used)
		}
	})
}

func TestReconcileStreakOnLogin(t *testing.T) {
	today := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	streak, ended, _ := ReconcileStreakOnLogin(&yesterday, 5, nil, today)
	if streak != 5 || ended {
		t.Fatalf("active yesterday: streak=%d ended=%v", streak, ended)
	}

	streak, ended, _ = ReconcileStreakOnLogin(&twoDaysAgo, 5, nil, today)
	if streak != 0 || !ended {
		t.Fatalf("missed day: streak=%d ended=%v", streak, ended)
	}
}

func TestXPAward(t *testing.T) {
	if XPAward(ActivityModuleItemViewed) != 5 {
		t.Fatalf("module item XP want 5 got %d", XPAward(ActivityModuleItemViewed))
	}
	if XPAward(ActivityQuizPassed) != 20 {
		t.Fatalf("quiz XP want 20 got %d", XPAward(ActivityQuizPassed))
	}
}
