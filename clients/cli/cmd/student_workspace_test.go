package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotebookFromTextFile(t *testing.T) {
	raw, err := notebookFromTextFile("# Notes\nHello")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["formatVersion"] != float64(2) {
		t.Fatalf("body=%v", body)
	}
}

func TestParseTodoDueColumn(t *testing.T) {
	col, err := parseTodoDueColumn("tomorrow", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if col == "" {
		t.Fatal("empty col")
	}
	if c, err := parseTodoDueColumn("done", ""); err != nil || c != "done" {
		t.Fatalf("c=%s err=%v", c, err)
	}
}

func TestGamificationStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/gamification" {
			_ = json.NewEncoder(w).Encode(map[string]any{"points": 42, "streak": 3})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	var out strings.Builder
	gamificationStatusCmd.SetOut(&out)
	if err := gamificationStatusCmd.RunE(gamificationStatusCmd, nil); err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out.String(), "points") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestTodoAdd_Success(t *testing.T) {
	var saved map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me/student-todo-board":
			_ = json.NewEncoder(w).Encode(map[string]any{"placements": []any{}})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/me/student-todo-board":
			_ = json.NewDecoder(r.Body).Decode(&saved)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	todoAddFlags.col = "mon"
	defer func() { todoAddFlags.col = "" }()
	if err := todoAddCmd.RunE(todoAddCmd, []string{"read chapter 3"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if saved["columns"] == nil {
		t.Fatalf("saved=%v", saved)
	}
}