package notifications

import "testing"

func TestGradePostedInAppContent(t *testing.T) {
	title, body, url := GradePostedInAppContent("Math 101", "QA Assignment", "C-ABC123")
	if title != "Grade posted for QA Assignment" {
		t.Fatalf("title = %q", title)
	}
	if body == "" {
		t.Fatal("body should not be empty")
	}
	if url != "/courses/C-ABC123/grades" {
		t.Fatalf("actionURL = %q", url)
	}
}

func TestCanEmailRecipient(t *testing.T) {
	cases := map[string]bool{
		"kid@school.edu":        true,
		"  parent@example.com ": true,
		"":                      false,
		"3f2c1a9e-0000-0000-0000-000000000000@managed.lextures.invalid": false,
		"3F2C1A9E-0000-0000-0000-000000000000@MANAGED.LEXTURES.INVALID": false,
	}
	for email, want := range cases {
		if got := canEmailRecipient(email); got != want {
			t.Errorf("canEmailRecipient(%q) = %v, want %v", email, got, want)
		}
	}
}
