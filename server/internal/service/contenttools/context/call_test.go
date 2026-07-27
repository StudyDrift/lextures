package context

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/aitutor"
)

func TestRedactPII_emailsAndPhones(t *testing.T) {
	in := "Contact me at student@example.com or 415-555-1212 please."
	out := aitutor.RedactPII(in)
	if strings.Contains(out, "student@example.com") || strings.Contains(out, "415-555-1212") {
		t.Fatalf("PII not redacted: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction markers: %q", out)
	}
}

func TestBudgetErrorLevels(t *testing.T) {
	cases := []struct {
		level string
		want  error
	}{
		{"request", ErrBudgetRequestTokens},
		{"daily_user", ErrBudgetDailyUserCalls},
		{"monthly_course", ErrBudgetCourseMonthly},
	}
	for _, tc := range cases {
		e := &BudgetError{Level: tc.level, Message: "x"}
		if e.Unwrap() != tc.want {
			t.Fatalf("%s unwrap=%v", tc.level, e.Unwrap())
		}
	}
}

func TestSupportsToolCallingDryRun(t *testing.T) {
	// DryRunToolCallingCompleter is in aiprovider; here we only assert citation filter.
	pack := &ContextPack{Segments: []ContextSegment{{Kind: KindLink, ID: "abc", Title: "T", Text: "x"}}}
	cites := []Citation{{Kind: CiteLink, ID: "abc", Title: "T"}, {Kind: CiteLink, ID: "nope", Title: "X"}}
	got := FilterValidCitations(cites, pack)
	if len(got) != 1 || got[0].ID != "abc" {
		t.Fatalf("%+v", got)
	}
}
