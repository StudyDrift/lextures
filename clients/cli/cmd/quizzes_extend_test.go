package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type quizzesExtendServerConfig struct {
	quizGetHandler      http.HandlerFunc
	quizPatchHandler    http.HandlerFunc
	attemptsHandler     http.HandlerFunc
	gradingGetHandler   http.HandlerFunc
	gradingPutHandler   http.HandlerFunc
	gradebookPutHandler http.HandlerFunc
	bankListHandler     http.HandlerFunc
	structurePatch      http.HandlerFunc
	structureDelete     http.HandlerFunc
}

func newQuizzesExtendServer(t *testing.T, cfg quizzesExtendServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.Contains(path, "/quizzes/") && strings.HasSuffix(path, "/attempts"):
			if cfg.attemptsHandler != nil {
				cfg.attemptsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/quizzes/") && strings.HasSuffix(path, "/grading"):
			if cfg.gradingGetHandler != nil {
				cfg.gradingGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.Contains(path, "/grading"):
			if cfg.gradingPutHandler != nil {
				cfg.gradingPutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/gradebook/grades"):
			if cfg.gradebookPutHandler != nil {
				cfg.gradebookPutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/question-banks/"):
			if cfg.bankListHandler != nil {
				cfg.bankListHandler(w, r)
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
		case r.Method == http.MethodPatch && strings.Contains(path, "/structure/items/"):
			if cfg.structurePatch != nil {
				cfg.structurePatch(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodDelete && strings.Contains(path, "/structure/items/"):
			if cfg.structureDelete != nil {
				cfg.structureDelete(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func sampleQuizJSON(questions []quizQuestion) []byte {
	q := quizPublic{
		ItemID:             "quiz-1",
		Title:              "Quiz 1",
		GradeAttemptPolicy: "highest",
		Questions:          questions,
	}
	raw, _ := json.Marshal(q)
	return raw
}

func resetQuizzesExtendFlags() {
	quizzesUpdateFlags = struct {
		course   string
		title    string
		points   int
		markdown string
		file     string
	}{}
	quizzesDeleteFlags.course = ""
	quizzesPublishFlags.course = ""
	quizzesSettingsSetFlags = struct {
		course          string
		timeLimit       int
		maxAttempts     int
		unlimited       bool
		shuffleQuestions bool
		shuffleChoices  bool
		availableFrom   string
		availableUntil  string
		policy          string
	}{timeLimit: -1, maxAttempts: -1}
	quizzesQuestionsAddFlags = struct {
		course  string
		bank    string
		count   int
		ids     []string
		content string
	}{}
	quizzesQuestionsRemoveFlags = struct {
		course string
		id     string
		index  int
	}{index: -1}
	quizzesQuestionsListFlags.course = ""
	quizzesQuestionsReorderFlags = struct {
		course string
		order  string
	}{}
	quizzesAttemptsListFlags = struct {
		course string
		user   string
		limit  int
		page   int
		yes    bool
	}{limit: 50, page: 1}
	quizzesAttemptsGetFlags.course = ""
	quizzesAttemptsGetFlags.user = ""
	quizzesGradeFlags = struct {
		course  string
		attempt string
		user    string
		all     bool
	}{}
	quizzesGradeSyncFlags.course = ""
	quizzesGradeSyncFlags.user = ""
}

func TestQuizzesQuestionsAdd_FromBank(t *testing.T) {
	var gotQuestions int
	srv := newQuizzesExtendServer(t, quizzesExtendServerConfig{
		quizGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(sampleQuizJSON(nil))
		},
		quizPatchHandler: func(w http.ResponseWriter, r *http.Request) {
			var patch map[string]any
			_ = json.NewDecoder(r.Body).Decode(&patch)
			qs, _ := patch["questions"].([]any)
			gotQuestions = len(qs)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(sampleQuizJSON([]quizQuestion{{ID: "bq-1", Prompt: "Q1", QuestionType: "multiple_choice", Points: 1}}))
		},
		bankListHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]questionPublic{
				sampleQuestion("bq-1", "multiple-choice", "What is 2+2?"),
				sampleQuestion("bq-2", "true-false", "Sky is blue"),
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesExtendFlags()
	quizzesQuestionsAddFlags.course = "CS101"
	quizzesQuestionsAddFlags.bank = "bank-1"
	quizzesQuestionsAddFlags.count = 1

	var out bytes.Buffer
	quizzesQuestionsAddCmd.SetOut(&out)
	if err := quizzesQuestionsAddCmd.RunE(quizzesQuestionsAddCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotQuestions != 1 {
		t.Errorf("patched questions = %d, want 1", gotQuestions)
	}
	if !strings.Contains(out.String(), "Added 1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestQuizzesSettingsSet_TimeLimit(t *testing.T) {
	var gotPatch map[string]any
	srv := newQuizzesExtendServer(t, quizzesExtendServerConfig{
		quizPatchHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotPatch)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(sampleQuizJSON(nil))
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesExtendFlags()
	quizzesSettingsSetFlags.course = "CS101"
	quizzesSettingsSetFlags.timeLimit = 45

	quizzesSettingsSetCmd.SetOut(&bytes.Buffer{})
	if err := quizzesSettingsSetCmd.RunE(quizzesSettingsSetCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotPatch["timeLimitMinutes"] != float64(45) {
		t.Errorf("timeLimitMinutes = %v, want 45", gotPatch["timeLimitMinutes"])
	}
}

func TestQuizzesAttemptsList_RequiresYesWithoutUser(t *testing.T) {
	setCfg("http://localhost:0", "test-key")
	resetQuizzesExtendFlags()
	quizzesAttemptsListFlags.course = "CS101"

	err := quizzesAttemptsListCmd.RunE(quizzesAttemptsListCmd, []string{"quiz-1"})
	if err == nil {
		t.Fatal("expected FERPA warning error")
	}
	if !strings.Contains(err.Error(), "FERPA") {
		t.Errorf("err = %v", err)
	}
}

func TestQuizzesAttemptsList_JSONWithYes(t *testing.T) {
	student := "user-1"
	srv := newQuizzesExtendServer(t, quizzesExtendServerConfig{
		attemptsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(quizAttemptsListBody{
				Attempts: []quizAttemptSummary{{
					ID:            "att-1",
					StudentUserID: &student,
					AttemptNumber: 1,
					PointsEarned:  floatPtr(8),
					PointsPossible: floatPtr(10),
				}},
				RetakePolicy: "highest",
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesExtendFlags()
	quizzesAttemptsListFlags.course = "CS101"
	quizzesAttemptsListFlags.yes = true
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	quizzesAttemptsListCmd.SetOut(&out)
	if err := quizzesAttemptsListCmd.RunE(quizzesAttemptsListCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result []map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(result))
	}
}

func floatPtr(f float64) *float64 { return &f }

func TestQuizzesGradeSync_PutsGradebook(t *testing.T) {
	student := "stu-1"
	var gotGrades map[string]map[string]string
	srv := newQuizzesExtendServer(t, quizzesExtendServerConfig{
		quizGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(sampleQuizJSON(nil))
		},
		attemptsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(quizAttemptsListBody{
				Attempts: []quizAttemptSummary{{
					ID:            "att-1",
					StudentUserID: &student,
					AttemptNumber: 1,
					PointsEarned:  floatPtr(9),
				}},
				RetakePolicy: "latest",
			})
		},
		gradebookPutHandler: func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Grades map[string]map[string]string `json:"grades"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotGrades = body.Grades
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesExtendFlags()
	quizzesGradeSyncFlags.course = "CS101"

	var out bytes.Buffer
	quizzesGradeSyncCmd.SetOut(&out)
	if err := quizzesGradeSyncCmd.RunE(quizzesGradeSyncCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotGrades[student]["quiz-1"] != "9" {
		t.Errorf("grades = %#v", gotGrades)
	}
	if !strings.Contains(out.String(), "Synced 1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestPolicyPointsFromAttempts_Highest(t *testing.T) {
	attempts := []quizAttemptSummary{
		{PointsEarned: floatPtr(7), AttemptNumber: 1},
		{PointsEarned: floatPtr(9), AttemptNumber: 2},
	}
	pts, ok := policyPointsFromAttempts(attempts, "highest")
	if !ok || pts != 9 {
		t.Errorf("pts=%v ok=%v", pts, ok)
	}
}

func TestQuizzesPublish_PatchesStructure(t *testing.T) {
	published := false
	srv := newQuizzesExtendServer(t, quizzesExtendServerConfig{
		structurePatch: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			published, _ = body["published"].(bool)
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuizzesExtendFlags()
	quizzesPublishFlags.course = "CS101"

	quizzesPublishCmd.SetOut(&bytes.Buffer{})
	if err := quizzesPublishCmd.RunE(quizzesPublishCmd, []string{"quiz-1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !published {
		t.Error("expected published=true")
	}
}