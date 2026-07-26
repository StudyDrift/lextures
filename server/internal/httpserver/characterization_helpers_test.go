package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// collectJSONKeys recursively collects sorted JSON object key paths.
// Arrays contribute "[*]" for the element shape (union of keys across elements).
// Volatile values (IDs, timestamps) are ignored — only structure is pinned (TD.1 FR-5).
func collectJSONKeys(v any, prefix string, out map[string]struct{}) {
	switch x := v.(type) {
	case map[string]any:
		if prefix != "" {
			out[prefix] = struct{}{}
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			out[p] = struct{}{}
			collectJSONKeys(x[k], p, out)
		}
	case []any:
		arrPath := prefix + "[*]"
		if prefix == "" {
			arrPath = "[*]"
		}
		out[arrPath] = struct{}{}
		for _, el := range x {
			collectJSONKeys(el, arrPath, out)
		}
	default:
		// scalars: leaf already recorded when walking parent object
	}
}

func sortedJSONKeySet(body []byte) ([]string, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	collectJSONKeys(v, "", set)
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func formatKeySetGolden(status int, contentType string, keys []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "status: %d\n", status)
	fmt.Fprintf(&b, "content-type: %s\n", normalizeContentType(contentType))
	b.WriteString("keys:\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s\n", k)
	}
	return b.String()
}

func normalizeContentType(ct string) string {
	ct = strings.TrimSpace(strings.ToLower(ct))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct
}

func parseKeySetGolden(text string) (status int, contentType string, keys []string, err error) {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	inKeys := false
	for _, line := range lines {
		if strings.HasPrefix(line, "status:") {
			_, err = fmt.Sscanf(line, "status: %d", &status)
			if err != nil {
				return 0, "", nil, fmt.Errorf("status line: %w", err)
			}
			continue
		}
		if strings.HasPrefix(line, "content-type:") {
			contentType = strings.TrimSpace(strings.TrimPrefix(line, "content-type:"))
			continue
		}
		if line == "keys:" {
			inKeys = true
			continue
		}
		if inKeys {
			k := strings.TrimSpace(line)
			if k != "" {
				keys = append(keys, k)
			}
		}
	}
	return status, contentType, keys, nil
}

func assertCharacterizationSnapshot(t *testing.T, name string, status int, contentType string, body []byte) {
	t.Helper()
	keys, err := sortedJSONKeySet(body)
	if err != nil {
		// Non-JSON responses: pin status + content-type only.
		keys = nil
		if normalizeContentType(contentType) == "application/json" {
			t.Fatalf("%s: expected JSON body: %v\nbody=%s", name, err, truncateBytes(body, 200))
		}
	}
	got := formatKeySetGolden(status, contentType, keys)
	path := filepath.Join("testdata", "characterization", name+".golden")

	if updateGoldenRequested() {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s", path)
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\nRun with UPDATE_GOLDEN=1 to create it.", path, err)
	}
	wantStatus, wantCT, wantKeys, err := parseKeySetGolden(string(raw))
	if err != nil {
		t.Fatalf("parse golden %s: %v", path, err)
	}
	if status != wantStatus {
		t.Errorf("%s: status %d want %d", name, status, wantStatus)
	}
	gotCT := normalizeContentType(contentType)
	if gotCT != wantCT {
		t.Errorf("%s: content-type %q want %q", name, gotCT, wantCT)
	}
	// Compare key sets with explicit add/remove messaging (AC-4).
	wantSet := map[string]struct{}{}
	for _, k := range wantKeys {
		wantSet[k] = struct{}{}
	}
	gotSet := map[string]struct{}{}
	for _, k := range keys {
		gotSet[k] = struct{}{}
	}
	var missing, extra []string
	for k := range wantSet {
		if _, ok := gotSet[k]; !ok {
			missing = append(missing, k)
		}
	}
	for k := range gotSet {
		if _, ok := wantSet[k]; !ok {
			extra = append(extra, k)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("%s: JSON key set diverged", name)
		for _, k := range extra {
			t.Errorf("  + key %s", k)
		}
		for _, k := range missing {
			t.Errorf("  - key %s", k)
		}
		t.Errorf("If intentional: UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestCharacterization -count=1")
	}
}

func truncateBytes(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func charDo(t *testing.T, h http.Handler, method, path string, token string, body any) (int, string, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Header().Get("Content-Type"), rr.Body.Bytes()
}

func charDoJSON(t *testing.T, h http.Handler, method, path, token string, body any, dest any) int {
	t.Helper()
	code, _, raw := charDo(t, h, method, path, token, body)
	if dest != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, dest); err != nil {
			t.Fatalf("%s %s: json %v body=%s", method, path, err, truncateBytes(raw, 300))
		}
	}
	return code
}
