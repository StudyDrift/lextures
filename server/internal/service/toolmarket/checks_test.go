package toolmarket

import (
	"encoding/json"
	"testing"
)

func sampleManifest(toolID, version string, sandbox string, hosts []string) json.RawMessage {
	m := map[string]any{
		"id":            toolID,
		"version":       version,
		"name":          "Titration Lab",
		"category":      "practice",
		"capabilities":  []string{"state", "network"},
		"configSchema":  map[string]any{"type": "object", "properties": map[string]any{}},
		"stateSchema":   map[string]any{"type": "object", "properties": map[string]any{}},
		"scoring":       map[string]any{"mode": "none"},
		"storage":       map[string]any{"maxStateBytes": 4096},
		"roles":         map[string]any{"interact": []string{"student"}},
		"a11y":          map[string]any{"keyboardOperable": true, "srPattern": "live"},
		"i18nNamespace": "tools.acme.titration",
		"ui":            map[string]any{"renderer": "iframe", "icon": "flask", "group": "science"},
		"sandbox":       sandbox,
		"dataSheet": map[string]any{
			"collects": map[string]any{
				"answer": map[string]any{"purpose": "practice", "retention": "course"},
			},
			"leavesPlatform": true,
			"processors":     []string{"acme.example"},
			"visibility":     "self",
			"wcagLevel":      "AA",
		},
		"network": map[string]any{"allowedHosts": hosts},
	}
	b, _ := json.Marshal(m)
	return b
}

func TestValidateMarketplaceToolID(t *testing.T) {
	if err := ValidateMarketplaceToolID("acme.titration_lab"); err != nil {
		t.Fatalf("expected valid id: %v", err)
	}
	if err := ValidateMarketplaceToolID("titration_lab"); err == nil {
		t.Fatal("expected non-namespaced id to fail")
	}
	if err := ValidateMarketplaceToolID("lextures.evil"); err == nil {
		t.Fatal("expected reserved namespace to fail")
	}
}

func TestRunAutomatedChecksPass(t *testing.T) {
	toolID := "acme.titration_lab"
	version := "1.0.0"
	manifest := sampleManifest(toolID, version, "iframe", []string{"api.example.com"})
	sheet := json.RawMessage(`{}`)
	rep := RunAutomatedChecks(toolID, version, manifest, sheet, []byte("bundle"), "pass", "pass", map[string]string{"title": "Lab"})
	if !rep.OK {
		t.Fatalf("expected pass, got %#v", rep.Checks)
	}
}

func TestRunAutomatedChecksRejectAxe(t *testing.T) {
	toolID := "acme.titration_lab"
	version := "1.0.0"
	manifest := sampleManifest(toolID, version, "iframe", []string{"api.example.com"})
	rep := RunAutomatedChecks(toolID, version, manifest, nil, []byte("bundle"), "fail", "pass", map[string]string{"title": "Lab"})
	if rep.OK {
		t.Fatal("expected axe failure to reject")
	}
	found := false
	for _, c := range rep.Checks {
		if c.Name == "axe" && !c.OK {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected named axe failure: %#v", rep.Checks)
	}
}

func TestRunAutomatedChecksRejectPrivateHost(t *testing.T) {
	toolID := "acme.titration_lab"
	version := "1.0.0"
	manifest := sampleManifest(toolID, version, "iframe", []string{"127.0.0.1"})
	rep := RunAutomatedChecks(toolID, version, manifest, nil, []byte("x"), "pass", "pass", map[string]string{"t": "1"})
	if rep.OK {
		t.Fatal("expected private host rejection")
	}
}

func TestRunAutomatedChecksRejectInprocessSandbox(t *testing.T) {
	toolID := "acme.titration_lab"
	version := "1.0.0"
	manifest := sampleManifest(toolID, version, "inprocess", []string{"api.example.com"})
	rep := RunAutomatedChecks(toolID, version, manifest, nil, []byte("x"), "pass", "pass", map[string]string{"t": "1"})
	if rep.OK {
		t.Fatal("expected inprocess sandbox rejection")
	}
}

func TestCapabilityPlainLanguage(t *testing.T) {
	if CapabilityPlainLanguage("network") == "" {
		t.Fatal("expected plain language")
	}
	if got := CapabilityPlainLanguage("network"); got != "Sends data to an external service" {
		t.Fatalf("got %q", got)
	}
}

func TestComputeSRI(t *testing.T) {
	sri := ComputeSRI([]byte("hello"))
	if sri[:7] != "sha256-" {
		t.Fatalf("unexpected sri %q", sri)
	}
}

func TestForceIframeManifest(t *testing.T) {
	raw := sampleManifest("acme.x", "1.0.0", "inprocess", nil)
	out, err := ForceIframeManifest(raw, "acme.x", "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	if m["sandbox"] != "iframe" {
		t.Fatalf("sandbox not forced: %v", m["sandbox"])
	}
	if m["version"] != "1.2.0" {
		t.Fatalf("version not set: %v", m["version"])
	}
}
