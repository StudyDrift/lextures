package canvasimportevents

import (
	"testing"

	"github.com/google/uuid"
)

func TestHubBroadcastSubscribe(t *testing.T) {
	h := New()
	jobID := uuid.New()
	ch, unsub := h.Subscribe(jobID)
	defer unsub()

	h.Broadcast(jobID, Message{Type: "progress", Message: "hello"})
	got := <-ch
	if got.Type != "progress" || got.Message != "hello" {
		t.Fatalf("got %+v", got)
	}
}

func TestHubIgnoresOtherJobs(t *testing.T) {
	h := New()
	jobA := uuid.New()
	jobB := uuid.New()
	ch, unsub := h.Subscribe(jobA)
	defer unsub()

	h.Broadcast(jobB, Message{Type: "progress", Message: "other"})
	select {
	case <-ch:
		t.Fatal("unexpected message for job A")
	default:
	}
}
