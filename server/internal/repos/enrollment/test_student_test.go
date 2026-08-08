package enrollment

import "testing"

func TestTestStudentDisplayName(t *testing.T) {
	t.Parallel()
	if got := TestStudentDisplayName(RoleTestStudent, "Ada Instructor"); got != DisplayNameTestStudent {
		t.Fatalf("test_student label: got %q want %q", got, DisplayNameTestStudent)
	}
	if got := TestStudentDisplayName("student", "Real Learner"); got != "Real Learner" {
		t.Fatalf("student label: got %q want Real Learner", got)
	}
	if !IsTestStudentRole(RoleTestStudent) || IsTestStudentRole("student") {
		t.Fatal("IsTestStudentRole mismatch")
	}
}
