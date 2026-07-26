package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestServeOpenAPI(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeOpenAPI(rr, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if rr.Code != 200 {
		t.Fatalf("code: %d", rr.Code)
	}
	if g := strings.TrimSpace(rr.Header().Get("Content-Type")); !strings.HasPrefix(g, "application/json") {
		t.Fatalf("content-type: %q", g)
	}
	if g := rr.Header().Get("Cache-Control"); !strings.Contains(g, "max-age=") {
		t.Fatalf("expected Cache-Control max-age, got %q", g)
	}
	var doc map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi version: %v", doc["openapi"])
	}
	info, _ := doc["info"].(map[string]any)
	if info == nil || info["title"] != "StudyDrift API" {
		t.Fatalf("info: %#v", doc["info"])
	}
}

func TestServeOpenAPI_MethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeOpenAPI(rr, httptest.NewRequest(http.MethodPost, "/api/openapi.json", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code: %d", rr.Code)
	}
}

func TestServeDocs(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeDocs(rr, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	if rr.Code != 200 {
		t.Fatalf("code: %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "swagger-ui") && !strings.Contains(body, "Swagger") {
		t.Fatalf("expected swagger in html")
	}
	if !strings.Contains(body, "/api/openapi.json") {
		t.Fatalf("expected spec url in html")
	}
}

func TestServeDocs_MethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeDocs(rr, httptest.NewRequest(http.MethodPut, "/api/docs", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code: %d", rr.Code)
	}
}

// TestSpec_ValidJSONNoTrailing asserts FR-1 / AC-1: the embedded document is
// strict JSON with zero trailing bytes (the class of defect TD.3 repairs).
func TestSpec_ValidJSONNoTrailing(t *testing.T) {
	t.Parallel()
	dec := json.NewDecoder(bytes.NewReader(specBytes))
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}
	if dec.More() {
		t.Fatalf("spec has trailing data after first JSON value (offset %d of %d)", dec.InputOffset(), len(specBytes))
	}
	// Decoder may stop before trailing whitespace; ensure only whitespace remains.
	rest := bytes.TrimSpace(specBytes[dec.InputOffset():])
	if len(rest) != 0 {
		t.Fatalf("spec has non-whitespace trailing data: %q…", truncate(rest, 80))
	}
}

// TestSpec_OpenAPI303Structure asserts FR-2 / AC-2 structural conformance to
// OpenAPI 3.0.3 (required root fields and types — not a full meta-schema run).
func TestSpec_OpenAPI303Structure(t *testing.T) {
	t.Parallel()
	doc := mustParseSpec(t)

	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi: want 3.0.3, got %#v", doc["openapi"])
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info must be an object")
	}
	for _, k := range []string{"title", "version"} {
		if _, ok := info[k].(string); !ok || info[k] == "" {
			t.Fatalf("info.%s must be a non-empty string", k)
		}
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatal("paths must be a non-empty object")
	}
	for path, raw := range paths {
		if !strings.HasPrefix(path, "/") {
			t.Errorf("path key %q must start with /", path)
		}
		item, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("paths[%q] must be an object", path)
			continue
		}
		for method, opRaw := range item {
			switch strings.ToLower(method) {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
				op, ok := opRaw.(map[string]any)
				if !ok {
					t.Errorf("%s %s: operation must be object", method, path)
					continue
				}
				if _, ok := op["responses"].(map[string]any); !ok {
					t.Errorf("%s %s: responses required", method, path)
				}
			case "parameters", "summary", "description", "servers", "$ref":
				// Path-item fields, not operations.
			default:
				// Vendor extensions (x-*) and other path-item fields are allowed.
				_ = opRaw
			}
		}
	}
	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatal("components must be present (TD.3 repair reattached this block)")
	}
	if _, ok := components["schemas"].(map[string]any); !ok {
		t.Fatal("components.schemas must be an object")
	}
}

// TestSpec_BearerAuthPresent asserts FR-4 / AC-3: the orphaned components block
// was reattached, including securitySchemes.bearerAuth.
func TestSpec_BearerAuthPresent(t *testing.T) {
	t.Parallel()
	doc := mustParseSpec(t)
	components, _ := doc["components"].(map[string]any)
	schemes, _ := components["securitySchemes"].(map[string]any)
	bearer, ok := schemes["bearerAuth"].(map[string]any)
	if !ok {
		t.Fatal("components.securitySchemes.bearerAuth missing")
	}
	if bearer["type"] != "http" {
		t.Fatalf("bearerAuth.type: %#v", bearer["type"])
	}
	if bearer["scheme"] != "bearer" {
		t.Fatalf("bearerAuth.scheme: %#v", bearer["scheme"])
	}
}

// TestSpec_AllRefsResolve asserts FR-6 / AC-5.
func TestSpec_AllRefsResolve(t *testing.T) {
	t.Parallel()
	doc := mustParseSpec(t)
	var refs []string
	collectRefs(doc, &refs)
	if len(refs) == 0 {
		t.Fatal("expected at least one $ref in the document")
	}
	var missing []string
	for _, ref := range refs {
		if !strings.HasPrefix(ref, "#/") {
			missing = append(missing, ref+" (external/non-local refs are not supported in this guard)")
			continue
		}
		if !resolveLocalRef(doc, ref) {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d unresolved $ref(s):\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestSpec_NothingLostInRepair asserts AC-8 against a minimal floor of known
// content that lived in the pre-repair document (paths + restored schemas).
func TestSpec_NothingLostInRepair(t *testing.T) {
	t.Parallel()
	doc := mustParseSpec(t)
	paths, _ := doc["paths"].(map[string]any)
	requiredPaths := []string{
		"/health",
		"/api/v1/platform/features",
		"/api/v1/settings/permissions", // restored by TD.3 (was orphaned methods)
		"/api/v1/settings/ai",
		"/api/v1/transcripts/orders",
	}
	for _, p := range requiredPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("missing path %s (repair must not drop documentation)", p)
		}
	}
	components, _ := doc["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	requiredSchemas := []string{
		"CatalogListing",
		"AdaptiveContentUnit",
		"AdaptiveContentVariant",
		"ContentToolsSettings",
		"ContentToolInstance",
	}
	for _, s := range requiredSchemas {
		if _, ok := schemas[s]; !ok {
			t.Errorf("missing schema %s (components must be fully restored)", s)
		}
	}
	if len(paths) < 200 {
		t.Fatalf("expected ≥200 documented paths after repair, got %d", len(paths))
	}
	if len(schemas) < 30 {
		t.Fatalf("expected ≥30 schemas after repair, got %d", len(schemas))
	}
}

// TestSpec_DocumentedPathsExist asserts FR-11: every documented path maps to a
// registered route (exact, trailing-slash-normalized, or chi wildcard prefix).
func TestSpec_DocumentedPathsExist(t *testing.T) {
	invPath := routeInventoryPath(t)
	registered := loadInventoryPatterns(t, invPath)
	doc := mustParseSpec(t)
	paths, _ := doc["paths"].(map[string]any)

	var phantoms []string
	for p := range paths {
		if !pathCoveredByInventory(p, registered) {
			phantoms = append(phantoms, p)
		}
	}
	sort.Strings(phantoms)
	if len(phantoms) > 0 {
		t.Fatalf("%d documented path(s) have no matching registered route:\n  %s",
			len(phantoms), strings.Join(phantoms, "\n  "))
	}
}

// TestSpec_CoverageBaseline asserts FR-7 / AC-7: absolute documented path count
// must not fall below the checked-in baseline (shrink-only ratchet).
func TestSpec_CoverageBaseline(t *testing.T) {
	baselinePath := coverageBaselinePath(t)
	minPaths := loadCoverageBaseline(t, baselinePath)
	doc := mustParseSpec(t)
	paths, _ := doc["paths"].(map[string]any)
	n := len(paths)

	invPath := routeInventoryPath(t)
	registered := loadInventoryPatterns(t, invPath)
	total := len(registered)
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(n) / float64(total)
	}
	t.Logf("OpenAPI documentation coverage: %d / %d unique routes (%.1f%%)", n, total, pct)

	if n < minPaths {
		t.Fatalf("documented paths %d < baseline %d — documentation coverage decreased (TD.3 ratchet). "+
			"Restore the removed path docs or lower the baseline only with explicit review.", n, minPaths)
	}
}

func mustParseSpec(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(specBytes, &doc); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}
	return doc
}

func collectRefs(v any, out *[]string) {
	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok {
			*out = append(*out, ref)
		}
		for _, child := range x {
			collectRefs(child, out)
		}
	case []any:
		for _, child := range x {
			collectRefs(child, out)
		}
	}
}

func resolveLocalRef(doc map[string]any, ref string) bool {
	// JSON Pointer after #/
	ptr := strings.TrimPrefix(ref, "#/")
	if ptr == ref || ptr == "" {
		return false
	}
	var cur any = doc
	for _, part := range strings.Split(ptr, "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur, ok = m[part]
		if !ok {
			return false
		}
	}
	return true
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n])
}

var paramSegment = regexp.MustCompile(`\{[^/]+\}`)

func normalizePath(p string) string {
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		p = "/"
	}
	return paramSegment.ReplaceAllString(p, "{}")
}

func pathCoveredByInventory(docPath string, registered []string) bool {
	docNorm := normalizePath(docPath)
	for _, reg := range registered {
		// Chi wildcard mount: /oneroster/v1p2/* covers /oneroster/v1p2/users
		if strings.HasSuffix(reg, "/*") {
			prefix := strings.TrimSuffix(reg, "/*")
			if docPath == prefix || strings.HasPrefix(docPath, prefix+"/") {
				return true
			}
			continue
		}
		if normalizePath(reg) == docNorm {
			return true
		}
	}
	return false
}

func loadInventoryPatterns(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read route inventory: %v", err)
	}
	seen := map[string]struct{}{}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		pat := parts[1]
		if _, ok := seen[pat]; ok {
			continue
		}
		seen[pat] = struct{}{}
		out = append(out, pat)
	}
	return out
}

func loadCoverageBaseline(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read coverage baseline: %v", err)
	}
	minPaths := 0
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: min_documented_paths=N
		if strings.HasPrefix(line, "min_documented_paths=") {
			n, err := atoi(line[len("min_documented_paths="):])
			if err != nil {
				t.Fatalf("parse baseline: %v", err)
			}
			minPaths = n
		}
	}
	if minPaths <= 0 {
		t.Fatalf("baseline %s missing min_documented_paths", path)
	}
	return minPaths
}

func atoi(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errBaseline
	}
	v := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errBaseline
		}
		v = v*10 + int(ch-'0')
	}
	return v, nil
}

type baselineError string

func (e baselineError) Error() string { return string(e) }

const errBaseline = baselineError("invalid baseline integer")

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func routeInventoryPath(t *testing.T) string {
	t.Helper()
	// server/internal/openapi -> server/internal/httpserver/testdata/...
	p := filepath.Join(packageDir(t), "..", "httpserver", "testdata", "route_inventory.golden")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("route inventory golden missing (TD.1 dependency): %v", err)
	}
	return p
}

func coverageBaselinePath(t *testing.T) string {
	t.Helper()
	// Prefer repo-root scripts/allowlists; fall back to package-relative walk.
	candidates := []string{
		filepath.Join(packageDir(t), "..", "..", "..", "scripts", "allowlists", "openapi-coverage.txt"),
		filepath.Join(packageDir(t), "..", "..", "..", "..", "scripts", "allowlists", "openapi-coverage.txt"),
	}
	// Also try from cwd (go test often runs with cwd = package dir).
	wd, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(wd, "scripts", "allowlists", "openapi-coverage.txt"),
		filepath.Join(wd, "..", "scripts", "allowlists", "openapi-coverage.txt"),
		filepath.Join(wd, "..", "..", "scripts", "allowlists", "openapi-coverage.txt"),
		filepath.Join(wd, "..", "..", "..", "scripts", "allowlists", "openapi-coverage.txt"),
	)
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	t.Fatal("scripts/allowlists/openapi-coverage.txt not found")
	return ""
}
