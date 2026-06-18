package canvassubmissionsyncqueue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryBusPublishConsume(t *testing.T) {
	bus, err := NewBus("", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bus.Close() }()

	jobID := uuid.New()
	userID := uuid.New()
	itemID := uuid.New()
	submissionID := uuid.New()
	msg := QueueMessage{
		JobID:        jobID,
		UserID:       userID,
		CourseCode:   "demo-101",
		ItemID:       itemID,
		SubmissionID: submissionID,
		AccessToken:  "token",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan struct{})
	done := make(chan QueueMessage, 1)
	go func() {
		close(ready)
		_ = bus.Consume(ctx, func(got QueueMessage) error {
			done <- got
			cancel()
			return nil
		})
	}()
	<-ready

	if err := bus.Publish(ctx, msg); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-done:
		if got.JobID != jobID || got.CourseCode != "demo-101" {
			t.Fatalf("unexpected message: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected consumed message")
	}
}