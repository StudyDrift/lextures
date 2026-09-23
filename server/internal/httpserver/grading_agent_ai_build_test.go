package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

func TestWriteGraderAgentAIBuildStream_keepalivesThenResult(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai-build", nil)
	started := make(chan struct{})
	release := make(chan struct{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		writeGraderAgentAIBuildStream(rec, req, 10*time.Millisecond, func(context.Context) (gradingagentsvc.BuilderResult, error) {
			close(started)
			<-release
			return gradingagentsvc.BuilderResult{
				Summary: "Award full credit when the answer matches.",
				Graph:   &gradingagentsvc.WorkflowGraph{},
			}, nil
		})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("build did not start")
	}
	time.Sleep(35 * time.Millisecond)
	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("build stream did not finish")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/x-ndjson") {
		t.Fatalf("content-type %q", ct)
	}
	lines := nonEmptyLines(rec.Body.String())
	if len(lines) < 3 {
		t.Fatalf("expected keepalive frames before the result, got %d lines: %s", len(lines), rec.Body.String())
	}
	var progress int
	for _, line := range lines[:len(lines)-1] {
		var frame graderAgentAIBuildFrame
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			t.Fatalf("progress frame: %v", err)
		}
		if frame.Type != "progress" {
			t.Fatalf("frame type %q before result", frame.Type)
		}
		progress++
	}
	if progress < 2 {
		t.Fatalf("expected at least 2 progress frames, got %d", progress)
	}
	var result graderAgentAIBuildFrame
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &result); err != nil {
		t.Fatal(err)
	}
	if result.Type != "result" || result.Summary == "" || len(result.WorkflowGraph) == 0 {
		t.Fatalf("result frame: %+v", result)
	}
}

func TestWriteGraderAgentAIBuildStream_errorFrame(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai-build", nil)
	writeGraderAgentAIBuildStream(rec, req, time.Hour, func(context.Context) (gradingagentsvc.BuilderResult, error) {
		return gradingagentsvc.BuilderResult{}, gradingagentsvc.ValidationError{Message: "missing score node"}
	})
	lines := nonEmptyLines(rec.Body.String())
	if len(lines) < 2 {
		t.Fatalf("body: %s", rec.Body.String())
	}
	var frame graderAgentAIBuildFrame
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &frame); err != nil {
		t.Fatal(err)
	}
	if frame.Type != "error" || !strings.Contains(frame.Message, "missing score node") {
		t.Fatalf("error frame: %+v", frame)
	}
}

func nonEmptyLines(body string) []string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
