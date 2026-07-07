package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type attendanceBehaviorServerConfig struct {
	sessionsListHandler   http.HandlerFunc
	sessionsPostHandler   http.HandlerFunc
	sessionGetHandler     http.HandlerFunc
	sessionRecordsHandler http.HandlerFunc
	enrollmentsHandler    http.HandlerFunc
	usersHandler          http.HandlerFunc
	behaviorHandler       http.HandlerFunc
	categoriesHandler     http.HandlerFunc
	pbisAwardsHandler     http.HandlerFunc
	courseGetHandler      http.HandlerFunc
	seatTimeReportHandler http.HandlerFunc
	hallPassListHandler   http.HandlerFunc
	hallPassPostHandler   http.HandlerFunc
	hallPassPatchHandler  http.HandlerFunc
	sectionsHandler       http.HandlerFunc
}

func newAttendanceBehaviorServer(t *testing.T, cfg attendanceBehaviorServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/attendance/sessions") && !strings.Contains(path, "/attendance/sessions/"):
			if cfg.sessionsListHandler != nil {
				cfg.sessionsListHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/attendance/sessions"):
			if cfg.sessionsPostHandler != nil {
				cfg.sessionsPostHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/attendance/sessions/"):
			if cfg.sessionGetHandler != nil {
				cfg.sessionGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/records"):
			if cfg.sessionRecordsHandler != nil {
				cfg.sessionRecordsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/enrollments"):
			if cfg.enrollmentsHandler != nil {
				cfg.enrollmentsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/users/"):
			if cfg.usersHandler != nil {
				cfg.usersHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/students/") && strings.HasSuffix(path, "/behavior"):
			if cfg.behaviorHandler != nil {
				cfg.behaviorHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/behavior/categories"):
			if cfg.categoriesHandler != nil {
				cfg.categoriesHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && path == "/api/v1/pbis/awards":
			if cfg.pbisAwardsHandler != nil {
				cfg.pbisAwardsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/courses/") && !strings.Contains(path, "/attendance/") && !strings.Contains(path, "/enrollments") && !strings.Contains(path, "/sections") && !strings.HasSuffix(path, "/seat-time-report"):
			if cfg.courseGetHandler != nil {
				cfg.courseGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/seat-time-report"):
			if cfg.seatTimeReportHandler != nil {
				cfg.seatTimeReportHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/hall-passes/active"):
			if cfg.hallPassListHandler != nil {
				cfg.hallPassListHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/hall-passes"):
			if cfg.hallPassPostHandler != nil {
				cfg.hallPassPostHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasPrefix(path, "/api/v1/hall-passes/"):
			if cfg.hallPassPatchHandler != nil {
				cfg.hallPassPatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/sections"):
			if cfg.sectionsHandler != nil {
				cfg.sectionsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func resetAttendanceBehaviorFlags() {
	attendanceListFlags.date = ""
	attendanceRecordFlags.date = ""
	attendanceRecordFlags.user = ""
	attendanceRecordFlags.status = "present"
	attendanceRecordFlags.period = ""
	attendanceRecordFlags.section = ""
	attendanceImportFlags.file = ""
	attendanceImportFlags.section = ""
	attendanceImportFlags.chunkSize = defaultAttendanceImportChunk
	attendanceExportFlags.from = ""
	attendanceExportFlags.to = ""
	attendanceExportFlags.out = ""
	attendanceExportFlags.format = "csv"
	attendanceExportFlags.yes = false
	attendanceSummaryFlags.from = ""
	attendanceSummaryFlags.to = ""
	behaviorExportFlags.out = ""
	behaviorExportFlags.format = "csv"
	behaviorExportFlags.yes = false
	behaviorAwardFlags.user = ""
	behaviorAwardFlags.points = 1
	behaviorAwardFlags.category = ""
	behaviorAwardFlags.note = ""
	seatTimeReportFlags.user = ""
	hallPassListFlags.section = ""
	hallPassIssueFlags.section = ""
	hallPassIssueFlags.destination = "bathroom"
	hallPassIssueFlags.estimatedMins = 5
	hallPassIssueFlags.approve = false
	hallPassReturnFlags.pass = ""
}

func TestNormalizeAttendanceStatus(t *testing.T) {
	tests := []struct {
		in    string
		want  string
		isErr bool
	}{
		{"present", "present", false},
		{"P", "present", false},
		{"absent", "absent", false},
		{"tardy", "tardy", false},
		{"late", "tardy", false},
		{"excused", "excused", false},
		{"bogus", "", true},
	}
	for _, tc := range tests {
		got, err := normalizeAttendanceStatus(tc.in)
		if tc.isErr {
			if err == nil {
				t.Fatalf("normalizeAttendanceStatus(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("normalizeAttendanceStatus(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestAttendanceIdempotencyKey(t *testing.T) {
	k1 := attendanceIdempotencyKey("alice@uni.edu", "2026-01-15", "")
	k2 := attendanceIdempotencyKey("ALICE@uni.edu", "2026-01-15", "")
	if k1 != k2 {
		t.Fatalf("keys differ for same student/date: %q vs %q", k1, k2)
	}
	k3 := attendanceIdempotencyKey("alice@uni.edu", "2026-01-15", "1")
	if k1 == k3 {
		t.Fatalf("period should change idempotency key")
	}
}

func TestParseAttendanceCSV_Valid(t *testing.T) {
	raw := []byte(`student,date,period,status
alice@uni.edu,2026-01-15,1,present
bob@uni.edu,2026-01-15,,absent
`)
	rows, err := parseAttendanceCSV(raw)
	if err != nil {
		t.Fatalf("parseAttendanceCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].Status != "present" || rows[1].Status != "absent" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestParseAttendanceCSV_RejectsMissingDate(t *testing.T) {
	_, err := parseAttendanceCSV([]byte("student,status\nalice@uni.edu,present\n"))
	if err == nil || !strings.Contains(err.Error(), "date column") {
		t.Fatalf("err = %v", err)
	}
}

func TestConfirmAttendanceExport_Gated(t *testing.T) {
	if err := confirmAttendanceExport(false); err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("err = %v", err)
	}
	if err := confirmAttendanceExport(true); err != nil {
		t.Fatalf("confirmAttendanceExport(true): %v", err)
	}
}

func TestBuildAttendanceSummary(t *testing.T) {
	rows := []attendanceExportRow{
		{StudentID: "u1", StudentName: "Alice", Status: "present"},
		{StudentID: "u1", StudentName: "Alice", Status: "tardy"},
		{StudentID: "u2", StudentName: "Bob", Status: "absent"},
	}
	summary := buildAttendanceSummary(rows)
	if len(summary) != 2 {
		t.Fatalf("len(summary) = %d, want 2", len(summary))
	}
	var alice attendanceSummaryRow
	for _, row := range summary {
		if row.StudentID == "u1" {
			alice = row
		}
	}
	if alice.Present != 1 || alice.Tardy != 1 || alice.Total != 2 {
		t.Fatalf("alice summary = %+v", alice)
	}
}

func TestAttendanceExport_Integration(t *testing.T) {
	resetAttendanceBehaviorFlags()
	attendanceExportFlags.from = "2026-01-01"
	attendanceExportFlags.to = "2026-01-31"
	attendanceExportFlags.yes = true

	srv := newAttendanceBehaviorServer(t, attendanceBehaviorServerConfig{
		sessionsListHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(attendanceSessionsBody{
				Sessions: []attendanceSession{{
					ID: "sess-1", Title: "Attendance — 2026-01-15", SessionDate: "2026-01-15", Status: "open",
				}},
			})
		},
		sessionGetHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(attendanceSessionDetail{
				attendanceSession: attendanceSession{ID: "sess-1", SessionDate: "2026-01-15"},
				Records: []attendanceRecord{{
					StudentUserID: "stu-1", DisplayName: "Alice", Status: "present",
				}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "token")
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var buf bytes.Buffer
	attendanceExportCmd.SetOut(&buf)
	if err := attendanceExportCmd.RunE(attendanceExportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("attendance export: %v", err)
	}

	var out []attendanceExportRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("decode export: %v body=%s", err, buf.String())
	}
	if len(out) != 1 || out[0].Status != "present" {
		t.Fatalf("export rows = %+v", out)
	}
}

func TestAttendanceImport_Integration(t *testing.T) {
	resetAttendanceBehaviorFlags()

	tmp, err := os.CreateTemp("", "attendance-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString("email,date,status\nalice@uni.edu,2026-01-15,present\n"); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()
	attendanceImportFlags.file = tmp.Name()

	created := false
	srv := newAttendanceBehaviorServer(t, attendanceBehaviorServerConfig{
		sessionsListHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(attendanceSessionsBody{Sessions: nil})
		},
		sessionsPostHandler: func(w http.ResponseWriter, r *http.Request) {
			created = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(attendanceSession{ID: "sess-new", SessionDate: "2026-01-15"})
		},
		usersHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(userPublic{ID: "stu-1", Email: "alice@uni.edu"})
		},
		sessionRecordsHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"saved": 1})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "token")
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var buf bytes.Buffer
	attendanceImportCmd.SetOut(&buf)
	if err := attendanceImportCmd.RunE(attendanceImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("attendance import: %v", err)
	}
	if !created {
		t.Fatal("expected session creation")
	}
	var summary attendanceImportSummary
	if err := json.Unmarshal(buf.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.RecordsSaved != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestSeatTimeReport_Integration(t *testing.T) {
	resetAttendanceBehaviorFlags()

	srv := newAttendanceBehaviorServer(t, attendanceBehaviorServerConfig{
		seatTimeReportHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(seatTimeReportBody{
				Students: []seatTimeStudentRow{{
					UserID: "u1", DisplayName: "Alice", TotalMinutes: 90, ContactHours: 1.5,
				}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "token")
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var buf bytes.Buffer
	seatTimeReportCmd.SetOut(&buf)
	if err := seatTimeReportCmd.RunE(seatTimeReportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("seat-time report: %v", err)
	}
	var out []seatTimeStudentRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].TotalMinutes != 90 {
		t.Fatalf("out = %+v", out)
	}
}

func TestBehaviorAward_Integration(t *testing.T) {
	resetAttendanceBehaviorFlags()
	behaviorAwardFlags.user = "alice@uni.edu"
	behaviorAwardFlags.points = 5

	srv := newAttendanceBehaviorServer(t, attendanceBehaviorServerConfig{
		courseGetHandler: func(w http.ResponseWriter, r *http.Request) {
			org := "org-1"
			_ = json.NewEncoder(w).Encode(courseDetail{coursePublic: coursePublic{OrgID: &org}})
		},
		usersHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(userPublic{ID: "stu-1", Email: "alice@uni.edu"})
		},
		categoriesHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(behaviorCategoriesBody{
				Categories: []behaviorCategory{{ID: "cat-1", Name: "Respect", Type: "positive", Active: true}},
			})
		},
		pbisAwardsHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"saved": 1})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "token")
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var buf bytes.Buffer
	behaviorAwardCmd.SetOut(&buf)
	if err := behaviorAwardCmd.RunE(behaviorAwardCmd, []string{"CS101"}); err != nil {
		t.Fatalf("behavior award: %v", err)
	}
}

func TestAttendanceBehaviorCommandsRegistered(t *testing.T) {
	for _, sub := range attendanceCmd.Commands() {
		if sub.Name() == "" {
			t.Fatal("empty attendance subcommand")
		}
	}
	wantAttendance := map[string]bool{"list": false, "record": false, "import": false, "export": false, "summary": false}
	for _, sub := range attendanceCmd.Commands() {
		wantAttendance[sub.Name()] = true
	}
	for name, ok := range wantAttendance {
		if !ok {
			t.Fatalf("missing attendance subcommand %q", name)
		}
	}
}