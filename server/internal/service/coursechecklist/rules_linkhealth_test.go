package coursechecklist

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/coursechecklist/linkhealth"
)

func snapWithExternalLink() CourseSnapshot {
	id := uuid.New()
	return CourseSnapshot{
		StructureItems: []StructureItem{{ID: id, Kind: "content_page", Title: "Lesson", Published: true}},
		ItemMeta: map[uuid.UUID]ItemMeta{
			id: {BodyMarkdown: "See [docs](https://example.com/guide)."},
		},
	}
}

func TestEvalLinksExternalHealthDisabledWhenCheckingIsOff(t *testing.T) {
	t.Setenv("CHECKLIST_LINKCHECK_ENABLED", "")
	snap := snapWithExternalLink()
	snap.Lazy = map[LazyLoaderID]any{
		LazyLinkHealth: LinkHealthLazy{Disabled: true},
	}
	f, err := evalLinksExternalHealth(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusUnknown {
		t.Fatalf("status=%s", f.Status)
	}
	if f.DetailDefault != "Outbound link checking is turned off." {
		t.Fatalf("detail=%q", f.DetailDefault)
	}
}

func TestEvalLinksExternalHealthPendingStaysCheckingWhenEnabled(t *testing.T) {
	t.Setenv("CHECKLIST_LINKCHECK_ENABLED", "true")
	snap := snapWithExternalLink()
	snap.Lazy = map[LazyLoaderID]any{
		LazyLinkHealth: LinkHealthLazy{Pending: true},
	}
	f, err := evalLinksExternalHealth(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusUnknown || f.DetailDefault != "Checking links…" {
		t.Fatalf("status=%s detail=%q", f.Status, f.DetailDefault)
	}
}

func TestEvalLinksExternalHealthPendingSaysOffWhenDisabled(t *testing.T) {
	t.Setenv("CHECKLIST_LINKCHECK_ENABLED", "false")
	snap := snapWithExternalLink()
	snap.Lazy = map[LazyLoaderID]any{
		LazyLinkHealth: LinkHealthLazy{Pending: true},
	}
	f, err := evalLinksExternalHealth(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.DetailDefault != "Outbound link checking is turned off." {
		t.Fatalf("detail=%q", f.DetailDefault)
	}
}

func TestEvalLinksExternalHealthNoLinksIsDone(t *testing.T) {
	t.Setenv("CHECKLIST_LINKCHECK_ENABLED", "true")
	f, err := evalLinksExternalHealth(context.Background(), CourseSnapshot{
		Lazy: map[LazyLoaderID]any{LazyLinkHealth: LinkHealthLazy{Pending: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusDone || f.DetailDefault != "No external links to check." {
		t.Fatalf("status=%s detail=%q", f.Status, f.DetailDefault)
	}
}

func TestEvalLinksExternalHealthUsesFreshCache(t *testing.T) {
	t.Setenv("CHECKLIST_LINKCHECK_ENABLED", "false")
	code := 404
	snap := snapWithExternalLink()
	snap.Lazy = map[LazyLoaderID]any{
		LazyLinkHealth: LinkHealthLazy{
			Pending: false,
			Rows: []linkhealth.Row{{
				URL:        "https://example.com/guide",
				Result:     linkhealth.ResultDead,
				StatusCode: &code,
			}},
		},
	}
	f, err := evalLinksExternalHealth(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("status=%s evidence=%d detail=%q", f.Status, len(f.Evidence), f.DetailDefault)
	}
}
