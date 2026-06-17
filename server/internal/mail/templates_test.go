package mail

import (
	"strings"
	"testing"
)

func TestRenderCertificateIssued(t *testing.T) {
	rendered, err := RenderTemplate("certificate_issued", map[string]string{
		"credentialName": "Intro to Data Science",
		"verifyUrl":      "http://localhost:5173/verify/abc",
		"linkedInUrl":    "https://www.linkedin.com/profile/add?name=Intro",
		"credentialsUrl": "http://localhost:5173/me/credentials",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.Subject, "Intro to Data Science") {
		t.Fatalf("subject: %q", rendered.Subject)
	}
	if !strings.Contains(rendered.HTMLBody, "Add to LinkedIn") {
		t.Fatalf("html missing LinkedIn CTA")
	}
}

func TestRenderGradePosted(t *testing.T) {
	rendered, err := RenderTemplate("grade_posted", map[string]string{
		"courseName":     "Algebra I",
		"assignmentName": "Quiz 2",
		"link":           "http://localhost:5173/courses/ALG101/grades",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.Subject, "Algebra I") {
		t.Fatalf("subject: %q", rendered.Subject)
	}
	if !strings.Contains(rendered.HTMLBody, "Quiz 2") {
		t.Fatalf("html missing assignment")
	}
}
