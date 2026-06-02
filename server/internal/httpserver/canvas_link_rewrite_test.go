package httpserver

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCanvasURLHostMatches_AllowedSuffix(t *testing.T) {
	base := "https://byui.instructure.com"
	suffixes := []string{"instructure.com"}
	if !canvasURLHostMatches("eu01.instructure.com", base, suffixes) {
		t.Fatal("expected another instructure subdomain to match suffix policy")
	}
	if !canvasURLHostMatches("byui.instructure.com", base, suffixes) {
		t.Fatal("expected exact canvas host match")
	}
	if canvasURLHostMatches("example.com", base, suffixes) {
		t.Fatal("expected unrelated host to be rejected")
	}
}

func TestCanvasLinkRewriteCtx_RewriteAssignmentURL(t *testing.T) {
	localID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	rc := &canvasLinkRewriteCtx{
		CanvasBase:          "https://school.instructure.com",
		CanvasCourseID:      42,
		CourseCode:          "C-TEST",
		Assignments:         map[int64]uuid.UUID{99: localID},
		AllowedHostSuffixes: []string{"instructure.com"},
	}
	raw := "https://other.instructure.com/courses/42/assignments/99"
	got := rc.rewriteURL(raw)
	want := "/courses/C-TEST/modules/assignment/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if got != want {
		t.Fatalf("rewriteURL() = %q, want %q", got, want)
	}
}

func TestCanvasLinkRewriteCtx_RewriteMarkdownAndHTML(t *testing.T) {
	localID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	rc := &canvasLinkRewriteCtx{
		CanvasBase:          "https://school.instructure.com",
		CanvasCourseID:      7,
		CourseCode:          "C-XXZQQA",
		Assignments:         map[int64]uuid.UUID{123: localID},
		AllowedHostSuffixes: []string{"instructure.com"},
	}
	md := strings.Join([]string{
		"Read the [syllabus](https://school.instructure.com/courses/7/assignments/123).",
		`<a href="/courses/7/assignments/123">syllabus</a>`,
		"<https://school.instructure.com/courses/7/assignments/123>",
	}, "\n")
	got := rc.rewriteMarkdown(md)
	wantPath := "/courses/C-XXZQQA/modules/assignment/11111111-2222-3333-4444-555555555555"
	if !strings.Contains(got, "[syllabus]("+wantPath+")") {
		t.Fatalf("expected markdown link rewrite, got:\n%s", got)
	}
	if !strings.Contains(got, `href="`+wantPath+`"`) {
		t.Fatalf("expected HTML anchor rewrite, got:\n%s", got)
	}
	if !strings.Contains(got, "<"+wantPath+">") {
		t.Fatalf("expected angle autolink rewrite, got:\n%s", got)
	}
	if strings.Contains(got, "instructure.com") {
		t.Fatalf("expected no Canvas hosts left, got:\n%s", got)
	}
}
