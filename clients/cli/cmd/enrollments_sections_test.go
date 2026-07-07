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

type enrollmentsSectionsServerConfig struct {
	listHandler         http.HandlerFunc
	postHandler         http.HandlerFunc
	deleteHandler       http.HandlerFunc
	statePatchHandler   http.HandlerFunc
	sectionTransfer     http.HandlerFunc
	sectionsListHandler http.HandlerFunc
	selfEnrollHandler   http.HandlerFunc
	userGetHandler      http.HandlerFunc
}

func newEnrollmentsSectionsServer(t *testing.T, cfg enrollmentsSectionsServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/enrollments"):
			if cfg.listHandler != nil {
				cfg.listHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/enrollments"):
			if cfg.postHandler != nil {
				cfg.postHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodDelete && strings.Contains(path, "/enrollments/"):
			if cfg.deleteHandler != nil {
				cfg.deleteHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/state"):
			if cfg.statePatchHandler != nil {
				cfg.statePatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/section"):
			if cfg.sectionTransfer != nil {
				cfg.sectionTransfer(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/sections"):
			if cfg.sectionsListHandler != nil {
				cfg.sectionsListHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/self-enroll"):
			if cfg.selfEnrollHandler != nil {
				cfg.selfEnrollHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/users/"):
			if cfg.userGetHandler != nil {
				cfg.userGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func resetEnrollmentsSectionsFlags() {
	enrollmentsListFlags.role = ""
	enrollmentsListFlags.section = ""
	enrollmentsListFlags.state = ""
	enrollmentsExportFlags.out = ""
	enrollmentsExportFlags.format = "csv"
	enrollmentsExportFlags.yes = false
	enrollmentsImportFlags.file = ""
	enrollmentsImportFlags.role = "student"
	enrollmentsImportFlags.chunkSize = defaultEnrollmentImportChunk
	enrollmentsImportFlags.createMissing = false
	enrollmentsAddFlags.user = ""
	enrollmentsAddFlags.role = "student"
	enrollmentsAddFlags.section = ""
	enrollmentsRemoveFlags.user = ""
	enrollmentsRemoveFlags.role = ""
	enrollmentsSetStateFlags.user = ""
	enrollmentsSetStateFlags.state = ""
	enrollmentsSetStateFlags.reason = ""
	enrollmentsSetStateFlags.role = "student"
	sectionsMoveFlags.user = ""
	sectionsMoveFlags.to = ""
	sectionsMoveFlags.role = "student"
}

func TestParseRosterCSV_Valid(t *testing.T) {
	raw := []byte(`email,role,section
alice@uni.edu,student,SEC-A
bob@uni.edu,ta,
`)
	rows, err := parseRosterCSV(raw, "student")
	if err != nil {
		t.Fatalf("parseRosterCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].Email != "alice@uni.edu" || rows[0].Role != "student" || rows[0].Section != "SEC-A" {
		t.Fatalf("row0 = %+v", rows[0])
	}
}

func TestParseRosterCSV_RejectsMissingEmailColumn(t *testing.T) {
	_, err := parseRosterCSV([]byte("role\nstudent\n"), "student")
	if err == nil || !strings.Contains(err.Error(), "email or sis_id") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseRosterCSV_RejectsSISOnly(t *testing.T) {
	raw := []byte("sis_id,role\n12345,student\n")
	_, err := parseRosterCSV(raw, "student")
	if err == nil || !strings.Contains(err.Error(), "sis_id lookup is not supported") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseRosterCSV_IdempotentKeys(t *testing.T) {
	raw := []byte("email,role\nalice@uni.edu,student\nalice@uni.edu,student\n")
	rows, err := parseRosterCSV(raw, "student")
	if err != nil {
		t.Fatalf("parseRosterCSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 duplicate rows preserved for caller grouping", len(rows))
	}
}

func TestNormalizeEnrollmentStateAlias(t *testing.T) {
	cases := map[string]string{
		"concluded":   "withdrawn",
		"deactivated": "dropped",
		"reactivate":  "active",
		"active":      "active",
	}
	for in, want := range cases {
		got, err := normalizeEnrollmentStateAlias(in)
		if err != nil || got != want {
			t.Fatalf("normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestConfirmRosterExport(t *testing.T) {
	if err := confirmRosterExport(false); err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("expected FERPA refusal, got %v", err)
	}
	if err := confirmRosterExport(true); err != nil {
		t.Fatalf("expected nil with --yes, got %v", err)
	}
}

func TestFilterEnrollments(t *testing.T) {
	state := "active"
	rows := []enrollmentRow{
		{ID: "1", UserID: "u1", Role: "student", State: &state},
		{ID: "2", UserID: "u2", Role: "ta", State: &state},
		{ID: "3", UserID: "u3", Role: "student", InvitationPending: true},
	}
	filtered := filterEnrollments(rows, "student", "", "")
	if len(filtered) != 2 {
		t.Fatalf("role filter len = %d, want 2", len(filtered))
	}
	invited := filterEnrollments(rows, "", "", "invited")
	if len(invited) != 1 || !invited[0].InvitationPending {
		t.Fatalf("invited filter = %+v", invited)
	}
}

func TestChunkStrings(t *testing.T) {
	chunks := chunkStrings([]string{"a", "b", "c", "d", "e"}, 2)
	if len(chunks) != 3 || len(chunks[2]) != 1 {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestEnrollmentsList_Success(t *testing.T) {
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		listHandler: func(w http.ResponseWriter, r *http.Request) {
			state := "active"
			_ = json.NewEncoder(w).Encode(enrollmentsListBody{
				Enrollments: []enrollmentRow{
					{ID: "e1", UserID: "u1", DisplayName: strPtr("Alice"), Role: "student", State: &state},
				},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	var out bytes.Buffer
	enrollmentsListCmd.SetOut(&out)
	if err := enrollmentsListCmd.RunE(enrollmentsListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Alice") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestEnrollmentsExport_RequiresYes(t *testing.T) {
	setCfg("http://example", "key")
	resetEnrollmentsSectionsFlags()
	enrollmentsExportFlags.yes = false
	err := enrollmentsExportCmd.RunE(enrollmentsExportCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("err = %v", err)
	}
}

func TestEnrollmentsExport_JSON(t *testing.T) {
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		listHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(enrollmentsListBody{
				Enrollments: []enrollmentRow{{ID: "e1", UserID: "u1", Role: "student"}},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	enrollmentsExportFlags.yes = true
	enrollmentsExportFlags.format = "json"

	var out bytes.Buffer
	enrollmentsExportCmd.SetOut(&out)
	if err := enrollmentsExportCmd.RunE(enrollmentsExportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var rows []enrollmentRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "e1" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestEnrollmentsImport_Summary(t *testing.T) {
	var postedEmails string
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		postHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			postedEmails, _ = body["emails"].(string)
			_ = json.NewEncoder(w).Encode(addEnrollmentsResponse{
				Added: []string{"alice@uni.edu", "bob@uni.edu"},
			})
		},
	})
	defer srv.Close()

	csvFile := t.TempDir() + "/roster.csv"
	if err := writeTestFile(csvFile, "email,role\nalice@uni.edu,student\nbob@uni.edu,student\n"); err != nil {
		t.Fatal(err)
	}

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	enrollmentsImportFlags.file = csvFile

	var out bytes.Buffer
	enrollmentsImportCmd.SetOut(&out)
	if err := enrollmentsImportCmd.RunE(enrollmentsImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(postedEmails, "alice@uni.edu") || !strings.Contains(postedEmails, "bob@uni.edu") {
		t.Fatalf("posted emails = %q", postedEmails)
	}
	if !strings.Contains(out.String(), "added=2") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestEnrollmentsImport_IdempotentSecondRun(t *testing.T) {
	call := 0
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		postHandler: func(w http.ResponseWriter, r *http.Request) {
			call++
			resp := addEnrollmentsResponse{Added: []string{"alice@uni.edu"}}
			if call > 1 {
				resp = addEnrollmentsResponse{AlreadyEnrolled: []string{"alice@uni.edu"}}
			}
			_ = json.NewEncoder(w).Encode(resp)
		},
	})
	defer srv.Close()

	csvFile := t.TempDir() + "/roster.csv"
	if err := writeTestFile(csvFile, "email,role\nalice@uni.edu,student\n"); err != nil {
		t.Fatal(err)
	}

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	enrollmentsImportFlags.file = csvFile

	if err := enrollmentsImportCmd.RunE(enrollmentsImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	enrollmentsImportCmd.SetOut(&bytes.Buffer{})
	if err := enrollmentsImportCmd.RunE(enrollmentsImportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("second import: %v", err)
	}
}

func TestEnrollmentsSetState_Concluded(t *testing.T) {
	var gotState string
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		listHandler: func(w http.ResponseWriter, r *http.Request) {
			state := "active"
			_ = json.NewEncoder(w).Encode(enrollmentsListBody{
				Enrollments: []enrollmentRow{
					{ID: "e99", UserID: "aaaaaaaa-bbbb-cccc-dddd-000000000099", Role: "student", State: &state},
				},
			})
		},
		statePatchHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotState = body["state"]
			_ = json.NewEncoder(w).Encode(enrollmentStatePatchResponse{ID: "e99", State: gotState})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	enrollmentsSetStateFlags.user = "aaaaaaaa-bbbb-cccc-dddd-000000000099"
	enrollmentsSetStateFlags.state = "concluded"

	enrollmentsSetStateCmd.SetOut(&bytes.Buffer{})
	if err := enrollmentsSetStateCmd.RunE(enrollmentsSetStateCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotState != "withdrawn" {
		t.Fatalf("state sent = %q, want withdrawn", gotState)
	}
}

func TestEnrollmentsAdd_ByEmail(t *testing.T) {
	var posted map[string]any
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		postHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&posted)
			_ = json.NewEncoder(w).Encode(addEnrollmentsResponse{Added: []string{"alice@uni.edu"}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	enrollmentsAddFlags.user = "alice@uni.edu"
	enrollmentsAddFlags.role = "student"

	enrollmentsAddCmd.SetOut(&bytes.Buffer{})
	if err := enrollmentsAddCmd.RunE(enrollmentsAddCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if posted["courseRole"] != "student" {
		t.Fatalf("courseRole = %v", posted["courseRole"])
	}
}

func TestSectionsMove_Success(t *testing.T) {
	var transferred string
	srv := newEnrollmentsSectionsServer(t, enrollmentsSectionsServerConfig{
		listHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(enrollmentsListBody{
				Enrollments: []enrollmentRow{{ID: "e1", UserID: "11111111-aaaa-bbbb-cccc-000000000001", Role: "student"}},
			})
		},
		sectionsListHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(sectionsListBody{
				Sections: []sectionRow{{ID: "s2", SectionCode: "LAB-2", Status: "active"}},
			})
		},
		sectionTransfer: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			transferred = body["sectionId"]
			_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetEnrollmentsSectionsFlags()
	sectionsMoveFlags.user = "11111111-aaaa-bbbb-cccc-000000000001"
	sectionsMoveFlags.to = "LAB-2"

	sectionsMoveCmd.SetOut(&bytes.Buffer{})
	if err := sectionsMoveCmd.RunE(sectionsMoveCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if transferred != "s2" {
		t.Fatalf("sectionId = %q, want s2", transferred)
	}
}

func TestEnrollmentsCmd_HasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range enrollmentsCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "export", "import", "add", "remove", "set-state", "self-enroll"} {
		if !names[want] {
			t.Errorf("enrollments subcommand %q not registered", want)
		}
	}
}

func TestSectionsCmd_HasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range sectionsCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "create", "update", "delete", "move", "cross-list"} {
		if !names[want] {
			t.Errorf("sections subcommand %q not registered", want)
		}
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}