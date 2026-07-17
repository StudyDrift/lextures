package transcriptdelivery

import (
	"testing"

	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func TestReleaseGuardNilOrder(t *testing.T) {
	t.Parallel()
	res, err := ReleaseGuard(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Reason == "" {
		t.Fatalf("expected deny, got %#v", res)
	}
}

func TestGuardResultHoldFields(t *testing.T) {
	t.Parallel()
	r := GuardResult{OK: false, Reason: "active hold blocks delivery", OnHold: true}
	if r.OK || !r.OnHold {
		t.Fatalf("unexpected %#v", r)
	}
	if transcriptsrepo.ErrReleaseGuardDenied == nil {
		t.Fatal("expected ErrReleaseGuardDenied")
	}
}
