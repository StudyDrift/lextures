package gradingagent

import "testing"

func TestRunStatusCancelled_isRecognized(t *testing.T) {
	if RunStatusCancelled != "cancelled" {
		t.Fatalf("RunStatusCancelled = %q", RunStatusCancelled)
	}
}
