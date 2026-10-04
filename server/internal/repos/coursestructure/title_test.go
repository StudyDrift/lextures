package coursestructure

import "testing"

func TestIsAPIErrorPayloadTitle(t *testing.T) {
	const reorder = `{"error":{"code":"INTERNAL","message":"Failed to reorder course structure."}}`
	if !IsAPIErrorPayloadTitle(reorder) {
		t.Fatal("expected reorder error body to be rejected")
	}
	if !IsAPIErrorPayloadTitle("\n" + reorder + "\n") {
		t.Fatal("expected surrounding whitespace to still match")
	}
	if !IsAPIErrorPayloadTitle("{\"error\":{\"message\":\"Nope.\",\"code\":\"INVALID_INPUT\"}}") {
		t.Fatal("expected any error envelope to be rejected")
	}
	for _, ok := range []string{
		"What Makes a Strong AI-Augmented Workflow",
		"Failed to reorder course structure.",
		"{not json",
		`{"error":"plain"}`,
		`{"title":"Week 1"}`,
		"",
	} {
		if IsAPIErrorPayloadTitle(ok) {
			t.Fatalf("title %q should be allowed", ok)
		}
	}
	if err := ValidateItemTitle(reorder); err != ErrAPIErrorPayloadTitle {
		t.Fatalf("ValidateItemTitle = %v", err)
	}
	if err := ValidateItemTitle("Week 1 overview"); err != nil {
		t.Fatalf("plain title: %v", err)
	}
}
