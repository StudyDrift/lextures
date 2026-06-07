package demographics

import "testing"

func TestApplySuppression(t *testing.T) {
	t.Parallel()
	suppressed, out := ApplySuppression(9, 72.5)
	if !suppressed || out != nil {
		t.Fatalf("n=9 should suppress, got suppressed=%v out=%v", suppressed, out)
	}
	suppressed, out = ApplySuppression(10, 72.5)
	if suppressed || out == nil || *out != 72.5 {
		t.Fatalf("n=10 should expose 72.5, got suppressed=%v out=%v", suppressed, out)
	}
}
