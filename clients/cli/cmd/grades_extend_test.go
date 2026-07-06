package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type gradesExtendServerConfig struct {
	gridHandler            http.HandlerFunc
	putHandler             http.HandlerFunc
	schemeGetHandler       http.HandlerFunc
	schemePutHandler       http.HandlerFunc
	gradingHandler         http.HandlerFunc
	backlogHandler         http.HandlerFunc
	finalPreviewHandler    http.HandlerFunc
	finalSubmitHandler     http.HandlerFunc
	curvePreviewHandler    http.HandlerFunc
	curveApplyHandler      http.HandlerFunc
	gradeHistoryHandler    http.HandlerFunc
}

func newGradesExtendServer(t *testing.T, cfg gradesExtendServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/gradebook/grid"):
			if cfg.gridHandler != nil {
				cfg.gridHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/gradebook/grades"):
			if cfg.putHandler != nil {
				cfg.putHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grading-scheme"):
			if cfg.schemeGetHandler != nil {
				cfg.schemeGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/grading-scheme"):
			if cfg.schemePutHandler != nil {
				cfg.schemePutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grading"):
			if cfg.gradingHandler != nil {
				cfg.gradingHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/grading-backlog"):
			if cfg.backlogHandler != nil {
				cfg.backlogHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/final-grades/preview"):
			if cfg.finalPreviewHandler != nil {
				cfg.finalPreviewHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/final-grades/submit"):
			if cfg.finalSubmitHandler != nil {
				cfg.finalSubmitHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.Contains(path, "/curve/preview"):
			if cfg.curvePreviewHandler != nil {
				cfg.curvePreviewHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.Contains(path, "/curve") && !strings.Contains(path, "/preview"):
			if cfg.curveApplyHandler != nil {
				cfg.curveApplyHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/grades/") && strings.HasSuffix(path, "/history"):
			if cfg.gradeHistoryHandler != nil {
				cfg.gradeHistoryHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func resetGradesExtendFlags() {
	gradebookGetFlags.format = "csv"
	gradebookGetFlags.yes = false
	gradebookImportFlags.file = ""
	finalGradesSetFlags.enrollment = ""
	finalGradesSetFlags.grade = ""
	finalGradesSetFlags.reason = ""
	finalGradesSetFlags.method = "csv"
	finalGradesSetFlags.yes = false
	finalGradesSubmitFlags.file = ""
	finalGradesSubmitFlags.method = "csv"
	finalGradesSubmitFlags.yes = false
	gradesSchemeSetFlags.file = ""
	gradesSchemeSetFlags.name = ""
	gradesSchemeSetFlags.schemeType = ""
	gradesSchemeSetFlags.scaleJSON = ""
	gradesCurveFlags.course = ""
	gradesCurveFlags.assignment = ""
	gradesCurveFlags.method = ""
	gradesCurveFlags.bonus = 0
	gradesCurveFlags.targetMean = 0
	gradesCurveFlags.targetMax = 0
	gradesCurveFlags.minimum = 0
	gradesCurveFlags.allowAbove = false
	gradesCurveFlags.dryRun = false
	gradesWhatIfFlags.course = ""
	gradesWhatIfFlags.user = ""
	gradesWhatIfFlags.override = nil
	gradesWhatIfFlags.file = ""
	gradesHistoryFlags.course = ""
	gradesHistoryFlags.assignment = ""
	gradesHistoryFlags.user = ""
}

func sampleGradingSettings() gradingSettingsResponse {
	return gradingSettingsResponse{
		GradingScale: "percent",
		AssignmentGroups: []assignmentGroupWeight{{
			ID: "group-1", WeightPercent: 100,
		}},
	}
}

func TestGradebookGet_MatrixCSV(t *testing.T) {
	grid := sampleGrid()
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gridHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(grid)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradebookGetFlags.yes = true

	var out bytes.Buffer
	gradebookGetCmd.SetOut(&out)
	if err := gradebookGetCmd.RunE(gradebookGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	records, err := csv.NewReader(&out).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected header + rows, got %d", len(records))
	}
	if records[0][0] != "student_id" || records[0][2] != "Homework 1" {
		t.Errorf("header = %v", records[0])
	}
	if records[1][0] != "user-001" || records[1][2] != "90" {
		t.Errorf("row = %v", records[1])
	}
}

func TestGradebookGet_RequiresYes(t *testing.T) {
	setCfg("http://unused", "test-key")
	resetGradesExtendFlags()
	err := gradebookGetCmd.RunE(gradebookGetCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("expected FERPA refusal, got %v", err)
	}
}

func TestGradebookImport_Success(t *testing.T) {
	grid := sampleGrid()
	var gotBody map[string]any
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gridHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(grid)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	csvPath := writeTempCSV(t, strings.Join([]string{
		"student_id,item_id,score",
		"user-001,item-aaa,92",
		"user-002,item-bbb,40",
	}, "\n"))

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradebookImportFlags.file = csvPath

	var out bytes.Buffer
	gradebookImportCmd.SetOut(&out)
	if err := gradebookImportCmd.RunE(gradebookImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "2 posted") {
		t.Errorf("output = %q; want posted count", out.String())
	}
	grades, _ := gotBody["grades"].(map[string]any)
	if grades == nil || grades["user-001"] == nil {
		t.Fatalf("PUT body missing grades: %v", gotBody)
	}
}

func TestGradebookImport_RoundTripByTitle(t *testing.T) {
	grid := sampleGrid()
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gridHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(grid)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	csvPath := writeTempCSV(t, strings.Join([]string{
		"student_id,assignment_title,score",
		"user-001,Homework 1,88",
	}, "\n"))

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradebookImportFlags.file = csvPath
	gradebookImportCmd.SetOut(&bytes.Buffer{})
	if err := gradebookImportCmd.RunE(gradebookImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}

func TestGradebookImport_PartialFailureSummary(t *testing.T) {
	grid := sampleGrid()
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gridHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(grid)
		},
	})
	defer srv.Close()

	csvPath := writeTempCSV(t, strings.Join([]string{
		"student_id,item_id,score",
		"unknown,item-aaa,90",
		"user-001,bad-item,90",
	}, "\n"))

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradebookImportFlags.file = csvPath
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	gradebookImportCmd.SetOut(&out)
	if err := gradebookImportCmd.RunE(gradebookImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var summary gradeImportSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if summary.Failed != 2 {
		t.Errorf("failed = %d, want 2", summary.Failed)
	}
}

func TestFinalGradesList_Success(t *testing.T) {
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		finalPreviewHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(finalGradesPreviewResponse{
				Grades: []struct {
					EnrollmentID     string  `json:"enrollmentId"`
					UserID           string  `json:"userId"`
					DisplayName      string  `json:"displayName"`
					State            string  `json:"state"`
					ComputedGrade    string  `json:"computedGrade"`
					FinalGrade       string  `json:"finalGrade"`
					OverrideReason   string  `json:"overrideReason,omitempty"`
					AlreadySubmitted bool    `json:"alreadySubmitted"`
					SubmittedAt      *string `json:"submittedAt,omitempty"`
				}{{
					DisplayName: "Alice", State: "active", ComputedGrade: "A", FinalGrade: "A", AlreadySubmitted: true,
				}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	var out bytes.Buffer
	finalGradesListCmd.SetOut(&out)
	if err := finalGradesListCmd.RunE(finalGradesListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Alice") || !strings.Contains(out.String(), "yes") {
		t.Errorf("output = %q", out.String())
	}
}

func TestFinalGradesSubmit_RequiresYes(t *testing.T) {
	setCfg("http://unused", "test-key")
	resetGradesExtendFlags()
	err := finalGradesSubmitCmd.RunE(finalGradesSubmitCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "irreversible") {
		t.Fatalf("expected confirmation error, got %v", err)
	}
}

func TestFinalGradesSubmit_Success(t *testing.T) {
	var gotBody map[string]any
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		finalSubmitHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(finalGradesSubmitResponse{Count: 2})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	finalGradesSubmitFlags.yes = true

	var out bytes.Buffer
	finalGradesSubmitCmd.SetOut(&out)
	if err := finalGradesSubmitCmd.RunE(finalGradesSubmitCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Submitted 2") {
		t.Errorf("output = %q", out.String())
	}
	if gotBody["method"] != "csv" {
		t.Errorf("method = %v", gotBody["method"])
	}
}

func TestGradesSchemeGet_Success(t *testing.T) {
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		schemeGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(gradingSchemeEnvelope{
				Scheme: &struct {
					ID        string          `json:"id"`
					Name      string          `json:"name"`
					Type      string          `json:"type"`
					ScaleJSON json.RawMessage `json:"scaleJson"`
				}{ID: "scheme-1", Name: "Default", Type: "letter", ScaleJSON: json.RawMessage(`{"A":90}`)},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	var out bytes.Buffer
	gradesSchemeGetCmd.SetOut(&out)
	if err := gradesSchemeGetCmd.RunE(gradesSchemeGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "letter") {
		t.Errorf("output = %q", out.String())
	}
}

func TestGradingBacklogList_Success(t *testing.T) {
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		backlogHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(gradingBacklogResponse{
				Items: []struct {
					ItemID          string `json:"itemId"`
					ItemType        string `json:"itemType"`
					AssignmentTitle string `json:"assignmentTitle"`
					UngradedCount   int64  `json:"ungradedCount"`
				}{{
					ItemType: "assignment", AssignmentTitle: "Essay 1", UngradedCount: 3,
				}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	var out bytes.Buffer
	gradingBacklogListCmd.SetOut(&out)
	if err := gradingBacklogListCmd.RunE(gradingBacklogListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Essay 1") || !strings.Contains(out.String(), "3") {
		t.Errorf("output = %q", out.String())
	}
}

func TestGradesCurve_DryRun(t *testing.T) {
	var gotMethod string
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		curvePreviewHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotMethod, _ = body["method"].(string)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(curvePreviewResponse{
				MaxPoints: 100,
				Preview: struct {
					EligibleCount int `json:"eligibleCount"`
					MeanBefore    *float64 `json:"meanBefore"`
					MeanAfter     *float64 `json:"meanAfter"`
					Results       []struct {
						StudentID     string  `json:"studentId"`
						RawScore      float64 `json:"rawScore"`
						AdjustedScore float64 `json:"adjustedScore"`
						Delta         float64 `json:"delta"`
						Changed       bool    `json:"changed"`
					} `json:"results"`
				}{
					EligibleCount: 1,
					Results: []struct {
						StudentID     string  `json:"studentId"`
						RawScore      float64 `json:"rawScore"`
						AdjustedScore float64 `json:"adjustedScore"`
						Delta         float64 `json:"delta"`
						Changed       bool    `json:"changed"`
					}{{
						StudentID: "user-001", RawScore: 70, AdjustedScore: 80, Delta: 10, Changed: true,
					}},
				},
			})
		},
		curveApplyHandler: func(w http.ResponseWriter, r *http.Request) {
			t.Error("apply should not be called in dry-run")
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradesCurveFlags.course = "CS101"
	gradesCurveFlags.assignment = "item-aaa"
	gradesCurveFlags.method = "sqrt"
	gradesCurveFlags.dryRun = true

	var out bytes.Buffer
	gradesCurveCmd.SetOut(&out)
	if err := gradesCurveCmd.RunE(gradesCurveCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotMethod != "sqrt_curve" {
		t.Errorf("method = %q, want sqrt_curve", gotMethod)
	}
	if !strings.Contains(out.String(), "[dry-run]") || !strings.Contains(out.String(), "70") {
		t.Errorf("output = %q", out.String())
	}
}

func TestGradesWhatIf_Projection(t *testing.T) {
	grid := sampleGrid()
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gridHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(grid)
		},
		gradingHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleGradingSettings())
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradesWhatIfFlags.course = "CS101"
	gradesWhatIfFlags.user = "user-001"
	gradesWhatIfFlags.override = []string{"item-bbb=50"}

	var out bytes.Buffer
	gradesWhatIfCmd.SetOut(&out)
	if err := gradesWhatIfCmd.RunE(gradesWhatIfCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Actual final") || !strings.Contains(out.String(), "Projected final") {
		t.Errorf("output = %q", out.String())
	}
}

func TestGradesHistory_Success(t *testing.T) {
	srv := newGradesExtendServer(t, gradesExtendServerConfig{
		gradeHistoryHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(assignmentGradeHistoryBody{
				Events: []assignmentGradeHistoryEvent{{
					Action: "graded", ChangedAt: "2026-01-01T00:00:00Z",
				}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetGradesExtendFlags()
	gradesHistoryFlags.course = "CS101"
	gradesHistoryFlags.assignment = "item-aaa"
	gradesHistoryFlags.user = "user-001"

	var out bytes.Buffer
	gradesHistoryCmd.SetOut(&out)
	if err := gradesHistoryCmd.RunE(gradesHistoryCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "graded") {
		t.Errorf("output = %q", out.String())
	}
}

func TestParseGradeImportCSV_RoundTrip(t *testing.T) {
	grid := sampleGrid()
	csvData := strings.Join([]string{
		"student_id,assignment_title,score",
		"user-001,Homework 1,91",
		"user-002,Midterm,44",
	}, "\n")
	grades, summary, err := parseGradeImportCSV(strings.NewReader(csvData), &grid)
	if err != nil {
		t.Fatalf("parseGradeImportCSV: %v", err)
	}
	if summary.Failed != 0 || summary.Posted != 0 {
		t.Errorf("summary = %+v", summary)
	}
	if grades["user-001"]["item-aaa"] != "91" {
		t.Errorf("grades = %v", grades)
	}
}

func TestMergeGradesForWhatIf_HeldItems(t *testing.T) {
	actual := map[string]string{"a": "80", "secret": "99"}
	held := map[string]bool{"secret": true}
	merged := mergeGradesForWhatIf(actual, map[string]string{"a": "95"}, held)
	if merged["secret"] != "" {
		t.Errorf("held item leaked: %v", merged)
	}
	if merged["a"] != "95" {
		t.Errorf("override not applied: %v", merged)
	}
}

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "grades-*.csv")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	_ = f.Close()
	return f.Name()
}