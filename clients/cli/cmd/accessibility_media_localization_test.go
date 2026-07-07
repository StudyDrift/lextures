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

func TestValidateLocale(t *testing.T) {
	if err := validateLocale("es"); err != nil {
		t.Fatalf("es: %v", err)
	}
	if err := validateLocale(""); err == nil {
		t.Fatal("expected error for empty locale")
	}
}

func TestIsWebVTT(t *testing.T) {
	if !isWebVTT([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHi")) {
		t.Fatal("expected valid vtt")
	}
	if isWebVTT([]byte("not vtt")) {
		t.Fatal("expected invalid vtt")
	}
}

func TestAltTextList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/accessibility" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"altTextCoverage": map[string]any{"withAlt": 3, "total": 5, "percent": 60},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	altTextListFlags.course = "CS101"
	defer func() { altTextListFlags.course = "" }()
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	altTextListCmd.SetOut(&out)
	if err := altTextListCmd.RunE(altTextListCmd, nil); err != nil {
		t.Fatalf("alt-text list: %v", err)
	}
	if !strings.Contains(out.String(), "altTextCoverage") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCaptionsUpload_InvalidVTT(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.vtt"
	if err := os.WriteFile(path, []byte("not-vtt"), 0o600); err != nil {
		t.Fatal(err)
	}
	captionsUploadFlags.file = path
	defer func() { captionsUploadFlags.file = "" }()
	err := captionsUploadCmd.RunE(captionsUploadCmd, []string{"obj-1"})
	if err == nil || !strings.Contains(err.Error(), "WebVTT") {
		t.Fatalf("err = %v", err)
	}
}

func TestTranslationsCoverage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/translation-coverage" {
			_ = json.NewEncoder(w).Encode(map[string]any{"locales": map[string]any{"es": 80}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	translationsCoverageFlags.course = "CS101"
	defer func() { translationsCoverageFlags.course = "" }()
	globalFlags.jsonOut = true
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	translationsCoverageCmd.SetOut(&out)
	if err := translationsCoverageCmd.RunE(translationsCoverageCmd, nil); err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if !strings.Contains(out.String(), "es") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAccessibilityCheck_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/accessibility" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"altTextCoverage": map[string]any{"withAlt": 2, "total": 4, "percent": 50},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	accessibilityCheckCmd.SetOut(&out)
	if err := accessibilityCheckCmd.RunE(accessibilityCheckCmd, []string{"CS101"}); err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out.String(), "50%") {
		t.Fatalf("output = %q", out.String())
	}
}
