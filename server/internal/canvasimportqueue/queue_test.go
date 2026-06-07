package canvasimportqueue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
)

func TestMemoryBusPublishConsume(t *testing.T) {
	bus, err := NewBus("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bus.Close() }()

	jobID := uuid.New()
	userID := uuid.New()
	msg := canvasimportjobs.QueueMessage{
		JobID:          jobID,
		UserID:         userID,
		CourseCode:     "demo-101",
		Mode:           "erase",
		CanvasBaseURL:  "https://school.instructure.com",
		CanvasCourseID: "42",
		AccessToken:    "token",
		Include:        canvasimportjobs.Include{Modules: true},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan struct{})
	done := make(chan canvasimportjobs.QueueMessage, 1)
	go func() {
		close(ready)
		_ = bus.Consume(ctx, func(got canvasimportjobs.QueueMessage) error {
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

func TestNewBusDefaultQueueName(t *testing.T) {
	bus, err := NewBus("", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = bus.Close()
}
