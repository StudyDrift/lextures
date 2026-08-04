package coursechecklist

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestA11yImageAltTextProgressAndDecorative(t *testing.T) {
	pageID := uuid.New()
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: pageID, Kind: "content_page", Title: "Lesson", Published: true},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			pageID: {Kind: "content_page", BodyMarkdown: "" +
				"![ok](a.png)\n" +
				"![](b.png)\n" +
				"![](c.png \"lex-decorative\")\n" +
				"![also](d.png)\n",
			},
		},
	}
	f, err := evalA11yImageAltText(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusInProgress {
		t.Fatalf("status=%s want in_progress", f.Status)
	}
	if f.Progress == nil || f.Progress.Done != 2 || f.Progress.Total != 3 {
		t.Fatalf("progress=%v want {2,3}", f.Progress)
	}
	if len(f.Evidence) != 1 {
		t.Fatalf("evidence=%d want 1 (decorative excluded)", len(f.Evidence))
	}
}

func TestA11yHeadingSkip(t *testing.T) {
	pageID := uuid.New()
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: pageID, Kind: "content_page", Title: "Lesson", Published: true},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			pageID: {BodyMarkdown: "## Intro\n\n#### Skipped\n"},
		},
	}
	f, err := evalA11yHeadingStructure(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("status=%s evidence=%d", f.Status, len(f.Evidence))
	}
	if !strings.Contains(f.Evidence[0].Sublabel, "Skipped") {
		t.Fatalf("sublabel=%q", f.Evidence[0].Sublabel)
	}
}

func TestA11yVideoCaptionsNA(t *testing.T) {
	f, err := evalA11yVideoCaptions(context.Background(), CourseSnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusNotApplicable {
		t.Fatalf("status=%s", f.Status)
	}
}

func TestA11yColorContrastDetail(t *testing.T) {
	// #777777 on white ≈ 4.48:1 — use a worse pair for AC-5 style 3.1:1.
	// #949494 on #ffffff is about 3.1:1.
	custom, _ := json.Marshal(map[string]string{
		"bodyColor": "#949494",
	})
	f, err := evalA11yColorContrast(context.Background(), CourseSnapshot{
		MarkdownThemePreset: "custom",
		MarkdownThemeCustom: custom,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s", f.Status)
	}
	if !strings.Contains(f.DetailDefault, "needs 4.5:1") {
		t.Fatalf("detail=%q", f.DetailDefault)
	}
}

func TestA11yDocumentAccessibility(t *testing.T) {
	fid := uuid.New()
	snap := CourseSnapshot{
		Files: []FileSnap{
			{ID: fid, DisplayName: "scan.pdf", ContentType: "application/pdf", TextLayer: "image_only"},
			{ID: uuid.New(), DisplayName: "ok.pdf", ContentType: "application/pdf", TextLayer: "has_text"},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			uuid.New(): {EmbeddedFileIDs: []uuid.UUID{fid}},
		},
	}
	f, err := evalA11yDocumentAccessibility(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("status=%s evidence=%v", f.Status, f.Evidence)
	}
}

func TestA11yDisclaimerInWhyCopy(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range reg.List() {
		if !strings.HasPrefix(string(it.ID), "a11y.") && !strings.HasPrefix(string(it.ID), "udl.") {
			continue
		}
		if !strings.Contains(strings.ToLower(it.WhyDefault), "automated") ||
			!strings.Contains(strings.ToLower(it.WhyDefault), "partial") {
			t.Errorf("%s why missing automated/partial disclaimer: %q", it.ID, it.WhyDefault)
		}
		if !strings.Contains(it.WhyDefault, "docs/accessibility/") {
			t.Errorf("%s why missing docs/accessibility link: %q", it.ID, it.WhyDefault)
		}
	}
}

func TestContentDocParsedOncePerEvaluation(t *testing.T) {
	pageID := uuid.New()
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: pageID, Kind: "content_page", Title: "Lesson", Published: true},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			pageID: {BodyMarkdown: "## Hello\n\n![x](a.png)\n"},
		},
		Lazy: map[LazyLoaderID]any{
			LazyLinkHealth: LinkHealthLazy{Pending: false},
		},
	}
	doc1 := EnsureContentDoc(&snap)
	doc2 := EnsureContentDoc(&snap)
	if doc1 != doc2 {
		t.Fatal("EnsureContentDoc re-parsed instead of reusing ContentDoc")
	}
	if doc1.ParseCount != 1 {
		t.Fatalf("ParseCount=%d want 1", doc1.ParseCount)
	}
	reg, err := NewRegistry(a11yRules())
	if err != nil {
		t.Fatal(err)
	}
	_ = Evaluate(context.Background(), snap, EvaluateOptions{Registry: reg})
	if snap.ContentDoc != doc1 {
		t.Fatal("Evaluate should reuse pre-attached ContentDoc (AC-13)")
	}
}

func TestLaunchStudentPreview(t *testing.T) {
	f, err := evalLaunchStudentPreview(context.Background(), CourseSnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s", f.Status)
	}
	now := time.Now().UTC()
	prev := now.Add(-time.Hour)
	changed := now.Add(-2 * time.Hour)
	f, err = evalLaunchStudentPreview(context.Background(), CourseSnapshot{
		StudentPreviewAt:   &prev,
		StructureChangedAt: &changed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone {
		t.Fatalf("status=%s want done", f.Status)
	}
}

func TestLaunchNoDraftsAfterStart(t *testing.T) {
	due := time.Now().UTC().Add(5 * 24 * time.Hour)
	start := time.Now().UTC().Add(-24 * time.Hour)
	id := uuid.New()
	snap := CourseSnapshot{
		StartsAt: &start,
		AssessmentItems: []AssessmentItemSnap{
			{ID: id, Kind: "quiz", Title: "Quiz 1", DueAt: &due, Published: false},
		},
	}
	f, err := evalLaunchNoDraftsAfterStart(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("status=%s evidence=%d", f.Status, len(f.Evidence))
	}
}

func TestContrastRatioReference(t *testing.T) {
	// Black on white = 21:1
	ratio := contrastRatio(relativeLuminance(0, 0, 0), relativeLuminance(255, 255, 255))
	if ratio < 20.9 || ratio > 21.1 {
		t.Fatalf("ratio=%v", ratio)
	}
}
