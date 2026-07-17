package scoring

import (
	"encoding/json"
	"testing"
)

func TestCompetitiveSpeedBonus(t *testing.T) {
	// AC-1: 2s vs 8s on 10s timer — faster earns more; both ≥ base.
	cfg := DefaultConfig(ProfileCompetitive)
	fast := Score(Input{
		IsCorrect: true, ResponseMs: 2000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Profile: ProfileCompetitive, Config: cfg,
	})
	slow := Score(Input{
		IsCorrect: true, ResponseMs: 8000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Profile: ProfileCompetitive, Config: cfg,
	})
	if fast.Points <= slow.Points {
		t.Fatalf("fast (%d) should beat slow (%d)", fast.Points, slow.Points)
	}
	if fast.Points < cfg.Base || slow.Points < cfg.Base {
		t.Fatalf("both should be ≥ base: fast=%d slow=%d base=%d", fast.Points, slow.Points, cfg.Base)
	}
	// Fastest ≈ 2× base: speed_factor≈0.8 → 1000+800=1800
	if fast.Points != 1800 {
		t.Fatalf("fast points: want 1800 got %d (bd=%+v)", fast.Points, fast.Breakdown)
	}
	if slow.Points != 1200 {
		t.Fatalf("slow points: want 1200 got %d", slow.Points)
	}
}

func TestDoublePointsExactlyTwice(t *testing.T) {
	// AC-2
	cfg := DefaultConfig(ProfileCompetitive)
	std := Score(Input{
		IsCorrect: true, ResponseMs: 2000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Profile: ProfileCompetitive, Config: cfg,
	})
	dbl := Score(Input{
		IsCorrect: true, ResponseMs: 2000, TimeLimitMs: 10000,
		PointsStyle: StyleDouble, QuestionType: "mc_single",
		Profile: ProfileCompetitive, Config: cfg,
	})
	if dbl.Points != std.Points*2 {
		t.Fatalf("double want %d got %d", std.Points*2, dbl.Points)
	}
}

func TestStreakBonusAndReset(t *testing.T) {
	// AC-3: three correct then miss → streak bonuses accrue, then reset.
	cfg := DefaultConfig(ProfileCompetitive)
	streak := 0
	var bonuses []int
	for i := 0; i < 3; i++ {
		r := Score(Input{
			IsCorrect: true, ResponseMs: 5000, TimeLimitMs: 10000,
			PointsStyle: StyleStandard, QuestionType: "mc_single",
			StreakBefore: streak, Profile: ProfileCompetitive, Config: cfg,
		})
		bonuses = append(bonuses, r.Breakdown.StreakBonus)
		streak = r.StreakAfter
	}
	if bonuses[0] != 0 || bonuses[1] != 100 || bonuses[2] != 200 {
		t.Fatalf("streak bonuses want [0,100,200] got %v", bonuses)
	}
	if streak != 3 {
		t.Fatalf("streak after 3 correct: want 3 got %d", streak)
	}
	miss := Score(Input{
		IsCorrect: false, ResponseMs: 1000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		StreakBefore: streak, Profile: ProfileCompetitive, Config: cfg,
	})
	if miss.StreakAfter != 0 {
		t.Fatalf("streak after miss: want 0 got %d", miss.StreakAfter)
	}
	if miss.Points != 0 {
		t.Fatalf("miss points: want 0 got %d", miss.Points)
	}
}

func TestFormativeIgnoresSpeed(t *testing.T) {
	// AC-4
	cfg := DefaultConfig(ProfileFormative)
	a := Score(Input{
		IsCorrect: true, ResponseMs: 1000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Profile: ProfileFormative, Config: cfg,
	})
	b := Score(Input{
		IsCorrect: true, ResponseMs: 9000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Profile: ProfileFormative, Config: cfg,
	})
	if a.Points != b.Points || a.Points != 1000 {
		t.Fatalf("formative equal fixed points: a=%d b=%d", a.Points, b.Points)
	}
	if a.Breakdown.SpeedBonus != 0 || a.Breakdown.StreakBonus != 0 {
		t.Fatalf("formative should have no speed/streak: %+v", a.Breakdown)
	}
}

func TestBreakdownSumsToTotal(t *testing.T) {
	// AC-6
	cfg := DefaultConfig(ProfileCompetitive)
	r := Score(Input{
		IsCorrect: true, ResponseMs: 2000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		StreakBefore: 2, Profile: ProfileCompetitive, Config: cfg,
	})
	bd := r.Breakdown
	sum := int(float64(bd.Base+bd.SpeedBonus+bd.StreakBonus)*bd.StyleMultiplier + 0.5)
	if bd.PowerUpFactor > 1 {
		sum = int(float64(sum) * bd.PowerUpFactor)
	}
	if sum != bd.Total || bd.Total != r.Points {
		t.Fatalf("breakdown sum %d != total %d (points %d) bd=%+v", sum, bd.Total, r.Points, bd)
	}
}

func TestReproducibilityByStoredConfig(t *testing.T) {
	// AC-7: recompute with stored profile/version/config matches.
	cfg := Config{Base: 500, SpeedWeight: 1, StreakStep: 50, StreakCap: 3}
	first := Score(Input{
		IsCorrect: true, ResponseMs: 2500, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		StreakBefore: 1, Profile: ProfileCustom, Config: ResolveConfig(ProfileCustom, cfg),
	})
	again := Recompute(ProfileCustom, Version, cfg, true, 2500, 10000, StyleStandard, "mc_single", 1, "", false)
	if again.Points != first.Points {
		t.Fatalf("recompute mismatch: %d vs %d", again.Points, first.Points)
	}
	b, _ := json.Marshal(first.Breakdown)
	var parsed Breakdown
	_ = json.Unmarshal(b, &parsed)
	if parsed.Total != first.Breakdown.Total {
		t.Fatalf("breakdown round-trip")
	}
}

func TestNoPointsAndPoll(t *testing.T) {
	cfg := DefaultConfig(ProfileCompetitive)
	r := Score(Input{
		IsCorrect: true, ResponseMs: 100, TimeLimitMs: 10000,
		PointsStyle: StyleNoPoints, QuestionType: "mc_single", Config: cfg,
	})
	if r.Points != 0 {
		t.Fatalf("no_points want 0 got %d", r.Points)
	}
	poll := Score(Input{
		IsCorrect: false, ResponseMs: 100, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "poll", Config: cfg,
	})
	if poll.Points != 0 || poll.StreakAfter != 0 {
		t.Fatalf("poll should be 0: %+v", poll)
	}
}

func TestDoubleOrNothing(t *testing.T) {
	cfg := DefaultConfig(ProfileCompetitive)
	cfg.PowerUpsEnabled = true
	base := Score(Input{
		IsCorrect: true, ResponseMs: 5000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single", Config: cfg,
	})
	don := Score(Input{
		IsCorrect: true, ResponseMs: 5000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Config: cfg, PowerUp: PowerUpDoubleOrNothing,
	})
	if don.Points != base.Points*2 {
		t.Fatalf("double-or-nothing want %d got %d", base.Points*2, don.Points)
	}
	miss := Score(Input{
		IsCorrect: false, ResponseMs: 5000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		Config: cfg, PowerUp: PowerUpDoubleOrNothing,
	})
	if miss.Points != 0 {
		t.Fatalf("don miss want 0 got %d", miss.Points)
	}
}

func TestShieldProtectsStreak(t *testing.T) {
	cfg := DefaultConfig(ProfileCompetitive)
	cfg.PowerUpsEnabled = true
	r := Score(Input{
		IsCorrect: false, ResponseMs: 1000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		StreakBefore: 4, Config: cfg, PowerUp: PowerUpShield, ShieldActive: true,
	})
	if r.StreakAfter != 4 || !r.ShieldUsed {
		t.Fatalf("shield should protect streak: %+v", r)
	}
	noShield := Score(Input{
		IsCorrect: false, ResponseMs: 1000, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single",
		StreakBefore: 4, Config: cfg, PowerUp: PowerUpShield, ShieldActive: false,
	})
	if noShield.StreakAfter != 0 {
		t.Fatalf("ineligible shield must not protect: %+v", noShield)
	}
}

func TestIncorrectZeroCompetitive(t *testing.T) {
	cfg := DefaultConfig(ProfileCompetitive)
	r := Score(Input{
		IsCorrect: false, ResponseMs: 100, TimeLimitMs: 10000,
		PointsStyle: StyleStandard, QuestionType: "mc_single", Config: cfg,
	})
	if r.Points != 0 {
		t.Fatalf("incorrect want 0 got %d", r.Points)
	}
}
