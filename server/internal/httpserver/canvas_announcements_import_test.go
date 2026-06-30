package httpserver

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanvasImportInclude_WithDefaults_LegacyAllExceptFilesEnablesAnnouncements(t *testing.T) {
	legacy := canvasImportInclude{Modules: true, Assignments: true, Quizzes: true, Enrollments: true, Grades: true, Settings: true}
	got := legacy.withDefaults()
	if !got.Files || !got.Announcements {
		t.Fatalf("legacy all-true include should default Files and Announcements=true, got %+v", got)
	}
}

func TestCanvasAnnouncementFeedBody(t *testing.T) {
	got := canvasAnnouncementFeedBody("Exam next week", "<p>Bring a pencil.</p>")
	if !strings.Contains(got, "Exam next week") || !strings.Contains(got, "Bring a pencil.") {
		t.Fatalf("unexpected body: %q", got)
	}
	if !strings.Contains(got, "\n\n") {
		t.Fatalf("expected title/body separator, got: %q", got)
	}
}

func TestCanvasAnnouncementFeedBody_TitleOnly(t *testing.T) {
	got := canvasAnnouncementFeedBody("Quick note", "")
	if got != "Quick note" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateFeedMessageBody(t *testing.T) {
	long := strings.Repeat("a", maxFeedMessageBodyLen+10)
	got := truncateFeedMessageBody(long)
	if utf8Count := len([]rune(got)); utf8Count != maxFeedMessageBodyLen {
		t.Fatalf("truncated length = %d, want %d", utf8Count, maxFeedMessageBodyLen)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis suffix, got %q", got[len(got)-5:])
	}
}

func TestCanvasAnnouncementAuthorID(t *testing.T) {
	importer := uuid.New()
	mapped := uuid.New()
	canvasUID := int64(42)
	topic := map[string]any{"user_id": float64(canvasUID)}
	got := canvasAnnouncementAuthorID(topic, map[int64]uuid.UUID{canvasUID: mapped}, importer)
	if got != mapped {
		t.Fatalf("got %v want mapped %v", got, mapped)
	}
	got = canvasAnnouncementAuthorID(topic, nil, importer)
	if got != importer {
		t.Fatalf("fallback author = %v want %v", got, importer)
	}
}

func TestCanvasImportAnnouncementsSortsByPostedAt(t *testing.T) {
	older := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	topics := []map[string]any{
		{"id": float64(2), "title": "B", "message": "b", "posted_at": newer.Format(time.RFC3339), "published": true},
		{"id": float64(1), "title": "A", "message": "a", "posted_at": older.Format(time.RFC3339), "published": true},
	}
	sortTopics := append([]map[string]any(nil), topics...)
	sort.Slice(sortTopics, func(i, j int) bool {
		ti := canvasTimeAt(sortTopics[i], "posted_at")
		tj := canvasTimeAt(sortTopics[j], "posted_at")
		return ti.Before(*tj)
	})
	if strAt(sortTopics[0], "title", "") != "A" {
		t.Fatalf("expected older announcement first, got %+v", sortTopics)
	}
}
