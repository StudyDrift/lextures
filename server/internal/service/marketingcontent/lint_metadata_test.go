package marketingcontent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	validator "github.com/lextures/lextures/server/internal/service/marketingcontent/validate"
)

func TestNormalizeLintMetadataAcceptsAuthorSlugAndDefaultsUpdated(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	raw := []byte(`{"title":"T","description":"A longer description for SEO cards.","authorSlug":"chase","cluster":"assessment","primaryQuestion":"What works?","keywords":["assessment"],"locale":"en"}`)
	var meta validator.Metadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	got := normalizeLintMetadata(meta, now)
	if got.Author != "chase" {
		t.Fatalf("author=%q", got.Author)
	}
	if got.Updated != "2026-08-13" {
		t.Fatalf("updated=%q", got.Updated)
	}
	report := validator.Article(validator.Input{
		Kind: "blog", BodyMD: ":::key-takeaways\n- a\n- b\n- c\n:::\n\n:::answer\n" + "word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word\n:::",
		Metadata: got, KnownPaths: map[string]struct{}{},
	})
	for _, f := range report.Findings {
		if f.Rule == "fm.author" || f.Rule == "fm.updated" {
			t.Fatalf("unexpected metadata finding after alias/default: %+v", f)
		}
	}
}

func TestLintBlockedErrorUnwraps(t *testing.T) {
	t.Parallel()
	err := &LintBlockedError{Report: validator.Report{Score: 3, Findings: []validator.Finding{{Rule: "struct.answer-block", Severity: "error", Message: "A direct answer block is required."}}}}
	if !errors.Is(err, ErrLintBlocked) {
		t.Fatal("expected errors.Is(ErrLintBlocked)")
	}
	var blocked *LintBlockedError
	if !errors.As(err, &blocked) || blocked.Report.Score != 3 || len(blocked.Report.Findings) != 1 {
		t.Fatalf("unexpected As result: %#v", blocked)
	}
}
