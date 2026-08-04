package coursechecklist

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestSyllabusCopyContract(t *testing.T) {
	for _, it := range syllabusRules() {
		if utf8.RuneCountInString(it.TitleDefault) > 60 {
			t.Errorf("%s title too long", it.ID)
		}
		low := strings.ToLower(it.TitleDefault + " " + it.WhyDefault)
		for _, ban := range []string{"failed", "should have", "!"} {
			if strings.Contains(low, ban) {
				t.Errorf("%s banned %q", it.ID, ban)
			}
		}
	}
}

func TestSyllabusLatePolicyAC11(t *testing.T) {
	// AC-11
	items := make([]StructureItem, 0, 5)
	meta := map[uuid.UUID]ItemMeta{}
	for i := 0; i < 5; i++ {
		id := uuid.New()
		items = append(items, StructureItem{ID: id, Kind: "assignment", Title: "A" + string(rune('1'+i))})
		meta[id] = ItemMeta{Kind: "assignment", LateSubmissionPolicy: "allow"}
	}
	f, err := findRule(t, ItemSyllabusLatePolicy).Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage: "en",
		SyllabusSections: []SyllabusSectionSnap{{
			Title: "Policies", Markdown: "No late work accepted under any circumstances.",
		}},
		StructureItems: items,
		ItemMeta:       meta,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusInProgress {
		t.Fatalf("status=%s detail=%q", f.Status, f.DetailDefault)
	}
	if len(f.Evidence) != 5 {
		t.Fatalf("evidence=%d", len(f.Evidence))
	}
}

func TestSyllabusMalformedAC13(t *testing.T) {
	// AC-13 — all syllabus rules unknown; non-syllabus unaffected
	snap := CourseSnapshot{
		SyllabusMalformed: true,
		Title:             "Real Course",
		Description:       strings.Repeat("x", 120),
		FeedEnabled:       true,
		Features: CourseFeatures{
			DiscussionsEnabled: true,
			FeedEnabled:        true,
		},
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{})
	for _, fr := range res.Findings {
		switch {
		case strings.HasPrefix(string(fr.ID), "syllabus."):
			if fr.Finding.Status != StatusUnknown {
				t.Errorf("%s want unknown got %s", fr.ID, fr.Finding.Status)
			}
		case fr.ID == ItemOrientationResponseTime || fr.ID == ItemOrientationInstructorContact ||
			fr.ID == ItemOrientationParticipationExpectations || fr.ID == ItemOrientationNetiquette ||
			fr.ID == ItemOrientationTechRequirements || fr.ID == ItemOrientationSupportResources:
			if fr.Finding.Status != StatusUnknown {
				t.Errorf("%s (syllabus text) want unknown got %s", fr.ID, fr.Finding.Status)
			}
		case fr.ID == ItemCourseTitleAndDescription:
			if fr.Finding.Status == StatusUnknown {
				t.Errorf("%s should not be unknown", fr.ID)
			}
		}
	}
}

func TestSyllabusExistsAndAcceptance(t *testing.T) {
	f, _ := findRule(t, ItemSyllabusExists).Evaluate(context.Background(), CourseSnapshot{
		SyllabusSections: []SyllabusSectionSnap{
			{Title: "A", Markdown: strings.Repeat("m", 100)},
		},
	})
	if f.Status != StatusTodo {
		t.Fatalf("short syllabus=%s", f.Status)
	}
	f, _ = findRule(t, ItemSyllabusExists).Evaluate(context.Background(), CourseSnapshot{
		SyllabusSections: []SyllabusSectionSnap{
			{Title: "A", Markdown: strings.Repeat("m", 400)},
			{Title: "B", Markdown: strings.Repeat("n", 400)},
		},
	})
	if f.Status != StatusDone {
		t.Fatalf("full syllabus=%s", f.Status)
	}

	acc := findRule(t, ItemSyllabusAcceptanceDecision)
	f, _ = acc.Evaluate(context.Background(), CourseSnapshot{})
	if f.Status != StatusTodo {
		t.Fatalf("undecided=%s", f.Status)
	}
	now := time.Now().UTC()
	f, _ = acc.Evaluate(context.Background(), CourseSnapshot{AcceptanceDecidedAt: &now})
	if f.Status != StatusDone {
		t.Fatalf("decided=%s", f.Status)
	}
}

func TestSyllabusPrintable(t *testing.T) {
	f, _ := findRule(t, ItemSyllabusPrintable).Evaluate(context.Background(), CourseSnapshot{
		SyllabusSections: []SyllabusSectionSnap{{Markdown: "Hello <iframe src='x'></iframe>"}},
	})
	if f.Status != StatusTodo || len(f.Evidence) == 0 {
		t.Fatalf("got %+v", f)
	}
}
