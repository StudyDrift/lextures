package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type assignmentsExtendServerConfig struct {
	getHandler           http.HandlerFunc
	patchAssignment      http.HandlerFunc
	patchStructureItem   http.HandlerFunc
	deleteStructureItem  http.HandlerFunc
	overridesGetHandler  http.HandlerFunc
	overridesPutHandler  http.HandlerFunc
	submissionsHandler   http.HandlerFunc
	gradeHistoryHandler  http.HandlerFunc
	annotateHandler      http.HandlerFunc
	commentHandler       http.HandlerFunc
	downloadHandler      http.HandlerFunc
	archiveHandler       http.HandlerFunc
}

func newAssignmentsExtendServer(t *testing.T, cfg assignmentsExtendServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && strings.HasSuffix(path, "/history"):
			if cfg.gradeHistoryHandler != nil {
				cfg.gradeHistoryHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && strings.HasSuffix(path, "/submissions"):
			if cfg.submissionsHandler != nil {
				cfg.submissionsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && strings.HasSuffix(path, "/attachments/archive"):
			if cfg.archiveHandler != nil {
				cfg.archiveHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && !strings.Contains(path, "/submissions"):
			if cfg.getHandler != nil {
				cfg.getHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/assignments/"):
			if cfg.patchAssignment != nil {
				cfg.patchAssignment(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/structure/items/"):
			if cfg.patchStructureItem != nil {
				cfg.patchStructureItem(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodDelete && strings.Contains(path, "/structure/items/"):
			if cfg.deleteStructureItem != nil {
				cfg.deleteStructureItem(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/items/") && strings.HasSuffix(path, "/overrides"):
			if cfg.overridesGetHandler != nil {
				cfg.overridesGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.Contains(path, "/items/") && strings.HasSuffix(path, "/overrides"):
			if cfg.overridesPutHandler != nil {
				cfg.overridesPutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.Contains(path, "/annotations"):
			if cfg.annotateHandler != nil {
				cfg.annotateHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.Contains(path, "/students/") && strings.HasSuffix(path, "/grade"):
			if cfg.commentHandler != nil {
				cfg.commentHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/course-files/"):
			if cfg.downloadHandler != nil {
				cfg.downloadHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func resetAssignmentsExtendFlags() {
	assignmentsUpdateFlags.course = ""
	assignmentsUpdateFlags.title = ""
	assignmentsUpdateFlags.points = -1
	assignmentsUpdateFlags.due = ""
	assignmentsUpdateFlags.file = ""
	assignmentsDeleteFlags.course = ""
	assignmentsPublishFlags.course = ""
	assignmentsGradeHistoryFlags.course = ""
	assignmentsGradeHistoryFlags.student = ""
	assignmentsOverridesListFlags.course = ""
	assignmentsOverridesSetFlags = struct {
		course         string
		section        string
		user           string
		due            string
		availableFrom  string
		availableUntil string
	}{}
	assignmentsOverridesDeleteFlags = struct {
		course  string
		section string
		user    string
	}{}
	assignmentsSubmissionsListFlags = struct {
		course string
		status string
		user   string
		late   bool
	}{}
	assignmentsSubmissionsGetFlags.course = ""
	assignmentsSubmissionsGetFlags.user = ""
	assignmentsSubmissionsDownloadFlags = struct {
		course       string
		out          string
		all          bool
		yes          bool
		user         string
		skipExisting bool
	}{skipExisting: true}
	assignmentsSubmissionsAnnotateFlags = struct {
		course     string
		submission string
		body       string
		tool       string
		page       int
	}{page: 1}
	assignmentsSubmissionsCommentFlags = struct {
		course  string
		user    string
		comment string
	}{}
}

func sampleAssignmentJSON() map[string]any {
	due := time.Date(2027, 9, 15, 23, 59, 0, 0, time.UTC).Format(time.RFC3339)
	return map[string]any{
		"itemId":               "item-001",
		"title":                "Homework 1",
		"markdown":             "Do the work.",
		"dueAt":                due,
		"pointsWorth":          100,
		"lateSubmissionPolicy": "allow",
		"postingPolicy":        "automatic",
		"blindGrading":         false,
		"moderatedGrading":     false,
		"neverDrop":            false,
		"replaceWithFinal":     false,
		"updatedAt":            time.Now().UTC().Format(time.RFC3339),
	}
}

func TestAssignmentsUpdate_PointsAndDue(t *testing.T) {
	var gotPatch map[string]any
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleAssignmentJSON())
		},
		patchAssignment: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotPatch)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleAssignmentJSON())
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsUpdateFlags.course = "CS101"
	assignmentsUpdateFlags.points = 50
	assignmentsUpdateFlags.due = "2027-10-01"

	assignmentsUpdateCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsUpdateCmd.RunE(assignmentsUpdateCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotPatch["pointsWorth"] != float64(50) {
		t.Errorf("pointsWorth = %v, want 50", gotPatch["pointsWorth"])
	}
	if _, ok := gotPatch["dueAt"]; !ok {
		t.Error("expected dueAt in patch")
	}
}

func TestAssignmentsUpdate_TitlePatchesStructureItem(t *testing.T) {
	structurePatched := false
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		patchStructureItem: func(w http.ResponseWriter, r *http.Request) {
			structurePatched = true
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsUpdateFlags.course = "CS101"
	assignmentsUpdateFlags.title = "Renamed"

	assignmentsUpdateCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsUpdateCmd.RunE(assignmentsUpdateCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !structurePatched {
		t.Fatal("expected structure item patch for title")
	}
}

func TestAssignmentsDelete_Success(t *testing.T) {
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		deleteStructureItem: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsDeleteFlags.course = "CS101"

	var out bytes.Buffer
	assignmentsDeleteCmd.SetOut(&out)
	if err := assignmentsDeleteCmd.RunE(assignmentsDeleteCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "item-001") {
		t.Errorf("output = %q", out.String())
	}
}

func TestAssignmentsPublish_Success(t *testing.T) {
	var gotBody map[string]any
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		patchStructureItem: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsPublishFlags.course = "CS101"

	assignmentsPublishCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsPublishCmd.RunE(assignmentsPublishCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotBody["published"] != true {
		t.Errorf("published = %v, want true", gotBody["published"])
	}
}

func TestAssignmentsOverridesSet_UserDue(t *testing.T) {
	existing := assignmentOverridesBody{Targets: []assignmentOverrideTarget{}}
	var putBody map[string]any
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		overridesGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(existing)
		},
		overridesPutHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(existing)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsOverridesSetFlags.course = "CS101"
	assignmentsOverridesSetFlags.user = "user-uuid-1"
	assignmentsOverridesSetFlags.due = "2026-09-01"

	assignmentsOverridesSetCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsOverridesSetCmd.RunE(assignmentsOverridesSetCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	targets, ok := putBody["targets"].([]any)
	if !ok || len(targets) != 1 {
		t.Fatalf("targets = %v", putBody["targets"])
	}
	target := targets[0].(map[string]any)
	if target["targetType"] != "student" || target["targetId"] != "user-uuid-1" {
		t.Errorf("target = %v", target)
	}
}

func TestAssignmentsOverridesDelete_RemovesUserTarget(t *testing.T) {
	existing := assignmentOverridesBody{
		Targets: []assignmentOverrideTarget{
			{ID: "o1", TargetType: "student", TargetID: overrideStrPtr("user-1"), DueAt: overrideStrPtr("2026-09-01T00:00:00Z")},
			{ID: "o2", TargetType: "section", TargetID: overrideStrPtr("sec-1")},
		},
	}
	var putBody map[string]any
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		overridesGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(existing)
		},
		overridesPutHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"targets": []any{}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsOverridesDeleteFlags.course = "CS101"
	assignmentsOverridesDeleteFlags.user = "user-1"

	assignmentsOverridesDeleteCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsOverridesDeleteCmd.RunE(assignmentsOverridesDeleteCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	targets := putBody["targets"].([]any)
	if len(targets) != 1 {
		t.Fatalf("expected 1 remaining target, got %d", len(targets))
	}
	if targets[0].(map[string]any)["targetType"] != "section" {
		t.Errorf("remaining target = %v", targets[0])
	}
}

func overrideStrPtr(s string) *string { return &s }

func TestAssignmentsSubmissionsList_FilterGraded(t *testing.T) {
	submissions := assignmentSubmissionsBody{
		Submissions: []assignmentSubmissionEntry{
			{ID: "s1", SubmittedBy: "u1", SubmittedAt: time.Now().Format(time.RFC3339), IsGraded: true},
			{ID: "s2", SubmittedBy: "u2", SubmittedAt: time.Now().Format(time.RFC3339), IsGraded: false},
		},
	}
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("graded") != "graded" {
				t.Errorf("graded query = %q, want graded", r.URL.Query().Get("graded"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(submissions)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsListFlags.course = "CS101"
	assignmentsSubmissionsListFlags.status = "graded"

	var out bytes.Buffer
	assignmentsSubmissionsListCmd.SetOut(&out)
	if err := assignmentsSubmissionsListCmd.RunE(assignmentsSubmissionsListCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "u1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestAssignmentsSubmissionsGet_UserSubmission(t *testing.T) {
	submissions := assignmentSubmissionsBody{
		Submissions: []assignmentSubmissionEntry{
			{ID: "s1", SubmittedBy: "u1", SubmittedAt: "2027-01-01T00:00:00Z", IsGraded: false, BodyText: "hello"},
		},
	}
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(submissions)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsGetFlags.course = "CS101"
	assignmentsSubmissionsGetFlags.user = "u1"

	var out bytes.Buffer
	assignmentsSubmissionsGetCmd.SetOut(&out)
	if err := assignmentsSubmissionsGetCmd.RunE(assignmentsSubmissionsGetCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "s1") || !strings.Contains(out.String(), "hello") {
		t.Errorf("output = %q", out.String())
	}
}

func TestAssignmentsSubmissionsDownload_AllRequiresYes(t *testing.T) {
	setCfg("http://localhost:0", "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsDownloadFlags.course = "CS101"
	assignmentsSubmissionsDownloadFlags.out = t.TempDir()
	assignmentsSubmissionsDownloadFlags.all = true

	err := assignmentsSubmissionsDownloadCmd.RunE(assignmentsSubmissionsDownloadCmd, []string{"item-001"})
	if err == nil {
		t.Fatal("expected FERPA refusal without --yes")
	}
	if !strings.Contains(err.Error(), "FERPA") {
		t.Errorf("err = %v", err)
	}
}

func TestAssignmentsSubmissionsDownload_BulkSummary(t *testing.T) {
	submissions := assignmentSubmissionsBody{
		Submissions: []assignmentSubmissionEntry{
			{
				ID: "s1", SubmittedBy: "u1", SubmittedAt: time.Now().Format(time.RFC3339),
				AttachmentFilename: "essay.pdf",
				AttachmentContentPath: "/api/v1/courses/CS101/course-files/file-1/content",
			},
			{
				ID: "s2", SubmittedBy: "u2", SubmittedAt: time.Now().Format(time.RFC3339),
				AttachmentFilename: "lab.pdf",
				AttachmentContentPath: "/api/v1/courses/CS101/course-files/file-2/content",
			},
		},
	}
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(submissions)
		},
		downloadHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("file-bytes"))
		},
	})
	defer srv.Close()

	outDir := filepath.Join(t.TempDir(), "subs")
	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsDownloadFlags.course = "CS101"
	assignmentsSubmissionsDownloadFlags.out = outDir
	assignmentsSubmissionsDownloadFlags.all = true
	assignmentsSubmissionsDownloadFlags.yes = true

	var out bytes.Buffer
	assignmentsSubmissionsDownloadCmd.SetOut(&out)
	if err := assignmentsSubmissionsDownloadCmd.RunE(assignmentsSubmissionsDownloadCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Downloaded 2 file(s)") {
		t.Errorf("output = %q", out.String())
	}
	files, _ := os.ReadDir(outDir)
	if len(files) < 2 {
		t.Errorf("expected files under %s, got %d entries", outDir, len(files))
	}
}

func TestAssignmentsSubmissionsDownload_RejectsTraversal(t *testing.T) {
	setCfg("http://localhost:0", "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsDownloadFlags.course = "CS101"
	assignmentsSubmissionsDownloadFlags.out = "../escape"
	assignmentsSubmissionsDownloadFlags.user = "u1"

	err := assignmentsSubmissionsDownloadCmd.RunE(assignmentsSubmissionsDownloadCmd, []string{"item-001"})
	if err == nil {
		t.Fatal("expected invalid output dir error")
	}
	if !strings.Contains(err.Error(), "invalid output directory") {
		t.Errorf("err = %v", err)
	}
}

func TestAssignmentsSubmissionsAnnotate_Success(t *testing.T) {
	var gotBody map[string]any
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		annotateHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"annotation": map[string]string{"id": "a1"}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsAnnotateFlags.course = "CS101"
	assignmentsSubmissionsAnnotateFlags.submission = "sub-1"
	assignmentsSubmissionsAnnotateFlags.body = "Nice work"

	assignmentsSubmissionsAnnotateCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsSubmissionsAnnotateCmd.RunE(assignmentsSubmissionsAnnotateCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotBody["body"] != "Nice work" || gotBody["toolType"] != "text" {
		t.Errorf("body = %v", gotBody)
	}
}

func TestAssignmentsSubmissionsComment_Success(t *testing.T) {
	var gotBody map[string]string
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		commentHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsSubmissionsCommentFlags.course = "CS101"
	assignmentsSubmissionsCommentFlags.user = "u1"
	assignmentsSubmissionsCommentFlags.comment = "Please revise section 2"

	assignmentsSubmissionsCommentCmd.SetOut(&bytes.Buffer{})
	if err := assignmentsSubmissionsCommentCmd.RunE(assignmentsSubmissionsCommentCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotBody["instructorComment"] != "Please revise section 2" {
		t.Errorf("comment = %q", gotBody["instructorComment"])
	}
}

func TestAssignmentsGradeHistory_Success(t *testing.T) {
	hist := assignmentGradeHistoryBody{
		Events: []assignmentGradeHistoryEvent{
			{ID: "e1", Action: "update", ChangedAt: "2026-01-01T00:00:00.000Z"},
		},
	}
	srv := newAssignmentsExtendServer(t, assignmentsExtendServerConfig{
		gradeHistoryHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hist)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetAssignmentsExtendFlags()
	assignmentsGradeHistoryFlags.course = "CS101"
	assignmentsGradeHistoryFlags.student = "u1"

	var out bytes.Buffer
	assignmentsGradeHistoryCmd.SetOut(&out)
	if err := assignmentsGradeHistoryCmd.RunE(assignmentsGradeHistoryCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "update") {
		t.Errorf("output = %q", out.String())
	}
}

func TestAssignmentsExtend_HasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range assignmentsCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{
		"update", "delete", "publish", "unpublish", "grade-history", "overrides", "submissions",
	} {
		if !names[want] {
			t.Errorf("assignments subcommand %q not registered", want)
		}
	}
}

func TestConfirmSensitiveExport(t *testing.T) {
	if err := confirmSensitiveExport(false); err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("expected FERPA error, got %v", err)
	}
	if err := confirmSensitiveExport(true); err != nil {
		t.Fatalf("expected nil with --yes, got %v", err)
	}
}

func TestSafeJoinOutput_RejectsTraversal(t *testing.T) {
	base := t.TempDir()
	_, err := safeJoinOutput(base, "../outside.txt")
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestOverrideTargetWriteFromFlags_RequiresDate(t *testing.T) {
	_, err := overrideTargetWriteFromFlags(assignmentOverrideSetOpts{
		user: "u1",
	})
	if err == nil || !strings.Contains(err.Error(), "provide at least one") {
		t.Fatalf("err = %v", err)
	}
}