package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestFilterXAPIEvents(t *testing.T) {
	events := []xapiEventRow{
		{StatementID: "s1", Verb: "http://adlnet.gov/expapi/verbs/completed", ObjectID: "obj-a"},
		{StatementID: "s2", Verb: "http://adlnet.gov/expapi/verbs/attempted", ObjectID: "obj-b"},
	}
	got := filterXAPIEvents(events, "completed", "", "")
	if len(got) != 1 || got[0].StatementID != "s1" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestStatementIDFromJSON(t *testing.T) {
	id := statementIDFromJSON(json.RawMessage(`{"id":"abc-123","verb":{"id":"completed"}}`))
	if id != "abc-123" {
		t.Fatalf("got %q", id)
	}
}

func TestReadJSONObjectsFromFile_Array(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/events.json"
	if err := os.WriteFile(path, []byte(`[{"eventType":"heartbeat"},{"eventType":"scroll_depth"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	objs, err := readJSONObjectsFromFile(path)
	if err != nil || len(objs) != 2 {
		t.Fatalf("objs=%d err=%v", len(objs), err)
	}
}

func TestXAPIQuery_RequiresYes(t *testing.T) {
	xapiQueryFlags.yes = false
	xapiQueryFlags.course = "CS101"
	defer func() {
		xapiQueryFlags.yes = false
		xapiQueryFlags.course = ""
	}()
	err := xapiQueryCmd.RunE(xapiQueryCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestEngagementEmit_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/analytics/events" {
			_ = json.NewEncoder(w).Encode(map[string]any{"stored": 1})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := dir + "/event.json"
	if err := os.WriteFile(path, []byte(`{"eventType":"heartbeat","value":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	engagementEmitFlags.file = path
	defer func() { engagementEmitFlags.file = "" }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	engagementEmitCmd.SetOut(&out)
	if err := engagementEmitCmd.RunE(engagementEmitCmd, nil); err != nil {
		t.Fatalf("engagement emit: %v", err)
	}
	if !strings.Contains(out.String(), `"stored":1`) {
		t.Fatalf("output = %q", out.String())
	}
}

func TestLRSDeadLetterList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/lrs-dead-letter" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "dl-1", "statementId": "stmt-1", "lastError": "timeout",
			}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	lrsDeadLetterListNestedCmd.SetOut(&out)
	if err := lrsDeadLetterListNestedCmd.RunE(lrsDeadLetterListNestedCmd, nil); err != nil {
		t.Fatalf("lrs dead-letter list: %v", err)
	}
	if !strings.Contains(out.String(), "stmt-1") {
		t.Fatalf("output = %q", out.String())
	}
}