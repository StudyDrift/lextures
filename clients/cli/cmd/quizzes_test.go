package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type quizzesServerConfig struct {
	structureHandler http.HandlerFunc
	quizGetHandler   http.HandlerFunc
	quizPatchHandler http.HandlerFunc
	quizCreateHandler http.HandlerFunc
}

func newQuizzesServer(t *testing.T, cfg quizzesServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/structure"):
			if cfg.structureHandler != nil {
				cfg.structureHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/quizzes/"):
			if cfg.quizGetHandler != nil {
				cfg.quizGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/quizzes/"):
			if cfg.quizPatchHandler != nil {
				cfg.quizPatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/quizzes"):
			if cfg.quizCreateHandler != nil {
				cfg.quizCreateHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func sampleQuizStructureItems() []structureItemPublic {
	due := time.Date(2027, 5, 1, 12, 0, 0, 0, time.UTC)
	pts := 10
	modParent := "mod-1"
	return []structureItemPublic{
		{ID: "mod-1", Kind: "module", Title: "Week 1", Published: true},
		{ID: "quiz-1", Kind: "quiz", Title: "Quiz 1", Published: true, ParentID: &modParent, PointsWorth: &pts, DueAt: &due},
		{ID: "assign-1", Kind: "assignment", Title: "HW", Published: true, ParentID: &modParent},
	}
}

func resetQuizzesFlags() {
	quizzesListFlags.course = ""
	quizzesListFlags.limit = 50
	quizzesListFlags.page = 1
	quizzesGetFlags.course = ""
	quizzesCreateFlags.course = ""
	quizzesCreateFlags.module = ""
	quizzesCreateFlags.title = ""
	quizzesCreateFlags.points = -1
}

func TestQuizzesList_FiltersQuizItems(t *testing.T) {
	srv := newQuizzesServer(t, quizzesServerConfig{
		structureHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: sampleQuizStructureItems()})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesFlags()
	quizzesListFlags.course = "CS101"

	var out bytes.Buffer
	quizzesListCmd.SetOut(&out)
	if err := quizzesListCmd.RunE(quizzesListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "quiz-1") {
		t.Errorf("output = %q; want quiz-1", out.String())
	}
	if strings.Contains(out.String(), "assign-1") {
		t.Errorf("output should not include assignments: %q", out.String())
	}
}

func TestQuizzesList_JSONOutput(t *testing.T) {
	srv := newQuizzesServer(t, quizzesServerConfig{
		structureHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: sampleQuizStructureItems()})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesFlags()
	quizzesListFlags.course = "CS101"
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	quizzesListCmd.SetOut(&out)
	if err := quizzesListCmd.RunE(quizzesListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result []map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 quiz, got %d", len(result))
	}
}

func TestQuizzesGet_Success(t *testing.T) {
	srv := newQuizzesServer(t, quizzesServerConfig{
		quizGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"itemId": "quiz-1",
				"title":  "Quiz 1",
				"questions": []map[string]any{
					{"id": "q1", "prompt": "2+2?", "questionType": "multiple_choice", "points": 1},
				},
				"updatedAt": time.Now().UTC().Format(time.RFC3339),
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesFlags()
	quizzesGetFlags.course = "CS101"

	var out bytes.Buffer
	quizzesGetCmd.SetOut(&out)
	if err := quizzesGetCmd.RunE(quizzesGetCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Quiz 1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestQuizzesCreate_Success(t *testing.T) {
	var gotTitle string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/quizzes") {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotTitle = body["title"]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(structureItemPublic{ID: "new-quiz", Title: body["title"], Kind: "quiz"})
		}
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesFlags()
	quizzesCreateFlags.course = "CS101"
	quizzesCreateFlags.module = "mod-1"
	quizzesCreateFlags.title = "Midterm"

	var out bytes.Buffer
	quizzesCreateCmd.SetOut(&out)
	if err := quizzesCreateCmd.RunE(quizzesCreateCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotTitle != "Midterm" {
		t.Errorf("title sent = %q, want Midterm", gotTitle)
	}
	if !strings.Contains(out.String(), "new-quiz") {
		t.Errorf("output = %q", out.String())
	}
}

func TestQuizzesCmd_HasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range quizzesCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "get", "create", "update", "delete", "publish", "questions", "attempts", "grade", "grade-sync"} {
		if !names[want] {
			t.Errorf("quizzes subcommand %q not registered", want)
		}
	}
}