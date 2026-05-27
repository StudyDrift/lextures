package dpa

import (
	"net"
	"testing"
	"time"

	repo "github.com/lextures/lextures/server/internal/repos/dpa"
)

func TestPermissionConstants_FourSegments(t *testing.T) {
	for _, perm := range []string{AdminPermission} {
		seg := 0
		for _, c := range perm {
			if c == ':' {
				seg++
			}
		}
		if seg != 3 {
			t.Errorf("permission %q must have 4 colon-delimited segments, got %d separators", perm, seg)
		}
	}
}

func TestReAcknowledgementWindow_ThirtyDays(t *testing.T) {
	if ReAcknowledgementWindow != 30*24*time.Hour {
		t.Errorf("ReAcknowledgementWindow = %v want 720h", ReAcknowledgementWindow)
	}
}

func TestErrSentinels_NotNil(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound must not be nil")
	}
	if ErrForbidden == nil {
		t.Error("ErrForbidden must not be nil")
	}
}

func TestGenerateNDPATemplate_RequiredFields(t *testing.T) {
	v := &repo.DPAVersion{
		VersionStr:  "2026-01-01",
		TemplateURL: "https://example.com/ndpa.pdf",
		EffectiveAt: time.Now().UTC(),
	}
	items := []repo.DataInventoryItem{
		{
			ElementName:             "Email address",
			Category:                "identity",
			Purpose:                 "Auth",
			LegalBasis:              "contract",
			SharedWithSubProcessors: false,
			SubProcessorNames:       []string{},
		},
		{
			ElementName:             "Assignment text",
			Category:                "academic",
			Purpose:                 "AI grading",
			LegalBasis:              "contract",
			SharedWithSubProcessors: true,
			SubProcessorNames:       []string{"OpenRouter"},
		},
	}
	tpl := GenerateNDPATemplate(v, items)
	if tpl.VendorName == "" {
		t.Error("VendorName must not be empty")
	}
	if tpl.DPAVersionStr != "2026-01-01" {
		t.Errorf("DPAVersionStr = %q want 2026-01-01", tpl.DPAVersionStr)
	}
	if tpl.TemplateURL != "https://example.com/ndpa.pdf" {
		t.Errorf("TemplateURL = %q", tpl.TemplateURL)
	}
	if len(tpl.SubProcessors) == 0 {
		t.Error("SubProcessors must not be empty")
	}
	if len(tpl.DataInventorySummary) != 2 {
		t.Errorf("DataInventorySummary len=%d want 2", len(tpl.DataInventorySummary))
	}
	if tpl.GeneratedAt == "" {
		t.Error("GeneratedAt must be set")
	}
	if _, err := time.Parse(time.RFC3339, tpl.GeneratedAt); err != nil {
		t.Errorf("GeneratedAt %q is not RFC3339: %v", tpl.GeneratedAt, err)
	}
}

func TestGenerateNDPATemplate_FallbackSubProcessors(t *testing.T) {
	v := &repo.DPAVersion{VersionStr: "2026-01-01", EffectiveAt: time.Now()}
	items := []repo.DataInventoryItem{
		{ElementName: "Email", Category: "identity", SubProcessorNames: []string{}},
	}
	tpl := GenerateNDPATemplate(v, items)
	if len(tpl.SubProcessors) == 0 {
		t.Error("SubProcessors fallback must return default list")
	}
}

func TestNDPASubProcessors_DeduplicatesNames(t *testing.T) {
	items := []repo.DataInventoryItem{
		{SubProcessorNames: []string{"OpenRouter", "AWS"}},
		{SubProcessorNames: []string{"OpenRouter"}},
	}
	subs := ndpaSubProcessors(items)
	seen := map[string]int{}
	for _, s := range subs {
		seen[s]++
	}
	for k, v := range seen {
		if v > 1 {
			t.Errorf("sub-processor %q appears %d times, want 1", k, v)
		}
	}
}

func TestSDPCCSVExport_EmptyInventory(t *testing.T) {
	// SDPCCSVExport requires a DB; this tests only the CSV writer path via a
	// helper that accepts items directly.
	items := []repo.DataInventoryItem{}
	csv, err := buildSDPCCSV(items)
	if err != nil {
		t.Fatalf("buildSDPCCSV: %v", err)
	}
	lines := splitLines(csv)
	if len(lines) < 1 {
		t.Error("CSV must have at least a header row")
	}
	if lines[0] == "" {
		t.Error("header row must not be empty")
	}
}

func TestSDPCCSVExport_RowCount(t *testing.T) {
	ret := 365
	items := []repo.DataInventoryItem{
		{ElementName: "Email", Category: "identity", Purpose: "auth", LegalBasis: "contract", RetentionDays: &ret, SharedWithSubProcessors: false, SubProcessorNames: []string{}},
		{ElementName: "Score", Category: "academic", Purpose: "grading", LegalBasis: "contract", SharedWithSubProcessors: true, SubProcessorNames: []string{"OpenRouter"}},
	}
	csv, err := buildSDPCCSV(items)
	if err != nil {
		t.Fatalf("buildSDPCCSV: %v", err)
	}
	lines := splitLines(csv)
	// header + 2 data rows
	if len(lines) < 3 {
		t.Errorf("CSV lines=%d want >= 3 (header+2 data rows)", len(lines))
	}
}

func TestParseIP_ValidHostPort(t *testing.T) {
	ip := parseIP("192.168.1.1:12345")
	if ip == nil {
		t.Fatal("parseIP should return non-nil for valid host:port")
	}
	if ip.String() != "192.168.1.1" {
		t.Errorf("ip=%q want 192.168.1.1", ip.String())
	}
}

func TestParseIP_BareIP(t *testing.T) {
	ip := parseIP("10.0.0.1")
	if ip == nil {
		t.Fatal("parseIP should return non-nil for bare IP")
	}
	if !ip.Equal(net.ParseIP("10.0.0.1")) {
		t.Errorf("ip=%v want 10.0.0.1", ip)
	}
}

func TestParseIP_InvalidReturnsNil(t *testing.T) {
	ip := parseIP("not-an-ip")
	if ip != nil {
		t.Errorf("parseIP(invalid) should return nil, got %v", ip)
	}
}

// buildSDPCCSV is a testable wrapper around the CSV generation logic.
func buildSDPCCSV(items []repo.DataInventoryItem) (string, error) {
	b, err := buildCSVBytes(items)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// splitLines splits CSV output into non-empty lines for counting.
func splitLines(s string) []string {
	var out []string
	for _, line := range splitOnNewline(s) {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitOnNewline(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
