package contenttools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRegistryContract(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("BuildBuiltinRegistry: %v", err)
	}
	if reg.Size() < 1 {
		t.Fatal("expected at least noop_probe")
	}
	for _, m := range reg.List() {
		if err := ValidateManifest(m.Manifest, m.I18nBundle); err != nil {
			t.Errorf("manifest %s: %v", m.ID, err)
		}
		if m.ConfigCompiled == nil || m.StateCompiled == nil {
			t.Errorf("manifest %s: schemas not compiled", m.ID)
		}
	}
	if reg.Get("noop_probe") == nil {
		t.Fatal("noop_probe missing")
	}
}

func TestRegistryDuplicateIDFails(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get("noop_probe")
	_, err = NewRegistry([]*CompiledManifest{m, m})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestManifestBadSemver(t *testing.T) {
	m := validProbeManifest(t)
	m.Version = "not-semver"
	if err := ValidateManifest(m, map[string]string{"name": "x"}); err == nil {
		t.Fatal("expected semver error")
	}
}

func TestManifestBadSchema(t *testing.T) {
	m := validProbeManifest(t)
	m.ConfigSchema = json.RawMessage(`{"type":`)
	_, err := CompileManifest(m, map[string]string{"name": "x"})
	if err == nil {
		t.Fatal("expected schema compile error")
	}
}

func TestManifestUnknownAIFeature(t *testing.T) {
	m := validProbeManifest(t)
	m.AI = &AIDecl{FeatureID: "not_a_real_feature", Required: true}
	if err := ValidateManifest(m, map[string]string{"name": "x"}); err == nil || !strings.Contains(err.Error(), "ai.featureId") {
		t.Fatalf("expected ai.featureId error, got %v", err)
	}
}

func TestManifestOversizeMaxStateBytes(t *testing.T) {
	m := validProbeManifest(t)
	m.Storage.MaxStateBytes = PlatformMaxStateBytes + 1
	if err := ValidateManifest(m, map[string]string{"name": "x"}); err == nil {
		t.Fatal("expected maxStateBytes ceiling error")
	}
}

func TestManifestMissingI18n(t *testing.T) {
	m := validProbeManifest(t)
	if err := ValidateManifest(m, nil); err == nil || !strings.Contains(err.Error(), "i18n") {
		t.Fatalf("expected i18n error, got %v", err)
	}
}

func TestValidateConfigJSON_RequiredPrompt(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get("noop_probe")
	err = ValidateConfigJSON(m, json.RawMessage(`{}`))
	ve, ok := err.(*ConfigValidationError)
	if !ok || len(ve.Errors) == 0 {
		t.Fatalf("expected ConfigValidationError, got %v", err)
	}
	if ve.Errors[0].Path != "/prompt" {
		t.Fatalf("path = %q want /prompt (errors=%+v)", ve.Errors[0].Path, ve.Errors)
	}
	if err := ValidateConfigJSON(m, json.RawMessage(`{"prompt":"hi","answerKey":"secret"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestRedactSensitiveConfig(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get("noop_probe")
	out, err := RedactSensitiveConfig(m.ConfigSchema, json.RawMessage(`{"prompt":"hi","answerKey":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg["answerKey"]; ok {
		t.Fatal("answerKey should be stripped")
	}
	if cfg["prompt"] != "hi" {
		t.Fatalf("prompt = %v", cfg["prompt"])
	}
}

func TestValidateStateJSON_TooLarge(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get("noop_probe")
	big := make([]byte, m.Storage.MaxStateBytes+10)
	for i := range big {
		big[i] = 'a'
	}
	raw, _ := json.Marshal(map[string]any{"response": string(big)})
	if err := ValidateStateJSON(m, raw); err != ErrStateTooLarge {
		t.Fatalf("got %v want ErrStateTooLarge", err)
	}
}

func TestFilterCatalog_Allowlist(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	all := FilterCatalog(reg, nil, "", nil)
	if len(all) < 1 {
		t.Fatal("expected tools")
	}
	filtered := FilterCatalog(reg, []string{"inline_questions"}, "", nil)
	if len(filtered) != 0 {
		t.Fatalf("allowlist with unknown id should return empty, got %d", len(filtered))
	}
	only := FilterCatalog(reg, []string{"noop_probe"}, "", nil)
	if len(only) != 1 || only[0].ID != "noop_probe" {
		t.Fatalf("got %+v", only)
	}
}

func TestActiveForCourseAndKillSwitch(t *testing.T) {
	t.Setenv(EnvKillSwitch, "")
	if !ActiveForCourse(true) || !AvailableForCourse(true) {
		t.Fatal("expected available")
	}
	t.Setenv(EnvKillSwitch, "on")
	if AvailableForCourse(true) {
		t.Fatal("kill switch should 404")
	}
}

func TestRewriteLexToolFences(t *testing.T) {
	md := "hello\n```lex-tool\n{\"instanceId\":\"aaa\",\"toolId\":\"noop_probe\",\"v\":1}\n```\n"
	out := RewriteLexToolFences(md, map[string]string{"aaa": "bbb"})
	if !strings.Contains(out, `"instanceId":"bbb"`) {
		t.Fatalf("rewrite failed: %s", out)
	}
}

func TestRegistryStartupBudget(t *testing.T) {
	start := time.Now()
	const n = 500
	manifests := make([]*CompiledManifest, 0, n)
	base, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	probe := base.Get("noop_probe")
	for i := 0; i < n; i++ {
		cp := *probe
		cp.ID = "tool_" + strings.ReplaceAll(strings.TrimLeft("000"+itoa(i), "0"), " ", "")
		if cp.ID == "tool_" {
			cp.ID = "tool_0"
		}
		// Ensure unique snake_case ids.
		cp.ID = "synth_tool_" + itoa(i)
		manifests = append(manifests, &cp)
	}
	_, err = NewRegistry(manifests)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("registry build with %d tools took %s (> 50ms)", n, elapsed)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func validProbeManifest(t *testing.T) Manifest {
	t.Helper()
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg.Get("noop_probe").Manifest
}
