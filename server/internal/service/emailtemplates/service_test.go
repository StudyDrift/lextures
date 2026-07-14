package emailtemplates

import (
	"strings"
	"testing"

	emailtemplatesrepo "github.com/lextures/lextures/server/internal/repos/emailtemplates"
)

func TestPreview_derivesTextFromCompiledMarkdown(t *testing.T) {
	var svc Service
	got := svc.Preview("**Hello** [go]({{link}})", nil, map[string]string{
		"link": "https://example.edu/x",
	})
	if !strings.Contains(got.HTML, "Hello") {
		t.Fatalf("html=%q", got.HTML)
	}
	if strings.Contains(got.Text, "<") {
		t.Fatalf("text should be tag-free, got %q", got.Text)
	}
	if !strings.Contains(got.Text, "Hello") {
		t.Fatalf("text=%q", got.Text)
	}
	if strings.Contains(got.HTML, "{{link}}") {
		t.Fatalf("link should be merged in preview: %q", got.HTML)
	}
	if !strings.Contains(got.HTML, "https://example.edu/x") {
		t.Fatalf("merged link missing: %q", got.HTML)
	}
}

func TestValidateUnknownMarkdown(t *testing.T) {
	slot := &emailtemplatesrepo.Slot{
		MergeFields: map[string]string{"link": "x", "user.first_name": "y"},
	}
	unknown := ValidateUnknownMarkdown(slot, "Hi {{user.first_name}} {{bogus}}", nil)
	if len(unknown) != 1 || unknown[0] != "bogus" {
		t.Fatalf("unknown=%v", unknown)
	}
}

func TestSubjectForSlot_systemDefaults(t *testing.T) {
	s := subjectForSlot(&emailtemplatesrepo.Slot{ID: "magic_link", Description: "Passwordless sign-in link"}, nil)
	if s != "Your StudyDrift sign-in link" {
		t.Fatalf("subject=%q", s)
	}
	s = subjectForSlot(&emailtemplatesrepo.Slot{ID: "coppa_consent"}, map[string]string{"student.name": "Sam"})
	if !strings.Contains(s, "Sam") {
		t.Fatalf("subject=%q", s)
	}
}
