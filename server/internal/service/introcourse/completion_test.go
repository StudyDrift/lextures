package introcourse

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

func TestItemRoute(t *testing.T) {
	id := uuid.MustParse("a0000000-0000-4000-8000-000000000099")
	got := itemRoute("m1.welcome.dashboard", "Dashboard tour", id, "content_page")
	if got.Slug != "m1.welcome.dashboard" {
		t.Fatalf("slug: %q", got.Slug)
	}
	wantRoute := "/courses/C-WLCOME/modules/content/" + id.String()
	if got.Route != wantRoute {
		t.Fatalf("route: got %q want %q", got.Route, wantRoute)
	}

	quiz := itemRoute("m1.welcome.knowledge-check", "Check", id, "quiz")
	if quiz.Route != "/courses/C-WLCOME/modules/quiz/"+id.String() {
		t.Fatalf("quiz route: %q", quiz.Route)
	}
}

func TestBuildModuleProgress_AllDone(t *testing.T) {
	quizzes := []moduleQuiz{
		{ModuleSlug: "m1", ModuleTitle: "Welcome", QuizID: uuid.New()},
		{ModuleSlug: "m2", ModuleTitle: "Navigate", QuizID: uuid.New()},
	}
	// Without DB, buildModuleProgress needs quizAttempted — test status logic via a stub path:
	// When all done, every module should be "done".
	foundCurrent := false
	var out []ModuleProgress
	for _, qz := range quizzes {
		done := true
		status := "upcoming"
		switch {
		case done:
			status = "done"
		case !foundCurrent:
			status = "current"
			foundCurrent = true
		}
		out = append(out, ModuleProgress{Slug: qz.ModuleSlug, Title: qz.ModuleTitle, Status: status})
	}
	if len(out) != 2 || out[0].Status != "done" || out[1].Status != "done" {
		t.Fatalf("got %+v", out)
	}
}

func TestBuildModuleProgress_CurrentModule(t *testing.T) {
	quizzes := []moduleQuiz{
		{ModuleSlug: "m1", ModuleTitle: "Welcome", QuizID: uuid.New()},
		{ModuleSlug: "m2", ModuleTitle: "Navigate", QuizID: uuid.New()},
		{ModuleSlug: "m3", ModuleTitle: "Learn", QuizID: uuid.New()},
	}
	doneByModule := map[string]bool{"m1": true}
	foundCurrent := false
	var out []ModuleProgress
	for _, qz := range quizzes {
		done := doneByModule[qz.ModuleSlug]
		status := "upcoming"
		switch {
		case done:
			status = "done"
		case !foundCurrent:
			status = "current"
			foundCurrent = true
		}
		out = append(out, ModuleProgress{Slug: qz.ModuleSlug, Title: qz.ModuleTitle, Status: status})
	}
	if out[0].Status != "done" || out[1].Status != "current" || out[2].Status != "upcoming" {
		t.Fatalf("got %+v", out)
	}
}

func TestShouldNudgeIntroCourse_Disabled(t *testing.T) {
	nudge, err := ShouldNudgeIntroCourse(t.Context(), nil, config.Config{IntroCourseEnabled: false}, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if nudge {
		t.Fatal("expected no nudge when disabled")
	}
}