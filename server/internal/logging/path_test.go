package logging

import "testing"

func TestRedactRequestPath_UserUUID(t *testing.T) {
	t.Parallel()
	in := "/api/v1/users/550e8400-e29b-41d4-a716-446655440000/profile"
	got := RedactRequestPath(in)
	want := "/api/v1/users/[REDACTED:user_id]/profile"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRedactRequestPath_PassThroughNoUUID(t *testing.T) {
	t.Parallel()
	in := "/api/v1/health"
	if got := RedactRequestPath(in); got != in {
		t.Fatalf("got %q", got)
	}
}
