package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetPeerReviewEvaluationsSurveysFlags() {
	peerReviewStatusFlags.course = ""
	peerReviewAllocateFlags.course = ""
	peerReviewAllocateFlags.per = 0
	peerReviewListFlags.course = ""
	evaluationTemplatesCreateFlags.name = ""
	evaluationTemplatesCreateFlags.file = ""
	evaluationsLaunchFlags.template = ""
	evaluationsLaunchFlags.opens = ""
	evaluationsLaunchFlags.closes = ""
	evaluationsListFlags.closedOnly = false
	evaluationsGetFlags.results = false
	evaluationsResultsFlags.format = ""
	evaluationsResultsFlags.out = ""
	surveysCreateFlags.title = ""
	surveysCreateFlags.module = ""
	surveysCreateFlags.file = ""
	surveysResultsFlags.format = ""
	surveysResultsFlags.out = ""
}

func newPeerReviewEvaluationsSurveysServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.Contains(path, "/peer-review/summary"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"config":{"reviewsPerReviewer":2,"anonymity":"double_blind","gradeMode":"none","blendWeight":0,"aggregation":"mean","excludeSameGroup":true},
				"totalAllocations":4,
				"completedReviews":2,
				"incompleteReviewers":["u-2"],
				"outlierReviewers":[],
				"submissions":[{"submissionId":"sub-1","studentUserId":"stu-1","peerAggregate":4.5,"reviewCount":2}]
			}`))
		case r.Method == http.MethodPut && strings.Contains(path, "/peer-review"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"reviewsPerReviewer":2}`))
		case r.Method == http.MethodPost && strings.Contains(path, "/peer-review/allocate"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"allocationsCreated":4}`))
		case r.Method == http.MethodGet && path == "/api/v1/admin/evaluation-templates":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"templates":[{"id":"tmpl-1","name":"End of term","updatedAt":"2026-05-01T00:00:00Z"}]}`))
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/admin/evaluation-templates/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"tmpl-1","name":"End of term","questions":[{"type":"rating","text":"Overall"}]}`))
		case r.Method == http.MethodPost && path == "/api/v1/admin/evaluation-templates":
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"tmpl-new","name":"New template"}`))
		case r.Method == http.MethodPost && strings.Contains(path, "/evaluation-windows"):
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"win-1","courseId":"course-1","templateId":"tmpl-1"}`))
		case r.Method == http.MethodGet && path == "/api/v1/admin/evaluations/report":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"rows":[{"courseCode":"CS101","windowId":"win-1","opensAt":"2026-05-01T00:00:00Z","closesAt":"2026-05-15T00:00:00Z","responseCount":12,"enrolledCount":20,"completionPct":60}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/evaluations/status"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"windowOpen":true,"windowId":"win-1","hasSubmitted":false,"opensAt":"2026-05-01T00:00:00Z","closesAt":"2026-05-15T00:00:00Z"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/evaluations/results"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"windowId":"win-1","responseCount":12,"enrolledCount":20,"completionPct":60,"meetsThreshold":true,"questions":[{"index":0,"type":"rating","text":"Overall","average":4.2}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/surveys") && !strings.Contains(path, "/results"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"survey-1","title":"Midterm pulse","anonymityMode":"anonymous","opensAt":"2026-03-01T00:00:00Z","closesAt":"2026-03-08T00:00:00Z"}]`))
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/surveys/") && strings.HasSuffix(path, "/results"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"responseCount":8,"questions":[{"questionId":"q1","subtype":"likert","responseCount":8,"mean":4.1,"distribution":{"4":3,"5":5}}]}`))
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/surveys/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"survey-1","title":"Midterm pulse","anonymityMode":"anonymous","questions":[{"id":"q1","subtype":"likert","stem":"How is the course?"}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/surveys"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"survey-new","title":"CLI survey"}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestPeerReviewPerReviewer(t *testing.T) {
	if err := peerReviewPerReviewer(2); err != nil {
		t.Fatalf("peerReviewPerReviewer(2): %v", err)
	}
	if err := peerReviewPerReviewer(0); err == nil {
		t.Fatal("expected error for per=0")
	}
}

func TestStripRespondentIdentity(t *testing.T) {
	data := map[string]any{
		"userId": "u-1",
		"responses": []any{
			map[string]any{"email": "student@school.edu", "answer": "yes"},
		},
	}
	stripRespondentIdentity(data)
	if _, ok := data["userId"]; ok {
		t.Fatal("top-level userId must be removed")
	}
	rows := data["responses"].([]any)
	if m, ok := rows[0].(map[string]any); ok {
		if _, ok := m["email"]; ok {
			t.Fatal("nested email must be removed")
		}
	}
}

func TestPrepareSurveyResultsExport_AnonymousMode(t *testing.T) {
	survey := map[string]any{
		"id":            "survey-1",
		"title":         "Pulse",
		"anonymityMode": "anonymous",
	}
	results := map[string]any{
		"responseCount": 3,
		"questions":     []any{},
	}
	out, err := prepareSurveyResultsExport(survey, results)
	if err != nil {
		t.Fatalf("prepareSurveyResultsExport: %v", err)
	}
	if out["anonymityMode"] != "anonymous" {
		t.Fatalf("expected anonymous mode, got %#v", out["anonymityMode"])
	}
	for _, key := range respondentIdentityKeys {
		if _, ok := out[key]; ok {
			t.Fatalf("export doc must not contain %s", key)
		}
	}
}

func TestSurveyResultsToCSV(t *testing.T) {
	data, err := surveyResultsToCSV(map[string]any{
		"surveyId":      "survey-1",
		"title":         "Pulse",
		"anonymityMode": "anonymous",
		"questions": []any{
			map[string]any{
				"questionId":     "q1",
				"subtype":        "likert",
				"responseCount":  float64(5),
				"mean":           4.2,
				"distribution":   map[string]any{"5": 3},
			},
		},
	})
	if err != nil {
		t.Fatalf("surveyResultsToCSV: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "survey-1") || !strings.Contains(text, "anonymous") {
		t.Fatalf("csv missing expected fields: %s", text)
	}
}

func TestPeerReviewAllocateCommand(t *testing.T) {
	resetPeerReviewEvaluationsSurveysFlags()
	srv := newPeerReviewEvaluationsSurveysServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")

	peerReviewAllocateFlags.course = "CS101"
	peerReviewAllocateFlags.per = 2
	var out bytes.Buffer
	peerReviewAllocateCmd.SetOut(&out)
	if err := peerReviewAllocateCmd.RunE(peerReviewAllocateCmd, []string{"item-1"}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if !strings.Contains(out.String(), "Created 4") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestEvaluationsLaunchCommand(t *testing.T) {
	resetPeerReviewEvaluationsSurveysFlags()
	srv := newPeerReviewEvaluationsSurveysServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")

	evaluationsLaunchFlags.template = "tmpl-1"
	evaluationsLaunchFlags.opens = "2026-05-01T00:00:00Z"
	evaluationsLaunchFlags.closes = "2026-05-15T00:00:00Z"
	var out bytes.Buffer
	evaluationsLaunchCmd.SetOut(&out)
	if err := evaluationsLaunchCmd.RunE(evaluationsLaunchCmd, []string{"CS101"}); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if !strings.Contains(out.String(), "win-1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestSurveysResultsExportAnonymousJSON(t *testing.T) {
	resetPeerReviewEvaluationsSurveysFlags()
	srv := newPeerReviewEvaluationsSurveysServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	surveysResultsFlags.format = "json"
	var out bytes.Buffer
	surveysResultsCmd.SetOut(&out)
	if err := surveysResultsCmd.RunE(surveysResultsCmd, []string{"export", "survey-1"}); err != nil {
		t.Fatalf("export: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode export json: %v", err)
	}
	if payload["anonymityMode"] != "anonymous" {
		t.Fatalf("expected anonymous mode, got %#v", payload["anonymityMode"])
	}
	for _, key := range respondentIdentityKeys {
		if _, ok := payload[key]; ok {
			t.Fatalf("export must not include identity key %s", key)
		}
	}
}