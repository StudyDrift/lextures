package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

// extendedCoursesServerConfig adds handlers for lifecycle endpoints.
type extendedCoursesServerConfig struct {
	listHandler          http.HandlerFunc
	getHandler           http.HandlerFunc
	createHandler        http.HandlerFunc
	deleteHandler        http.HandlerFunc
	putHandler           http.HandlerFunc
	restoreHandler       http.HandlerFunc // PATCH archived=false
	cloneHandler         http.HandlerFunc
	syllabusGetHandler   http.HandlerFunc
	syllabusPatchHandler http.HandlerFunc
	featuresPatchHandler http.HandlerFunc
	heroImagePutHandler  http.HandlerFunc
	catalogGetHandler    http.HandlerFunc
	catalogPutHandler    http.HandlerFunc
	blueprintPushHandler http.HandlerFunc
	storageUsageHandler  http.HandlerFunc
}

func newExtendedCoursesServer(t *testing.T, cfg extendedCoursesServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && path == "/api/v1/courses":
			if cfg.listHandler != nil {
				cfg.listHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && path == "/api/v1/courses":
			if cfg.createHandler != nil {
				cfg.createHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && path == "/api/v1/courses/import/from-course":
			if cfg.cloneHandler != nil {
				cfg.cloneHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasPrefix(path, "/api/v1/courses/") &&
			!strings.Contains(path, "/hero-image") && !strings.Contains(path, "/catalog-listing"):
			if cfg.putHandler != nil {
				cfg.putHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/courses/") &&
			!strings.Contains(path, "/syllabus") && !strings.Contains(path, "/catalog-listing") &&
			!strings.Contains(path, "/storage-usage"):
			if cfg.getHandler != nil {
				cfg.getHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/archived"):
			if cfg.restoreHandler != nil {
				cfg.restoreHandler(w, r)
			} else if cfg.deleteHandler != nil {
				cfg.deleteHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/syllabus"):
			if cfg.syllabusGetHandler != nil {
				cfg.syllabusGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/syllabus"):
			if cfg.syllabusPatchHandler != nil {
				cfg.syllabusPatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/features"):
			if cfg.featuresPatchHandler != nil {
				cfg.featuresPatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/hero-image"):
			if cfg.heroImagePutHandler != nil {
				cfg.heroImagePutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/catalog-listing"):
			if cfg.catalogGetHandler != nil {
				cfg.catalogGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/catalog-listing"):
			if cfg.catalogPutHandler != nil {
				cfg.catalogPutHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/blueprint/push"):
			if cfg.blueprintPushHandler != nil {
				cfg.blueprintPushHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/storage-usage"):
			if cfg.storageUsageHandler != nil {
				cfg.storageUsageHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func sampleCourseDetail(code, title string, published bool) courseDetail {
	co := sampleCourse(code, title)
	co.Published = published
	return courseDetail{
		coursePublic: co,
		Description:  "A test course",
		ScheduleMode: "fixed",
		courseFeatures: courseFeatures{
			NotebookEnabled:    true,
			FeedEnabled:        true,
			CalendarEnabled:    true,
			DiscussionsEnabled: true,
		},
	}
}

func resetCoursesExtendFlags() {
	resetCoursesFlags()
	coursesUpdateFlags = coursesUpdateOpts{}
	coursesCloneFlags.toTerm = ""
	coursesCloneFlags.name = ""
	coursesSyllabusSetFlags.file = ""
	coursesSettingsSetFlags.file = ""
	coursesCatalogListingSetFlags.file = ""
}

func TestCoursePublic_JSONFieldParity(t *testing.T) {
	// Keys from server/internal/repos/course/list_enrolled.go CoursePublic.
	wantKeys := []string{
		"id", "courseCode", "title", "description", "published", "archived",
		"courseType", "scheduleMode", "startsAt", "endsAt", "visibleFrom", "hiddenAt",
		"relativeEndAfter", "relativeHiddenAfter", "orgId", "orgUnitId", "termId",
		"isBlueprint", "createdAt", "updatedAt",
	}
	typ := reflect.TypeOf(coursePublic{})
	got := make(map[string]bool)
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		key := strings.Split(tag, ",")[0]
		got[key] = true
	}
	for _, key := range wantKeys {
		if !got[key] {
			t.Errorf("coursePublic missing json tag %q", key)
		}
	}

	raw := `{
		"id":"11111111-0000-0000-0000-000000000001",
		"courseCode":"CS101",
		"title":"Intro",
		"description":"Desc",
		"published":true,
		"archived":false,
		"courseType":"traditional",
		"scheduleMode":"fixed",
		"isBlueprint":false,
		"createdAt":"2026-01-01T00:00:00Z",
		"updatedAt":"2026-01-02T00:00:00Z"
	}`
	var co coursePublic
	if err := json.Unmarshal([]byte(raw), &co); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if co.CourseCode != "CS101" || co.Description != "Desc" || co.ScheduleMode != "fixed" {
		t.Errorf("unexpected decode: %+v", co)
	}
}

func TestCoursesUpdate_Success(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Old Title", false)
	var gotBody map[string]any
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			detail.Title = "New Title"
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(detail.coursePublic)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesUpdateFlags.title = "New Title"

	var out bytes.Buffer
	coursesUpdateCmd.SetOut(&out)
	if err := coursesUpdateCmd.RunE(coursesUpdateCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotBody["title"] != "New Title" {
		t.Errorf("title = %v, want New Title", gotBody["title"])
	}
	if !strings.Contains(out.String(), "CS101") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCoursesUpdate_JSONOutput(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", false)
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail.coursePublic)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesUpdateFlags.title = "Title"
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	coursesUpdateCmd.SetOut(&out)
	if err := coursesUpdateCmd.RunE(coursesUpdateCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestCoursesUpdate_Forbidden(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", false)
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "You do not have permission for this action."})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesUpdateFlags.title = "X"

	err := coursesUpdateCmd.RunE(coursesUpdateCmd, []string{"CS101"})
	if err == nil {
		t.Fatal("expected 403 error")
	}
	if !strings.Contains(err.Error(), "permission") {
		t.Errorf("err = %v", err)
	}
}

func TestCoursesPublish_Idempotent(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", true)
	putCalled := false
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			putCalled = true
			_ = json.NewEncoder(w).Encode(detail.coursePublic)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	var out bytes.Buffer
	coursesPublishCmd.SetOut(&out)
	if err := coursesPublishCmd.RunE(coursesPublishCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if putCalled {
		t.Error("PUT should not be called when already published")
	}
	if !strings.Contains(out.String(), "already") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCoursesPublish_Success(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", false)
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["published"] != true {
				t.Errorf("published = %v, want true", body["published"])
			}
			detail.Published = true
			_ = json.NewEncoder(w).Encode(detail.coursePublic)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	coursesPublishCmd.SetOut(&bytes.Buffer{})
	if err := coursesPublishCmd.RunE(coursesPublishCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}

func TestCoursesPublish_JSONOutput(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", true)
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	coursesPublishCmd.SetOut(&out)
	if err := coursesPublishCmd.RunE(coursesPublishCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if result["changed"] != false {
		t.Errorf("changed = %v, want false", result["changed"])
	}
}

func TestCoursesUnpublish_Idempotent(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", false)
	putCalled := false
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			putCalled = true
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	coursesUnpublishCmd.SetOut(&bytes.Buffer{})
	if err := coursesUnpublishCmd.RunE(coursesUnpublishCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if putCalled {
		t.Error("PUT should not be called when already unpublished")
	}
}

func TestCoursesRestore_Success(t *testing.T) {
	var gotArchived any
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		restoreHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotArchived = body["archived"]
			co := sampleCourse("CS101", "Title")
			co.Archived = false
			_ = json.NewEncoder(w).Encode(co)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	var out bytes.Buffer
	coursesRestoreCmd.SetOut(&out)
	if err := coursesRestoreCmd.RunE(coursesRestoreCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotArchived != false {
		t.Errorf("archived = %v, want false", gotArchived)
	}
	if !strings.Contains(out.String(), "Restored") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCoursesClone_Success(t *testing.T) {
	src := sampleCourseDetail("TEMPLATE", "Template Course", true)
	var cloneBody map[string]any
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/TEMPLATE") {
				_ = json.NewEncoder(w).Encode(src)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/CS-NEW01") {
				co := sampleCourse("CS-NEW01", "Spring Template")
				_ = json.NewEncoder(w).Encode(co)
			}
		},
		cloneHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&cloneBody)
			co := sampleCourse("CS-NEW01", "Spring Template")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(co)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			co := sampleCourse("CS-NEW01", "Spring Template")
			_ = json.NewEncoder(w).Encode(co)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesCloneFlags.toTerm = "term-uuid-1"
	coursesCloneFlags.name = "Spring Template"

	var out bytes.Buffer
	coursesCloneCmd.SetOut(&out)
	if err := coursesCloneCmd.RunE(coursesCloneCmd, []string{"TEMPLATE"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if cloneBody["sourceCourseCode"] != "TEMPLATE" {
		t.Errorf("sourceCourseCode = %v", cloneBody["sourceCourseCode"])
	}
	if !strings.Contains(out.String(), "CS-NEW01") {
		t.Errorf("output = %q, want new course code", out.String())
	}
}

func TestCoursesSyllabusGet_JSON(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		syllabusGetHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sections":                  []any{},
				"updatedAt":                 "2026-01-01T00:00:00Z",
				"requireSyllabusAcceptance": false,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	coursesSyllabusGetCmd.SetOut(&out)
	if err := coursesSyllabusGetCmd.RunE(coursesSyllabusGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestCoursesSyllabusSet_Success(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		syllabusPatchHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"sections": []any{}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesSyllabusSetFlags.file = "-"
	coursesSyllabusSetCmd.SetIn(strings.NewReader(`{"sections":[],"requireSyllabusAcceptance":false}`))

	coursesSyllabusSetCmd.SetOut(&bytes.Buffer{})
	if err := coursesSyllabusSetCmd.RunE(coursesSyllabusSetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}

func TestCoursesSettingsGet_JSON(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", true)
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	coursesSettingsGetCmd.SetOut(&out)
	if err := coursesSettingsGetCmd.RunE(coursesSettingsGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if result["notebookEnabled"] != true {
		t.Errorf("notebookEnabled = %v", result["notebookEnabled"])
	}
}

func TestCoursesSettingsSet_Success(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		featuresPatchHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(sampleCourse("CS101", "Title"))
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesSettingsSetFlags.file = "-"
	coursesSettingsSetCmd.SetIn(strings.NewReader(`{"notebookEnabled":false,"feedEnabled":true,"calendarEnabled":true,"questionBankEnabled":true,"lockdownModeEnabled":false,"discussionsEnabled":true}`))

	coursesSettingsSetCmd.SetOut(&bytes.Buffer{})
	if err := coursesSettingsSetCmd.RunE(coursesSettingsSetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}

func TestCoursesHeroImageSet_URL(t *testing.T) {
	var gotBody map[string]any
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		heroImagePutHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(sampleCourse("CS101", "Title"))
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	coursesHeroImageSetCmd.SetOut(&bytes.Buffer{})
	if err := coursesHeroImageSetCmd.RunE(coursesHeroImageSetCmd, []string{"CS101", "https://example.com/img.png"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotBody["imageUrl"] != "https://example.com/img.png" {
		t.Errorf("imageUrl = %v", gotBody["imageUrl"])
	}
}

func TestCoursesCatalogListingSet_Success(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		catalogPutHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"listing": map[string]any{"isPublic": true}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesCatalogListingSetFlags.file = "-"
	coursesCatalogListingSetCmd.SetIn(strings.NewReader(`{"isPublic":true,"language":"en","priceCents":0,"slug":"cs101"}`))

	coursesCatalogListingSetCmd.SetOut(&bytes.Buffer{})
	if err := coursesCatalogListingSetCmd.RunE(coursesCatalogListingSetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}

func TestCoursesBlueprintSync_Success(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		blueprintPushHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"childrenTotal":   3,
				"childrenSuccess": 3,
				"childrenError":   0,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	coursesBlueprintSyncCmd.SetOut(&out)
	if err := coursesBlueprintSyncCmd.RunE(coursesBlueprintSyncCmd, []string{"BP01"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestCoursesStorageUsage_Success(t *testing.T) {
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		storageUsageHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"used_bytes":   1024,
				"limit_bytes":  4096,
				"percent_used": 25.0,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()

	var out bytes.Buffer
	coursesStorageUsageCmd.SetOut(&out)
	if err := coursesStorageUsageCmd.RunE(coursesStorageUsageCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "1024") {
		t.Errorf("output = %q", out.String())
	}
}

func TestCoursesCmd_HasExtendedSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range coursesCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{
		"list", "get", "create", "delete", "update", "publish", "unpublish",
		"restore", "clone", "syllabus", "settings", "hero-image", "catalog-listing",
		"blueprint", "storage-usage",
	} {
		if !names[want] {
			t.Errorf("courses subcommand %q not registered", want)
		}
	}
}

func TestCoursesUpdate_UserAgent(t *testing.T) {
	detail := sampleCourseDetail("CS101", "Title", false)
	var gotUA string
	srv := newExtendedCoursesServer(t, extendedCoursesServerConfig{
		getHandler: func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")
			_ = json.NewEncoder(w).Encode(detail)
		},
		putHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(detail.coursePublic)
		},
	})
	defer srv.Close()

	client.DefaultUserAgent = "lextures-cli/test"
	defer func() { client.DefaultUserAgent = "" }()

	setCfg(srv.URL, "test-key")
	resetCoursesExtendFlags()
	coursesUpdateFlags.title = "Title"

	coursesUpdateCmd.SetOut(&bytes.Buffer{})
	if err := coursesUpdateCmd.RunE(coursesUpdateCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotUA != "lextures-cli/test" {
		t.Errorf("User-Agent = %q, want lextures-cli/test", gotUA)
	}
}