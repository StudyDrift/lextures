package mail

import (
	"strings"
	"testing"
)

func TestRenderSubmissionCommentIncludesUnsubscribe(t *testing.T) {
	rendered, err := RenderTemplate("submission_comment", map[string]string{
		"courseName":     "Writing",
		"assignmentName": "Essay",
		"excerpt":        "Nice work",
		"link":           "https://example.test/courses/C-1/modules/assignment/item",
		"unsubscribeUrl": "https://example.test/unsubscribe?token=abc",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.BodyText, "Nice work") || !strings.Contains(rendered.HTMLBody, "unsubscribe?token=abc") {
		t.Fatalf("rendered %#v", rendered)
	}
}

func TestRenderParentWeeklyProgress(t *testing.T) {
	rendered, err := RenderTemplate("parent_weekly_progress", map[string]string{
		"subject":        "Weekly progress",
		"bodyText":       "Hi Pat,\n\nNo learning activity this week.\n",
		"bodyHtml":       "<p>No learning activity this week.</p>",
		"link":           "https://example.test/parent",
		"unsubscribeUrl": "https://example.test/unsubscribe?token=abc",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.BodyText, "No learning activity") || !strings.Contains(rendered.HTMLBody, "Unsubscribe") {
		t.Fatalf("rendered %#v", rendered)
	}
}
