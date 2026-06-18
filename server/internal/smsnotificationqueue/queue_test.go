package smsnotificationqueue

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

	userID := uuid.New()
	msg := QueueMessage{
		UserID:    userID,
		EventType: "grade_posted",
		Phone:     "+15551234567",
		Title:     "Grade posted",
		Body:      "Your grade has been posted.",
		ActionURL: "https://app.example/courses/demo/grades",
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
		if got.UserID != userID || got.EventType != "grade_posted" {
			t.Fatalf("unexpected message: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected consumed message")
	}
}