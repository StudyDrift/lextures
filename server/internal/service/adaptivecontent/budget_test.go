package adaptivecontent

import (
	"testing"
	"time"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

func TestCheckAndReserve_Unlimited(t *testing.T) {
	t.Parallel()
	c := CheckAndReserve(0, 1_000_000, DefaultEstimateTokens)
	if !c.Allowed {
		t.Fatal("budget 0 should be unlimited")
	}
	if c.Reason != "unlimited" {
		t.Fatalf("reason: %s", c.Reason)
	}
}

func TestCheckAndReserve_WithinBudget(t *testing.T) {
	t.Parallel()
	c := CheckAndReserve(100_000, 90_000, 5_000)
	if !c.Allowed {
		t.Fatal("expected allowed")
	}
	if c.Remaining != 10_000 {
		t.Fatalf("remaining: %d", c.Remaining)
	}
}

func TestCheckAndReserve_Exhausted(t *testing.T) {
	t.Parallel()
	c := CheckAndReserve(100_000, 98_000, 5_000)
	if c.Allowed {
		t.Fatal("expected exhausted")
	}
	if c.Reason != "budget_exhausted" {
		t.Fatalf("reason: %s", c.Reason)
	}
	if c.Remaining != 2_000 {
		t.Fatalf("remaining: %d", c.Remaining)
	}
}

func TestCheckAndReserve_ExactBoundary(t *testing.T) {
	t.Parallel()
	// used + est == budget is allowed
	c := CheckAndReserve(1000, 600, 400)
	if !c.Allowed {
		t.Fatal("exact boundary should be allowed")
	}
	// one over fails
	c2 := CheckAndReserve(1000, 600, 401)
	if c2.Allowed {
		t.Fatal("one over should fail")
	}
}

func TestPeriodStartUTC(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 7, 25, 15, 30, 0, 0, time.UTC)
	p := acrepo.PeriodStartUTC(ts)
	if p.Year() != 2026 || p.Month() != 7 || p.Day() != 1 {
		t.Fatalf("got %v", p)
	}
}

func TestJobBackoff_Schedule(t *testing.T) {
	t.Parallel()
	if JobBackoff(1) != 30*time.Second {
		t.Fatalf("attempt 1: %v", JobBackoff(1))
	}
	if JobBackoff(2) != 2*time.Minute {
		t.Fatalf("attempt 2: %v", JobBackoff(2))
	}
	// Beyond schedule reuses last
	last := JobBackoff(100)
	if last != 2*time.Hour {
		t.Fatalf("attempt 100: %v", last)
	}
}

func TestIsTransientGenError(t *testing.T) {
	t.Parallel()
	if IsTransientGenError(ErrBudgetExhausted) {
		t.Fatal("budget should not retry")
	}
	if IsTransientGenError(ErrRejectedFidelity) {
		t.Fatal("fidelity should not retry")
	}
	if IsTransientGenError(ErrRejectedSafety) {
		t.Fatal("safety should not retry")
	}
	if !IsTransientGenError(ErrGenerationFailed) {
		t.Fatal("generation failed should retry")
	}
}

func TestGlobalRateLimiter_Burst(t *testing.T) {
	t.Parallel()
	lim := NewGlobalRateLimiter(100, 3)
	for i := 0; i < 3; i++ {
		if !lim.TryAcquire() {
			t.Fatalf("token %d should succeed", i)
		}
	}
	if lim.TryAcquire() {
		t.Fatal("burst exhausted")
	}
}
