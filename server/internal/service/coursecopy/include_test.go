package coursecopy

import "testing"

func TestInclude_WithDefaults_AllFalseGivesAll(t *testing.T) {
	got := (Include{}).WithDefaults()
	want := Include{Modules: true, Assignments: true, Quizzes: true, Enrollments: true, Grades: true, Settings: true, Files: true}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestInclude_WithDefaults_PartialUnchanged(t *testing.T) {
	partial := Include{Modules: true, Enrollments: true}
	if got := partial.WithDefaults(); got != partial {
		t.Fatalf("got %+v want %+v", got, partial)
	}
}

func TestInclude_shouldCopyKind(t *testing.T) {
	inc := Include{Modules: true}
	if !inc.shouldCopyKind("content_page") {
		t.Fatal("expected content_page with modules")
	}
	if inc.shouldCopyKind("assignment") {
		t.Fatal("assignment should require assignments flag")
	}
	inc2 := Include{Assignments: true}
	if !inc2.shouldCopyKind("module") {
		t.Fatal("module shell needed for assignments")
	}
}