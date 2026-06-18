package sms

import "testing"

func TestBuildMessage(t *testing.T) {
	got := BuildMessage("Grade posted", "Your grade for Quiz 1 has been posted.", "https://app.example/courses/demo/grades")
	want := "Grade posted — Your grade for Quiz 1 has been posted. — https://app.example/courses/demo/grades"
	if got != want {
		t.Fatalf("BuildMessage() = %q, want %q", got, want)
	}
}

func TestBuildMessageTitleOnly(t *testing.T) {
	got := BuildMessage("New message", "", "")
	if got != "New message" {
		t.Fatalf("BuildMessage() = %q", got)
	}
}