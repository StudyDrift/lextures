package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetContentExtrasFlags() {
	scormImportFlags.module = ""
	scormImportFlags.title = ""
	scormImportFlags.quiet = true
	scormProgressOut = nil
	h5pImportFlags.module = ""
	h5pImportFlags.title = ""
	h5pImportFlags.quiet = true
	h5pProgressOut = nil
	glossaryListFlags.sourceLocale = "en"
	glossaryListFlags.targetLocale = ""
	glossarySetFlags.file = ""
	glossarySetFlags.sourceLocale = "en"
	glossarySetFlags.targetLocale = ""
	glossaryAddFlags.sourceLocale = "en"
	glossaryAddFlags.targetLocale = ""
	glossaryAddFlags.sourceTerm = ""
	glossaryAddFlags.targetTerm = ""
	toolsAddFlags.register = false
	toolsAddFlags.name = ""
	toolsAddFlags.clientID = ""
	toolsAddFlags.issuer = ""
	toolsAddFlags.jwksURL = ""
	toolsAddFlags.oidcAuthURL = ""
	toolsAddFlags.tokenURL = ""
	toolsAddFlags.module = ""
	toolsAddFlags.toolID = ""
	toolsAddFlags.title = ""
	toolsAddFlags.resourceLink = ""
	toolsRemoveFlags.deactivate = false
	resourcesLinkFlags.module = ""
	resourcesLinkFlags.title = ""
	resourcesLinkFlags.kind = "library"
	resourcesLinkFlags.resourceType = ""
	resourcesLinkFlags.sourceURL = ""
	resourcesLinkFlags.provider = "vitalsource"
	resourcesLinkFlags.toolID = ""
	collabDocsExportFlags.yes = false
	collabDocsExportFlags.file = ""
	collabDocsExportInput = nil
	whiteboardsExportFlags.yes = false
	whiteboardsExportFlags.file = ""
	whiteboardsExportInput = nil
}

func sampleStructureWithKinds(kinds ...string) courseStructureBody {
	var items []structureItemPublic
	items = append(items, structureItemPublic{ID: "mod-1", Kind: "module", Title: "Week 1"})
	for i, kind := range kinds {
		id := kind + "-" + strings.Repeat("x", i+1)
		items = append(items, structureItemPublic{
			ID:       id,
			Kind:     kind,
			Title:    strings.ToUpper(kind),
			ParentID: strPtr("mod-1"),
		})
	}
	return courseStructureBody{Items: items}
}

// --- glossary parsing ---

func TestParseGlossaryCSV_WithHeader(t *testing.T) {
	csv := "sourceTerm,targetTerm\nhello,hola\nworld,mundo\n"
	entries, err := parseGlossaryCSV([]byte(csv), "en", "es")
	if err != nil {
		t.Fatalf("parseGlossaryCSV: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].SourceTerm != "hello" || entries[0].TargetTerm != "hola" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[0].SourceLocale != "en" || entries[0].TargetLocale != "es" {
		t.Fatalf("expected default locales, got %+v", entries[0])
	}
}

func TestParseGlossaryCSV_NoHeader(t *testing.T) {
	csv := "alpha,beta\n"
	entries, err := parseGlossaryCSV([]byte(csv), "en", "fr")
	if err != nil {
		t.Fatalf("parseGlossaryCSV: %v", err)
	}
	if len(entries) != 1 || entries[0].SourceTerm != "alpha" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestParseGlossaryJSON_Array(t *testing.T) {
	raw := `[{"sourceTerm":"cat","targetTerm":"gato","targetLocale":"es"}]`
	entries, err := parseGlossaryJSON([]byte(raw), "en", "es")
	if err != nil {
		t.Fatalf("parseGlossaryJSON: %v", err)
	}
	if len(entries) != 1 || entries[0].SourceTerm != "cat" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestParseGlossaryJSON_Wrapped(t *testing.T) {
	raw := `{"entries":[{"sourceTerm":"dog","targetTerm":"perro"}]}`
	entries, err := parseGlossaryJSON([]byte(raw), "en", "es")
	if err != nil {
		t.Fatalf("parseGlossaryJSON: %v", err)
	}
	if entries[0].TargetTerm != "perro" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestParseGlossaryFile_RejectsEmpty(t *testing.T) {
	_, err := parseGlossaryFile([]byte("   \n"), "en", "es")
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

// --- secret redaction ---

func TestRedactExternalTool_OmitsSecret(t *testing.T) {
	tool := externalToolPublic{
		ID:              "tool-1",
		Name:            "Publisher",
		ClientID:        "cid",
		ToolIssuer:      "https://issuer.example",
		ToolJWKSURL:     "https://issuer.example/jwks",
		ToolOidcAuthURL: "https://issuer.example/oidc",
		Active:          true,
		ClientSecret:    "super-secret-key",
	}
	out := redactExternalTool(tool)
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "super-secret") {
		t.Fatalf("secret leaked in JSON: %s", raw)
	}
	if _, ok := out["clientSecret"]; ok {
		t.Fatal("clientSecret key should not be present")
	}
}

// --- scorm / h5p list ---

func TestScormList_FiltersStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/structure") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleStructureWithKinds("scorm", "h5p", "page"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()

	var out bytes.Buffer
	scormListCmd.SetOut(&out)
	if err := scormListCmd.RunE(scormListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "scorm") || strings.Contains(out.String(), "h5p") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestH5PList_FiltersStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/structure") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleStructureWithKinds("scorm", "h5p"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()

	var out bytes.Buffer
	h5pListCmd.SetOut(&out)
	if err := h5pListCmd.RunE(h5pListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "h5p") || strings.Contains(out.String(), "scorm") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

// --- multipart import ---

func TestScormImport_MultipartUpload(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "package.zip")
	if err := os.WriteFile(zipPath, []byte("PK\x03\x04fake"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/scorm") {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
				t.Errorf("expected multipart content-type, got %q", r.Header.Get("Content-Type"))
				http.Error(w, "bad content type", http.StatusBadRequest)
				return
			}
			mr := multipart.NewReader(r.Body, params["boundary"])
			var gotFile bool
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("reading part: %v", err)
				}
				if part.FormName() == "file" {
					gotFile = true
				}
			}
			if !gotFile {
				t.Error("missing file part")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(structureItemPublic{ID: "scorm-item-1", Kind: "scorm", Title: "SCORM 1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()
	scormImportFlags.module = "mod-1"
	scormImportFlags.quiet = true

	var out bytes.Buffer
	scormImportCmd.SetOut(&out)
	if err := scormImportCmd.RunE(scormImportCmd, []string{"CS101", zipPath}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "scorm-item-1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

// --- glossary set ---

func TestGlossarySet_FromCSV(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "glossary.csv")
	if err := os.WriteFile(csvPath, []byte("sourceTerm,targetTerm\nterm1,term2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var posted int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/glossary") {
			posted++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(glossaryEntry{
				ID: "g1", SourceTerm: "term1", TargetTerm: "term2", SourceLocale: "en", TargetLocale: "es",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()
	glossarySetFlags.file = csvPath
	glossarySetFlags.targetLocale = "es"

	var out bytes.Buffer
	glossarySetCmd.SetOut(&out)
	if err := glossarySetCmd.RunE(glossarySetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if posted != 1 {
		t.Fatalf("expected 1 POST, got %d", posted)
	}
	if !strings.Contains(out.String(), "Loaded 1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

// --- tools register ---

func TestToolsAdd_Register_JSONRedactsSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/lti/external-tools" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "tool-1", "name": "Tool", "clientId": "cid",
				"toolIssuer": "https://issuer.example", "toolJwksUrl": "https://issuer.example/jwks",
				"toolOidcAuthUrl": "https://issuer.example/oidc", "active": true,
				"clientSecret": "must-not-appear",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()
	toolsAddFlags.register = true
	toolsAddFlags.name = "Tool"
	toolsAddFlags.clientID = "cid"
	toolsAddFlags.issuer = "https://issuer.example"
	toolsAddFlags.jwksURL = "https://issuer.example/jwks"
	toolsAddFlags.oidcAuthURL = "https://issuer.example/oidc"

	var out bytes.Buffer
	toolsAddCmd.SetOut(&out)
	if err := toolsAddCmd.RunE(toolsAddCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if strings.Contains(out.String(), "must-not-appear") {
		t.Fatalf("secret leaked: %s", out.String())
	}
}

// --- FERPA export gate ---

func TestCollabDocsExport_RequiresConfirmation(t *testing.T) {
	setCfg("http://example.invalid", "key")
	resetContentExtrasFlags()
	collabDocsExportInput = strings.NewReader("n\n")

	var out bytes.Buffer
	collabDocsExportCmd.SetOut(&out)
	err := collabDocsExportCmd.RunE(collabDocsExportCmd, []string{"CS101"})
	if err == nil {
		t.Fatal("expected abort error")
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestWhiteboardsExport_WithYes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/whiteboards") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(whiteboardsListBody{
				Whiteboards: []whiteboardPublic{{ID: "wb-1", Title: "Board 1"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()
	whiteboardsExportFlags.yes = true
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	whiteboardsExportCmd.SetOut(&out)
	if err := whiteboardsExportCmd.RunE(whiteboardsExportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "wb-1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestResourcesList_FiltersKinds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/structure") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sampleStructureWithKinds("library_resource", "textbook_resource", "page"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetContentExtrasFlags()

	var out bytes.Buffer
	resourcesListCmd.SetOut(&out)
	if err := resourcesListCmd.RunE(resourcesListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "library_resource") || !strings.Contains(out.String(), "textbook_resource") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}