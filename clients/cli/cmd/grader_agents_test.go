package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

func resetGraderAgentsFlags() {
	graderAgentsGetFlags.assignment = ""
	graderAgentsSetFlags.assignment = ""
	graderAgentsSetFlags.file = ""
	graderAgentsDeleteFlags.assignment = ""
	graderAgentsDryRunFlags.assignment = ""
	graderAgentsDryRunFlags.sample = 1
	graderAgentsDryRunFlags.submission = ""
	graderAgentsRunFlags.course = ""
	graderAgentsRunFlags.scope = "ungraded"
	graderAgentsRunFlags.mode = "suggest"
	graderAgentsRunFlags.yes = false
	graderAgentsRunFlags.wait = false
	graderAgentsRunFlags.timeout = 600
	graderAgentsRunFlags.overwrite = false
	graderAgentsResultsFlags.course = ""
	graderAgentsResultsFlags.run = ""
	graderAgentsResultsFlags.accept = false
	graderAgentsResultsFlags.yes = false
	graderTemplatesCreateFlags.name = ""
	graderTemplatesCreateFlags.file = ""
}

func newGraderAgentsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && path == "/api/v1/settings/ai-opt-out":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"aiProcessingOptOut":false,"disclosureUrl":"/ai-disclosure"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grader-agents"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"agents":[{"id":"cfg-1","itemId":"item-1","assignmentTitle":"Essay 1","status":"accepted","autoGradeNew":false,"reviewCount":2,"updatedAt":"2026-06-01T00:00:00Z"}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grader-agent") && !strings.Contains(path, "/runs"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"config":{"id":"cfg-1","status":"accepted","postPolicy":"draft","modelId":"openrouter/auto","workflowGraph":{"nodes":[{"id":"n1","type":"output"}],"edges":[]}}}`))
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/grader-agent"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"config":{"id":"cfg-1","status":"accepted","postPolicy":"draft","workflowGraph":{"nodes":[{"id":"out","type":"output"}],"edges":[]}}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grader-agent-templates"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"templates":[{"id":"tmpl-1","name":"Rubric grader","isBuiltin":false,"updatedAt":"2026-06-01T00:00:00Z"}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/grader-agent/runs"):
			w.WriteHeader(http.StatusAccepted)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"runId":"run-1","totalCount":3,"queuedCount":3,"mode":"suggest","targetSummary":"3 ungraded submissions"}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/grader-agent/runs/run-1"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"done","totalCount":3,"completedCount":3,"failedCount":0,"promptTokens":1200,"completionTokens":300,"costUsd":0.04,"results":[{"id":"res-1","submissionId":"sub-1","status":"held","suggestedPoints":8.5,"confidence":0.82}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grader-agent/runs"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"runs":[{"id":"run-1","status":"done","totalCount":3,"completedCount":3,"failedCount":0,"createdAt":"2026-06-01T00:00:00Z"}]}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && strings.HasSuffix(path, "/submissions"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"submissions":[{"id":"sub-1","submittedBy":"stu-1","submittedAt":"2026-06-01T00:00:00Z"},{"id":"sub-2","submittedBy":"stu-2","submittedAt":"2026-06-02T00:00:00Z"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestValidateGraderAgentConfigPayload(t *testing.T) {
	if err := validateGraderAgentConfigPayload([]byte(`{"prompt":"Grade fairly"}`)); err != nil {
		t.Fatalf("valid prompt config: %v", err)
	}
	if err := validateGraderAgentConfigPayload([]byte(`{"workflowGraph":{"nodes":[],"edges":[]}}`)); err != nil {
		t.Fatalf("valid workflow config: %v", err)
	}
	if err := validateGraderAgentConfigPayload([]byte(`{"status":"accepted"}`)); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNormalizeGraderAgentSetPayload(t *testing.T) {
	raw, err := normalizeGraderAgentSetPayload([]byte(`{"config":{"prompt":"x","workflowGraph":{"nodes":[],"edges":[]}}}`))
	if err != nil {
		t.Fatalf("normalize wrapped config: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := doc["config"]; ok {
		t.Fatal("config wrapper must be removed")
	}
	if doc["prompt"] != "x" {
		t.Fatalf("prompt=%#v", doc["prompt"])
	}
}

func TestParseAIOptOutResponse(t *testing.T) {
	optedOut, err := parseAIOptOutResponse([]byte(`{"aiProcessingOptOut":true}`))
	if err != nil || !optedOut {
		t.Fatalf("optedOut=%v err=%v", optedOut, err)
	}
	optedOut, err = parseAIOptOutResponse([]byte(`{"aiProcessingOptOut":false}`))
	if err != nil || optedOut {
		t.Fatalf("optedOut=%v err=%v", optedOut, err)
	}
}

func TestHTTPToWSURL(t *testing.T) {
	got, err := httpToWSURL("http://localhost:8080", "/api/v1/ws")
	if err != nil || got != "ws://localhost:8080/api/v1/ws" {
		t.Fatalf("http->ws = %q err=%v", got, err)
	}
	got, err = httpToWSURL("https://app.example.com", "/stream")
	if err != nil || got != "wss://app.example.com/stream" {
		t.Fatalf("https->wss = %q err=%v", got, err)
	}
}

func TestLimitSubmissionIDs(t *testing.T) {
	ids := []string{"a", "b", "c"}
	if got := limitSubmissionIDs(ids, 0); len(got) != 3 {
		t.Fatalf("sample 0: len=%d", len(got))
	}
	if got := limitSubmissionIDs(ids, 2); len(got) != 2 || got[1] != "b" {
		t.Fatalf("sample 2: %#v", got)
	}
}

func TestCollectDryRunSampleResult(t *testing.T) {
	events := []dryRunExecutionEvent{
		{Type: "log", Message: "starting"},
		{Type: "complete", Result: &struct {
			SuggestedPoints  float64 `json:"suggestedPoints"`
			Comment          string  `json:"comment"`
			Confidence       float64 `json:"confidence"`
			PromptTokens     int     `json:"promptTokens,omitempty"`
			CompletionTokens int     `json:"completionTokens,omitempty"`
			CostUSD          float64 `json:"costUsd,omitempty"`
		}{SuggestedPoints: 9, Confidence: 0.9, PromptTokens: 100, CompletionTokens: 20, CostUSD: 0.01}},
	}
	out := collectDryRunSampleResult("sub-1", events)
	if out.SuggestedPoints != 9 || out.PromptTokens != 100 || out.CostUSD != 0.01 {
		t.Fatalf("unexpected sample: %#v", out)
	}
}

func TestGraderAgentRunIsTerminal(t *testing.T) {
	if !graderAgentRunIsTerminal("done") || !graderAgentRunIsTerminal("failed") {
		t.Fatal("expected terminal statuses")
	}
	if graderAgentRunIsTerminal("running") {
		t.Fatal("running must not be terminal")
	}
}

func TestEnsureAIGradingAllowedBlocksOptOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"aiProcessingOptOut":true}`))
	}))
	defer srv.Close()
	c := clientFromURL(srv.URL)
	err := ensureAIGradingAllowed(c)
	if err == nil {
		t.Fatal("expected opt-out refusal")
	}
	if !strings.Contains(err.Error(), "AI processing is disabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraderAgentsSetAndGetCommands(t *testing.T) {
	resetGraderAgentsFlags()
	srv := newGraderAgentsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")

	dir := t.TempDir()
	cfgPath := dir + "/agent.json"
	if err := os.WriteFile(cfgPath, []byte(`{"prompt":"Grade with rubric","status":"accepted","workflowGraph":{"nodes":[{"id":"out","type":"output"}],"edges":[]}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	graderAgentsSetFlags.assignment = "item-1"
	graderAgentsSetFlags.file = cfgPath
	var setOut bytes.Buffer
	graderAgentsSetCmd.SetOut(&setOut)
	if err := graderAgentsSetCmd.RunE(graderAgentsSetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if !strings.Contains(setOut.String(), "Saved grader agent") {
		t.Fatalf("set output: %s", setOut.String())
	}

	graderAgentsGetFlags.assignment = "item-1"
	var getOut bytes.Buffer
	graderAgentsGetCmd.SetOut(&getOut)
	if err := graderAgentsGetCmd.RunE(graderAgentsGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(getOut.String(), "accepted") {
		t.Fatalf("get output: %s", getOut.String())
	}
}

func TestGraderAgentsRunCommand(t *testing.T) {
	resetGraderAgentsFlags()
	srv := newGraderAgentsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")

	graderAgentsRunFlags.course = "CS101"
	var out bytes.Buffer
	graderAgentsRunCmd.SetOut(&out)
	if err := graderAgentsRunCmd.RunE(graderAgentsRunCmd, []string{"item-1"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "run-1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestGraderAgentsDryRunWebSocket(t *testing.T) {
	resetGraderAgentsFlags()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/settings/ai-opt-out", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"aiProcessingOptOut":false}`))
	})
	mux.HandleFunc("/api/v1/courses/CS101/assignments/item-1/grader-agent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":{"workflowGraph":{"nodes":[{"id":"out","type":"output"}],"edges":[]}}}`))
	})
	mux.HandleFunc("/api/v1/courses/CS101/assignments/item-1/grader-agent/dry-run/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		ctx := r.Context()
		_, _, _ = conn.Read(ctx)
		_ = writeWSText(ctx, conn, `{"type":"log","level":"info","message":"Starting dry run…"}`)
		_ = writeWSText(ctx, conn, `{"type":"complete","result":{"suggestedPoints":8,"comment":"Good work","confidence":0.85,"promptTokens":50,"completionTokens":10,"costUsd":0.002}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	setCfg(srv.URL, "test-key")

	graderAgentsDryRunFlags.assignment = "item-1"
	graderAgentsDryRunFlags.submission = "sub-1"
	var out bytes.Buffer
	graderAgentsDryRunCmd.SetOut(&out)
	if err := graderAgentsDryRunCmd.RunE(graderAgentsDryRunCmd, []string{"CS101"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "8.00 pts") || !strings.Contains(text, "prompt=50") {
		t.Fatalf("unexpected dry-run output: %s", text)
	}
}

func writeWSText(ctx context.Context, conn *websocket.Conn, payload string) error {
	return conn.Write(ctx, websocket.MessageText, []byte(payload))
}

func clientFromURL(url string) *client.Client {
	return client.New(url, "test-key")
}