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

type structureServerConfig struct {
	getHandler      http.HandlerFunc
	reorderHandler  http.HandlerFunc
	createModHandler http.HandlerFunc
	patchModHandler http.HandlerFunc
	deleteModHandler http.HandlerFunc
	createItemHandler http.HandlerFunc
	patchItemHandler http.HandlerFunc
	deleteItemHandler http.HandlerFunc
	patchPageHandler http.HandlerFunc
	requirementsHandler http.HandlerFunc
}

func newStructureServer(t *testing.T, cfg structureServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/structure"):
			if cfg.getHandler != nil {
				cfg.getHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/structure/reorder"):
			if cfg.reorderHandler != nil {
				cfg.reorderHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/structure/modules"):
			if cfg.createModHandler != nil {
				cfg.createModHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/structure/modules/") && !strings.HasSuffix(path, "/requirements"):
			if cfg.patchModHandler != nil {
				cfg.patchModHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodDelete && strings.Contains(path, "/structure/modules/"):
			if cfg.deleteModHandler != nil {
				cfg.deleteModHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.Contains(path, "/structure/modules/"):
			if cfg.createItemHandler != nil {
				cfg.createItemHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/structure/items/"):
			if cfg.patchItemHandler != nil {
				cfg.patchItemHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodDelete && strings.Contains(path, "/structure/items/"):
			if cfg.deleteItemHandler != nil {
				cfg.deleteItemHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.Contains(path, "/content-pages/"):
			if cfg.patchPageHandler != nil {
				cfg.patchPageHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/requirements"):
			if cfg.requirementsHandler != nil {
				cfg.requirementsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func sampleModule(id, title string, order int) structureItemPublic {
	return structureItemPublic{
		ID:        id,
		SortOrder: order,
		Kind:      "module",
		Title:     title,
		Published: true,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func samplePage(id, title, moduleID string) structureItemPublic {
	parent := moduleID
	return structureItemPublic{
		ID:        id,
		SortOrder: 1,
		Kind:      "content_page",
		Title:     title,
		ParentID:  &parent,
		Published: false,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func resetStructureFlags() {
	structureGetFlags.tree = false
	structureApplyFlags.file = ""
	structureApplyFlags.dryRun = false
	modulesCreateFlags.title = ""
	modulesReorderFlags.order = ""
	modulesItemsAddFlags.typeFlag = ""
	modulesItemsAddFlags.title = ""
	modulesItemsAddFlags.url = ""
	modulesItemsReorderFlags.order = ""
	pagesCreateFlags.module = ""
	pagesCreateFlags.title = ""
	pagesCreateFlags.file = ""
	pagesCreateFlags.publish = false
	linksAddFlags.module = ""
	linksAddFlags.title = ""
	linksAddFlags.url = ""
	globalFlags.jsonOut = false
}

func TestParseOrderFlag(t *testing.T) {
	ids, err := parseOrderFlag("a,b, c")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Fatalf("got %v", ids)
	}
	if _, err := parseOrderFlag(""); err == nil {
		t.Fatal("expected error for empty order")
	}
	if _, err := parseOrderFlag("a,,b"); err == nil {
		t.Fatal("expected error for empty segment")
	}
}

func TestValidateItemKind(t *testing.T) {
	seg, err := validateItemKind("page")
	if err != nil || seg != "content-pages" {
		t.Fatalf("page: seg=%q err=%v", seg, err)
	}
	if _, err := validateItemKind("bogus"); err == nil {
		t.Fatal("expected error for bogus type")
	}
}

func TestStructureGet_JSON(t *testing.T) {
	modA := sampleModule("mod-a", "Week 1", 1)
	page := samplePage("page-1", "Intro", "mod-a")
	srv := newStructureServer(t, structureServerConfig{
		getHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: []structureItemPublic{modA, page}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetStructureFlags()
	globalFlags.jsonOut = true

	var out bytes.Buffer
	structureGetCmd.SetOut(&out)
	if err := structureGetCmd.RunE(structureGetCmd, []string{"CS101"}); err != nil {
		t.Fatal(err)
	}
	var body courseStructureBody
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(body.Items))
	}
}

func TestStructureGet_Tree(t *testing.T) {
	modA := sampleModule("mod-a", "Week 1", 1)
	page := samplePage("page-1", "Intro", "mod-a")
	srv := newStructureServer(t, structureServerConfig{
		getHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: []structureItemPublic{modA, page}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetStructureFlags()
	structureGetFlags.tree = true

	var out bytes.Buffer
	structureGetCmd.SetOut(&out)
	if err := structureGetCmd.RunE(structureGetCmd, []string{"CS101"}); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "Week 1") || !strings.Contains(text, "content_page") {
		t.Fatalf("tree output missing expected content: %q", text)
	}
}

func TestModulesReorder_SendsBody(t *testing.T) {
	var gotBody map[string]any
	srv := newStructureServer(t, structureServerConfig{
		reorderHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: []structureItemPublic{}})
		},
		getHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: []structureItemPublic{
				sampleModule("c", "C", 1),
				sampleModule("a", "A", 2),
				sampleModule("b", "B", 3),
			}})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetStructureFlags()
	modulesReorderFlags.order = "c,a,b"

	if err := modulesReorderCmd.RunE(modulesReorderCmd, []string{"CS101"}); err != nil {
		t.Fatal(err)
	}
	order, ok := gotBody["moduleOrder"].([]any)
	if !ok || len(order) != 3 {
		t.Fatalf("moduleOrder: %v", gotBody["moduleOrder"])
	}
}

func TestPagesCreate_WithFileAndPublish(t *testing.T) {
	var patchedPage string
	var published bool
	srv := newStructureServer(t, structureServerConfig{
		createItemHandler: func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/content-pages") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(samplePage("page-new", "Lesson", "mod-1"))
		},
		patchPageHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			patchedPage = body["markdown"]
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(contentPagePublic{ItemID: "page-new", Title: "Lesson", Markdown: patchedPage})
		},
		patchItemHandler: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]bool
			_ = json.NewDecoder(r.Body).Decode(&body)
			published = body["published"]
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(samplePage("page-new", "Lesson", "mod-1"))
		},
	})
	defer srv.Close()

	dir := t.TempDir()
	lessonPath := filepath.Join(dir, "lesson.md")
	if err := os.WriteFile(lessonPath, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	setCfg(srv.URL, "test-key")
	resetStructureFlags()
	pagesCreateFlags.module = "mod-1"
	pagesCreateFlags.title = "Lesson"
	pagesCreateFlags.file = lessonPath
	pagesCreateFlags.publish = true

	var out bytes.Buffer
	pagesCreateCmd.SetOut(&out)
	if err := pagesCreateCmd.RunE(pagesCreateCmd, []string{"CS101"}); err != nil {
		t.Fatal(err)
	}
	if patchedPage != "# Hello\n" {
		t.Fatalf("markdown = %q", patchedPage)
	}
	if !published {
		t.Fatal("expected publish patch")
	}
	if !strings.Contains(out.String(), "page-new") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestStructureApply_DryRun(t *testing.T) {
	modA := sampleModule("mod-a", "Week 1", 1)
	srv := newStructureServer(t, structureServerConfig{
		getHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(courseStructureBody{Items: []structureItemPublic{modA}})
		},
	})
	defer srv.Close()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "structure.json")
	spec := structureApplySpec{Modules: []structureApplyModule{
		{Title: "Week 1"},
		{Title: "Week 2"},
	}}
	raw, _ := json.Marshal(spec)
	if err := os.WriteFile(specPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	setCfg(srv.URL, "test-key")
	resetStructureFlags()
	structureApplyFlags.file = specPath
	structureApplyFlags.dryRun = true

	var out bytes.Buffer
	structureApplyCmd.SetOut(&out)
	if err := structureApplyCmd.RunE(structureApplyCmd, []string{"CS101"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[dry-run]") || !strings.Contains(out.String(), "created") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestComputeStructureDiff(t *testing.T) {
	current := []structureItemPublic{
		sampleModule("m1", "Old", 1),
	}
	desired := structureApplySpec{Modules: []structureApplyModule{
		{Title: "New Module"},
	}}
	summary := computeStructureDiff(current, desired)
	if summary.Created < 1 || summary.Deleted < 1 {
		t.Fatalf("summary: %+v", summary)
	}
}

func TestModulesItemsAdd_InvalidType(t *testing.T) {
	setCfg("http://localhost:0", "key")
	resetStructureFlags()
	modulesItemsAddFlags.typeFlag = "invalid"
	modulesItemsAddFlags.title = "X"
	if err := modulesItemsAddCmd.RunE(modulesItemsAddCmd, []string{"CS101", "mod-1"}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLinksAdd_RequiresURL(t *testing.T) {
	setCfg("http://localhost:0", "key")
	resetStructureFlags()
	linksAddFlags.module = "mod-1"
	linksAddFlags.title = "Docs"
	linksAddFlags.url = ""
	if err := linksAddCmd.RunE(linksAddCmd, []string{"CS101"}); err == nil {
		t.Fatal("expected url required error")
	}
}