package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseFormat(t *testing.T) {
	if ParseFormat("", true) != FormatJSON {
		t.Fatal("json alias")
	}
	if ParseFormat("csv", false) != FormatCSV {
		t.Fatal("csv")
	}
}

func TestRequireYes(t *testing.T) {
	if err := RequireYes(false, ""); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
	if err := RequireYes(true, ""); err != nil {
		t.Fatalf("confirmed: %v", err)
	}
}

func TestParseRFC3339InTZ(t *testing.T) {
	tm, err := ParseRFC3339InTZ("2026-07-01T10:00:00Z", "")
	if err != nil || tm.UTC().Hour() != 10 {
		t.Fatalf("rfc3339: %v %v", tm, err)
	}
	tm2, err := ParseRFC3339InTZ("2026-07-01 10:00", "America/New_York")
	if err != nil {
		t.Fatalf("local parse: %v", err)
	}
	if tm2.IsZero() {
		t.Fatal("zero time")
	}
}

func TestWaitForJob_Success(t *testing.T) {
	n := 0
	st, code, err := WaitForJob(func(_ string) (JobStatus, error) {
		n++
		if n < 2 {
			return JobStatus{ID: "j1", Status: "running"}, nil
		}
		return JobStatus{ID: "j1", Status: "completed"}, nil
	}, "j1", 5*time.Second, 10*time.Millisecond, nil)
	if err != nil || code != 0 || st.Status != "completed" {
		t.Fatalf("st=%+v code=%d err=%v", st, code, err)
	}
}

func TestWaitForJob_Timeout(t *testing.T) {
	_, code, err := WaitForJob(func(_ string) (JobStatus, error) {
		return JobStatus{Status: "running"}, nil
	}, "j1", 50*time.Millisecond, 10*time.Millisecond, nil)
	if err == nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestReadJSONFile_YAML(t *testing.T) {
	m, err := ParseObject([]byte("title: Hello\n"), ".yaml")
	if err != nil || m["title"] != "Hello" {
		t.Fatalf("m=%v err=%v", m, err)
	}
}

func TestWriteRows_CSV(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Format: FormatCSV, Stdout: &buf}
	if err := opts.WriteRows([]string{"a", "b"}, [][]string{{"1", "2"}}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "1,2") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestRedactSecrets(t *testing.T) {
	m := map[string]any{"token": "secret", "label": "ok"}
	RedactSecrets(m)
	if m["token"] != "[redacted]" || m["label"] != "ok" {
		t.Fatalf("m=%v", m)
	}
}

func TestCollectTutorSSE(t *testing.T) {
	body := strings.NewReader("data: {\"type\":\"content\",\"text\":\"Hi\"}\n\ndata: {\"type\":\"done\",\"conversationId\":\"c1\"}\n\n")
	res, err := CollectTutorSSE(body, nil, true)
	if err != nil || res.Text != "Hi" || res.ConversationID != "c1" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestExtractPage(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"messages": []map[string]any{{"id": "1"}},
		"hasMore":  false,
	})
	page, err := ExtractPage(raw, "messages")
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func TestBulkSummary_ExitCode(t *testing.T) {
	s := BulkSummary{Failed: 1}
	if s.ExitCode(false) != 2 {
		t.Fatal("expected 2")
	}
	if s.ExitCode(true) != 0 {
		t.Fatal("expected 0 with continue")
	}
}