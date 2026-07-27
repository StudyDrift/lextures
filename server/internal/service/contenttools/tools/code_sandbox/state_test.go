package code_sandbox

import (
	"strings"
	"testing"
	"time"
)

func TestTruncateOutput(t *testing.T) {
	short := TruncateOutput("hi", 100)
	if short != "hi" {
		t.Fatalf("short = %q", short)
	}
	long := strings.Repeat("a", 100)
	out := TruncateOutput(long, 40)
	if !strings.HasSuffix(out, TruncationMarker) {
		t.Fatalf("expected truncation marker, got %q", out)
	}
	if len(out) > 40+len(TruncationMarker) {
		t.Fatalf("len=%d too large", len(out))
	}
}

func TestRateLimitHourly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RunLimitPerHour = 2
	now := time.Date(2026, 7, 27, 15, 30, 0, 0, time.UTC)
	st := EmptyState()
	st = RecordRateUsage(st, ActionRun, now)
	st = RecordRateUsage(st, ActionRun, now)
	if err := CheckRateLimit(cfg, st, ActionRun, now); err == nil {
		t.Fatal("expected rate limit")
	} else if err.ResetAt != HourResetAt(now) {
		t.Fatalf("resetAt = %d want %d", err.ResetAt, HourResetAt(now))
	}
	// Next hour clears.
	later := now.Add(time.Hour)
	st = EnsureRateWindow(st, later)
	if err := CheckRateLimit(cfg, st, ActionRun, later); err != nil {
		t.Fatalf("unexpected limit after hour roll: %v", err)
	}
}

func TestMatchErrorHint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ErrorHints = []ErrorHint{{Match: "NameError", Hint: "Did you define the variable?"}}
	got := MatchErrorHint(cfg, "Traceback\nNameError: name 'x' is not defined")
	if got != "Did you define the variable?" {
		t.Fatalf("hint = %q", got)
	}
	if MatchErrorHint(cfg, "SyntaxError") != "" {
		t.Fatal("expected no hint")
	}
}

func TestAppendRunCaps(t *testing.T) {
	st := EmptyState()
	for i := 0; i < 15; i++ {
		st = AppendRun(st, RunRecord{At: NowRFC3339(), Action: ActionRun, Status: StatusOK}, 10)
	}
	if len(st.Runs) != 10 {
		t.Fatalf("len=%d", len(st.Runs))
	}
}

func TestUpdateBest(t *testing.T) {
	st := EmptyState()
	st = UpdateBest(st, 2, 5, "t1")
	st = UpdateBest(st, 1, 5, "t2")
	if st.Best == nil || st.Best.Passed != 2 {
		t.Fatalf("best=%v", st.Best)
	}
	st = UpdateBest(st, 4, 5, "t3")
	if st.Best.Passed != 4 {
		t.Fatalf("best.passed=%d", st.Best.Passed)
	}
}

func TestComposeCode(t *testing.T) {
	cfg := Config{PrefixCode: "PREFIX", SuffixCode: "SUFFIX"}
	got := ComposeCode(cfg, "body")
	if !strings.HasPrefix(got, "PREFIX\n") || !strings.HasSuffix(got, "SUFFIX") {
		t.Fatalf("got %q", got)
	}
}

func TestEffectiveScoringMode(t *testing.T) {
	cfg := DefaultConfig()
	if EffectiveScoringMode(cfg) != ScoringNone {
		t.Fatal("no tests => none")
	}
	cfg.Tests = []TestCase{{ID: "t1", Name: "one"}}
	if EffectiveScoringMode(cfg) != ScoringAuto {
		t.Fatal("tests => auto")
	}
	cfg.ScoringMode = ScoringNone
	if EffectiveScoringMode(cfg) != ScoringNone {
		t.Fatal("explicit none")
	}
}

func TestNormalizeLanguage(t *testing.T) {
	if NormalizeLanguage("Python3") != "python" {
		t.Fatal("python")
	}
	if NormalizeLanguage("node") != "javascript" {
		t.Fatal("js")
	}
	if !SupportedLanguage("python") || SupportedLanguage("ruby") {
		t.Fatal("supported")
	}
}
