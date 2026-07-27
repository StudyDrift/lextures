package contenttools

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/boardfilter"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

func TestEvaluateToolPolicy_DenyCapability(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	// Give noop_probe an AI capability via a synthetic copy.
	m := *reg.Get("noop_probe")
	m.Capabilities = append(append([]string{}, m.Capabilities...), "ai")
	m.AI = &AIDecl{FeatureID: "content_tool", Required: false}
	pol := &ctrepo.PolicyRow{
		DeniedCapabilities: []string{"ai"},
	}
	dec := EvaluateToolPolicy(pol, &m, nil)
	if dec.Allowed || dec.Reason != PolicyDenialCapability {
		t.Fatalf("want capability denial, got %+v", dec)
	}
}

func TestEvaluateToolPolicy_Allowlist(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get("noop_probe")
	pol := &ctrepo.PolicyRow{AllowedToolIDs: []string{"sandbox_probe"}}
	dec := EvaluateToolPolicy(pol, m, nil)
	if dec.Allowed || dec.Reason != PolicyDenialAllowlist {
		t.Fatalf("want allowlist denial, got %+v", dec)
	}
}

func TestEvaluateToolPolicy_AIKillEnv(t *testing.T) {
	t.Setenv(EnvAIKillSwitch, "on")
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := *reg.Get("noop_probe")
	m.Capabilities = append(append([]string{}, m.Capabilities...), "ai")
	dec := EvaluateToolPolicy(nil, &m, nil)
	if dec.Allowed || dec.Reason != PolicyDenialKillAI {
		t.Fatalf("want AI kill denial, got %+v", dec)
	}
}

func TestScreenFreeText_Block(t *testing.T) {
	term := boardfilter.DefaultEnglish[0]
	res := ScreenFreeText("hello "+term, FilterActionBlock, true)
	if res.Action != FilterActionBlock || res.Category != FilterCategoryProfanity {
		t.Fatalf("want block, got %+v", res)
	}
	if res.Guidance == "" {
		t.Fatal("expected guidance")
	}
}

func TestScreenFreeText_Crisis(t *testing.T) {
	res := ScreenFreeText("I want to kill myself", FilterActionFlag, true)
	if !res.Crisis || res.Category != FilterCategoryCrisis {
		t.Fatalf("want crisis, got %+v", res)
	}
}

func TestConformanceGate_BuiltinsPass(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if err := MustConformanceOK(reg); err != nil {
		t.Fatal(err)
	}
	rep := EvaluateConformance(reg, nil, nil)
	if !rep.OK || len(rep.Tools) < 2 {
		t.Fatalf("unexpected report: %+v", rep)
	}
}

func TestValidateDataSheet_Required(t *testing.T) {
	m := validProbeManifest(t)
	m.DataSheet = nil
	if err := ValidateManifest(m, map[string]string{"name": "x"}); err == nil || !strings.Contains(err.Error(), "dataSheet") {
		t.Fatalf("expected dataSheet error, got %v", err)
	}
}

func TestExtractFreeTextFromState(t *testing.T) {
	got := ExtractFreeTextFromState([]byte(`{"response":"hello","attempts":1}`))
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}
