package parameter_explorer

import (
	"math"
	"testing"
)

func TestEvalExpression_Basic(t *testing.T) {
	v, err := EvalExpression("2 + 3 * 4", nil)
	if err != nil {
		t.Fatal(err)
	}
	if v != 14 {
		t.Fatalf("got %v want 14", v)
	}
}

func TestEvalExpression_Params(t *testing.T) {
	v, err := EvalExpression("a * x^2 + b * x + c", map[string]float64{
		"a": 1, "b": 0, "c": 0, "x": 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if v != 9 {
		t.Fatalf("got %v want 9", v)
	}
}

func TestEvalExpression_RejectUnknownFn(t *testing.T) {
	_, err := EvalExpression("eval(1)", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	ee, ok := err.(*EvalError)
	if !ok || ee.Code != "unknown_fn" {
		t.Fatalf("got %#v", err)
	}
}

func TestEvalExpression_RejectPropertyAccess(t *testing.T) {
	_, err := EvalExpression("a.b", map[string]float64{"a": 1})
	if err == nil {
		t.Fatal("expected error for property access")
	}
}

func TestEvalPredicate(t *testing.T) {
	ok, err := EvalPredicate("r > 3.5", map[string]float64{"r": 4})
	if err != nil || !ok {
		t.Fatalf("want true, got %v %v", ok, err)
	}
	ok, err = EvalPredicate("r > 3.5", map[string]float64{"r": 3})
	if err != nil || ok {
		t.Fatalf("want false, got %v %v", ok, err)
	}
}

func TestEvalExpression_ExponentCap(t *testing.T) {
	_, err := EvalExpression("2^10000", nil)
	if err == nil {
		t.Fatal("expected exponent error")
	}
}

func TestPresets_QuadraticReference(t *testing.T) {
	def, ok := LookupPreset("quadratic")
	if !ok {
		t.Fatal("missing quadratic")
	}
	vars := ResolveBoundVars(map[string]string{"a": "a", "b": "b", "c": "c"}, map[string]float64{
		"a": 1, "b": -2, "c": 1,
	}, 1)
	v, err := EvalExpression(def.Expression, vars)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v-0) > 1e-9 {
		t.Fatalf("at x=1, (x-1)^2 should be 0, got %v", v)
	}
}

func TestPresets_LogisticReference(t *testing.T) {
	def, ok := LookupPreset("logistic")
	if !ok {
		t.Fatal("missing logistic")
	}
	vars := ResolveBoundVars(map[string]string{"K": "K", "P0": "P0", "r": "r"}, map[string]float64{
		"K": 100, "P0": 10, "r": 0.3,
	}, 0)
	v, err := EvalExpression(def.Expression, vars)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v-10) > 1e-6 {
		t.Fatalf("at t=0 want P0=10, got %v", v)
	}
}

func TestValidateExpression_Adversarial(t *testing.T) {
	// Tokenize / parse rejects
	for _, e := range []string{"while(1){}", "function(){}", "a[0]", ""} {
		if err := ValidateExpression(e); err == nil {
			t.Fatalf("expected reject for %q", e)
		}
	}
	// Bare identifiers parse but are unknown at eval (no host object access).
	for _, e := range []string{"constructor", "__proto__"} {
		if err := ValidateExpression(e); err != nil {
			t.Fatalf("bare ident should parse: %q: %v", e, err)
		}
		_, err := EvalExpression(e, nil)
		if err == nil {
			t.Fatalf("%q should be unknown var at eval", e)
		}
	}
}

func TestAppendTrace_Downsample(t *testing.T) {
	st := EmptyState()
	for i := 0; i < 300; i++ {
		st = AppendTrace(st, map[string]any{"a": float64(i)}, "t")
	}
	if len(st.Trace) > MaxTraceEntries {
		t.Fatalf("trace len %d > cap", len(st.Trace))
	}
}

func TestIsComplete(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NoticingPrompts = []NoticingPrompt{
		{ID: "p1", Text: "What happened?", Kind: "text", Required: true, UnlockWhen: "a > 1"},
	}
	st := EmptyState()
	st.Params = DefaultParams(cfg)
	if IsComplete(cfg, st) {
		t.Fatal("should not complete without answer/checkpoint")
	}
	st.Checkpoints["p1"] = NowRFC3339()
	st.Answers["p1"] = "it steepened"
	if !IsComplete(cfg, st) {
		t.Fatal("expected complete")
	}
}

func TestValidateConfig_BadExpression(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Model = Model{
		Kind:       "expression",
		Expression: "foo(bar)",
		Sweep:      &SweepSpec{ParamID: "x", From: -1, To: 1, Points: 20},
	}
	if err := ValidateConfigForAuthoring(cfg); err == nil {
		t.Fatal("expected reject unknown fn")
	}
}
