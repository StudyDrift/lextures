package notifications

import "testing"

func TestSubmissionCommentInAppContent(t *testing.T) {
	title, body, url := SubmissionCommentInAppContent("Essay", "Writing", "C-1", "assignment", "item-1", "cmt-9")
	if title != "New comment on Essay" {
		t.Fatalf("title %q", title)
	}
	if body == "" || url != "/courses/C-1/modules/assignment/item-1?comment=cmt-9" {
		t.Fatalf("body %q url %q", body, url)
	}
	_, _, quiz := SubmissionCommentInAppContent("Quiz", "Math", "C-1", "quiz", "item-2", "cmt-2")
	if quiz != "/courses/C-1/modules/quiz/item-2?comment=cmt-2" {
		t.Fatalf("quiz url %q", quiz)
	}
	_, _, other := SubmissionCommentInAppContent("Page", "Math", "C-1", "content_page", "item-3", "cmt-3")
	if other != "/courses/C-1/my-grades" {
		t.Fatalf("other url %q", other)
	}
}
