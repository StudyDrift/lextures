package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/worked_example"
)

func sampleWorkedExampleConfig() worked_example.Config {
	return worked_example.Config{
		Problem:         "Expand $3(x+2)$.",
		Variables:       []string{"x"},
		BlankPolicy:     worked_example.BlankAuthor,
		AttemptsPerStep: 3,
		PracticeOnly:    true,
		ShowAllSteps:    false,
		Steps: []worked_example.Step{
			{
				ID:    "s1",
				Label: "Step 1",
				Text:  "Distribute the 3:",
				Blank: &worked_example.Blank{
					Type:     worked_example.BlankExpression,
					Expected: "3(x+2)",
				},
				Hints:       []string{"Multiply 3 by each term inside.", "You should get a sum of two terms."},
				Explanation: "3(x+2) = 3x + 6",
			},
			{
				ID:   "s2",
				Text: "Simplified form:",
				Blank: &worked_example.Blank{
					Type:     worked_example.BlankExpression,
					Expected: "3x+6",
				},
				Explanation: "Collect like terms: 3x + 6",
			},
			{
				ID:   "s3",
				Text: "Numeric check at x=1:",
				Blank: &worked_example.Blank{
					Type:      worked_example.BlankNumeric,
					Expected:  9.0,
					Tolerance: &worked_example.Tolerance{Kind: worked_example.ToleranceAbsolute, Value: 0.01},
				},
			},
		},
	}
}

func TestWorkedExampleCheckHintReveal(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(worked_example.ID)
	if m == nil {
		t.Fatal("missing worked_example manifest")
	}

	cfg := sampleWorkedExampleConfig()
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(worked_example.EmptyState())
	enroll := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	// Wrong expression → incorrect
	in, _ := json.Marshal(map[string]any{"stepId": "s1", "value": "3x+5"})
	res, err := contenttools.DispatchAction(m, "check_step", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    stJSON,
		Input:        in,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["result"] != "incorrect" {
		t.Fatalf("want incorrect: %#v", res.Result)
	}

	// Hint
	hin, _ := json.Marshal(map[string]any{"stepId": "s1"})
	hres, err := contenttools.DispatchAction(m, "hint", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    res.StatePatch,
		Input:        hin,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hres.Result["hint"] != "Multiply 3 by each term inside." {
		t.Fatalf("hint: %#v", hres.Result)
	}

	// Correct via normalisation
	in2, _ := json.Marshal(map[string]any{"stepId": "s1", "value": "3x + 6"})
	res2, err := contenttools.DispatchAction(m, "check_step", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    hres.StatePatch,
		Input:        in2,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Result["result"] != "correct" {
		t.Fatalf("want correct via normaliser: %#v", res2.Result)
	}
	if res2.Result["nextStep"] != "s2" {
		t.Fatalf("nextStep: %#v", res2.Result)
	}

	// Exhaust s2 then reveal
	st := res2.StatePatch
	for i := 0; i < 3; i++ {
		wrong, _ := json.Marshal(map[string]any{"stepId": "s2", "value": "x"})
		r, err := contenttools.DispatchAction(m, "check_step", contenttools.ActionContext{
			ConfigJSON:   cfgJSON,
			StateJSON:    st,
			Input:        wrong,
			EnrollmentID: enroll,
		})
		if err != nil {
			t.Fatal(err)
		}
		if r.StatePatch != nil {
			st = r.StatePatch
		}
	}
	revIn, _ := json.Marshal(map[string]any{"stepId": "s2"})
	rev, err := contenttools.DispatchAction(m, "reveal_step", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    st,
		Input:        revIn,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rev.Result["revealed"] != true {
		t.Fatalf("reveal: %#v", rev.Result)
	}
	if rev.Result["explanation"] == "" {
		t.Fatal("expected explanation")
	}

	// Locale numeric on s3
	numIn, _ := json.Marshal(map[string]any{"stepId": "s3", "value": "9,00"})
	num, err := contenttools.DispatchAction(m, "check_step", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    rev.StatePatch,
		Input:        numIn,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if num.Result["result"] != "correct" {
		t.Fatalf("numeric locale: %#v", num.Result)
	}
}

func TestWorkedExampleRedaction(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(worked_example.ID)
	cfgJSON, _ := json.Marshal(sampleWorkedExampleConfig())
	redacted, err := contenttools.RedactSensitiveConfig(m.ConfigSchema, cfgJSON)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(redacted, &cfg); err != nil {
		t.Fatal(err)
	}
	steps, _ := cfg["steps"].([]any)
	if len(steps) == 0 {
		t.Fatal("no steps")
	}
	s0 := steps[0].(map[string]any)
	if _, ok := s0["hints"]; ok {
		t.Fatal("hints must be redacted")
	}
	if _, ok := s0["explanation"]; ok {
		t.Fatal("explanation must be redacted")
	}
	blank, _ := s0["blank"].(map[string]any)
	if blank == nil {
		t.Fatal("blank should remain (type visible)")
	}
	if _, ok := blank["expected"]; ok {
		t.Fatal("expected must be redacted")
	}
	if blank["type"] != "expression" {
		t.Fatalf("type should remain: %#v", blank["type"])
	}
}

func TestWorkedExampleSequentialLock(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(worked_example.ID)
	cfg := sampleWorkedExampleConfig()
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(worked_example.EmptyState())
	enroll := uuid.New()

	in, _ := json.Marshal(map[string]any{"stepId": "s2", "value": "3x+6"})
	res, err := contenttools.DispatchAction(m, "check_step", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    stJSON,
		Input:        in,
		EnrollmentID: enroll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "sequential_locked" {
		t.Fatalf("want sequential_locked: %#v", res.Result)
	}
}

func TestWorkedExampleVerify(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(worked_example.ID)
	cfgJSON, _ := json.Marshal(sampleWorkedExampleConfig())
	res, err := contenttools.DispatchAction(m, "verify", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["ok"] != true {
		t.Fatalf("verify failed: %#v", res.Result)
	}
}
