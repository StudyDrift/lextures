package transcriptnotify

import (
	"testing"

	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func TestEventForOrderStatus(t *testing.T) {
	cases := []struct {
		status transcriptsrepo.OrderStatus
		want   LifecycleEvent
		ok     bool
	}{
		{transcriptsrepo.OrderInReview, EventSubmitted, true},
		{transcriptsrepo.OrderOnHold, EventOnHold, true},
		{transcriptsrepo.OrderPendingConsent, EventConsentNeeded, true},
		{transcriptsrepo.OrderPendingPayment, EventPaymentNeeded, true},
		{transcriptsrepo.OrderProcessing, EventApproved, true},
		{transcriptsrepo.OrderRejected, EventRejected, true},
		{transcriptsrepo.OrderCanceled, EventCanceled, true},
		{transcriptsrepo.OrderCompleted, "", false},
		{transcriptsrepo.OrderDraft, "", false},
	}
	for _, tc := range cases {
		got, ok := EventForOrderStatus(tc.status)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("status %s: got (%s,%v) want (%s,%v)", tc.status, got, ok, tc.want, tc.ok)
		}
	}
}

func TestMappingTableDriven(t *testing.T) {
	checks := []struct {
		ev            LifecycleEvent
		transactional bool
		learner       bool
		guardian      bool
		registrar     bool
	}{
		{EventSubmitted, false, true, true, false},
		{EventConsentNeeded, true, true, true, false},
		{EventPaymentNeeded, true, true, true, false},
		{EventOpened, false, true, true, false},
		{EventExceptionFail, false, false, false, true},
		{EventExceptionHold, false, false, false, true},
	}
	for _, tc := range checks {
		_, title, tx, learner, guardian, registrar, ok := MappingForTest(tc.ev)
		if !ok {
			t.Fatalf("missing mapping for %s", tc.ev)
		}
		if title == "" {
			t.Fatalf("%s: empty title", tc.ev)
		}
		if tx != tc.transactional || learner != tc.learner || guardian != tc.guardian || registrar != tc.registrar {
			t.Fatalf("%s: got tx=%v learner=%v guardian=%v registrar=%v",
				tc.ev, tx, learner, guardian, registrar)
		}
	}
}

func TestLearnerCancelAllowed(t *testing.T) {
	o := &transcriptsrepo.Order{Status: transcriptsrepo.OrderInReview}
	if !transcriptsrepo.LearnerCancelAllowed(o) {
		t.Fatal("in_review should allow cancel")
	}
	o.Status = transcriptsrepo.OrderCompleted
	if transcriptsrepo.LearnerCancelAllowed(o) {
		t.Fatal("completed should not allow cancel")
	}
	o.Status = transcriptsrepo.OrderProcessing
	o.Items = []transcriptsrepo.OrderItem{{Status: transcriptsrepo.ItemReady}}
	if !transcriptsrepo.LearnerCancelAllowed(o) {
		t.Fatal("processing with ready items should allow cancel")
	}
	o.Items = []transcriptsrepo.OrderItem{{Status: transcriptsrepo.ItemDelivered}}
	if transcriptsrepo.LearnerCancelAllowed(o) {
		t.Fatal("processing with delivered item should not allow cancel")
	}
}
