package transcriptorder

import "testing"

func TestStudentFacingMessage_NeverLeaksReason(t *testing.T) {
	msg := StudentFacingMessage(HoldFinancial, nil)
	if msg == "" || msg == "owes $500 tuition" {
		t.Fatalf("unexpected message %q", msg)
	}
	custom := "Please contact the bursar."
	got := StudentFacingMessage(HoldFinancial, &custom)
	if got != custom {
		t.Fatalf("got %q", got)
	}
}
