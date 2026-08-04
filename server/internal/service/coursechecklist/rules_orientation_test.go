package coursechecklist

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestOrientationCopyContract(t *testing.T) {
	for _, it := range orientationRules() {
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

func TestWelcomeMessageAC4AC5(t *testing.T) {
	rule := findRule(t, ItemOrientationWelcomeMessage)
	// AC-4
	posted := time.Now().UTC()
	f, err := rule.Evaluate(context.Background(), CourseSnapshot{
		FeedEnabled: true,
		CreatedAt:   posted.Add(-time.Hour),
		AnnouncementsWelcome: &WelcomeMessageSnap{
			BodyLen: 400, AuthorIsStaff: true, PostedAt: &posted,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone {
		t.Fatalf("AC-4 status=%s", f.Status)
	}
	// AC-5 — student short post: no StaffWelcome
	f, err = rule.Evaluate(context.Background(), CourseSnapshot{
		FeedEnabled: true,
		FeedChannels: []FeedChannelSnap{
			{Name: "announcements", LatestTitle: "hi everyone!!"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("AC-5 status=%s", f.Status)
	}
}

func TestResponseTimeAC6AC7(t *testing.T) {
	rule := findRule(t, ItemOrientationResponseTime)
	// AC-6 done
	f, _ := rule.Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage: "en",
		SyllabusSections: []SyllabusSectionSnap{{
			Title: "Contact", Markdown: "I respond to email within 24 hours for all questions.",
		}},
	})
	if f.Status != StatusDone {
		t.Fatalf("AC-6 done=%s detail=%q", f.Status, f.DetailDefault)
	}
	// AC-6 todo
	f, _ = rule.Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage:  "en",
		SyllabusSections: []SyllabusSectionSnap{{Title: "Overview", Markdown: "Welcome to the course."}},
	})
	if f.Status != StatusTodo {
		t.Fatalf("AC-6 todo=%s", f.Status)
	}
	// AC-7 Spanish
	f, _ = rule.Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage: "es",
		SyllabusSections: []SyllabusSectionSnap{{
			Title: "Contacto", Markdown: "Les responderé en 24 horas por correo.",
		}},
	})
	if f.Status != StatusDone {
		t.Fatalf("AC-7 status=%s detail=%q", f.Status, f.DetailDefault)
	}
}

func TestStartHereAndNetiquetteNA(t *testing.T) {
	mod := uuid.New()
	page := uuid.New()
	f, _ := findRule(t, ItemOrientationStartHere).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "Week 0", SortOrder: 0},
			{ID: page, Kind: "content_page", Title: "Start Here", ParentID: &mod, SortOrder: 0},
		},
	})
	if f.Status != StatusDone {
		t.Fatalf("start-here=%s", f.Status)
	}

	res := Evaluate(context.Background(), CourseSnapshot{
		Features: CourseFeatures{},
	}, EvaluateOptions{Only: []ItemID{ItemOrientationNetiquette}})
	if res.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("netiquette=%s", res.Findings[0].Finding.Status)
	}
}
