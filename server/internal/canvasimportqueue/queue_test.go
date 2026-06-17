package canvasimportqueue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
)

func TestMemoryBusPublishConsume(t *testing.T) {
	bus, err := NewBus("", "", 1)
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
	bus, err := NewBus("", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	_ = bus.Close()
}

func TestMemoryBusConcurrentConsume(t *testing.T) {
	bus, err := NewBus("", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bus.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 3)
	release := make(chan struct{})
	var active int
	var peak int
	var mu sync.Mutex

	ready := make(chan struct{})
	go func() {
		close(ready)
		_ = bus.Consume(ctx, func(_ canvasimportjobs.QueueMessage) error {
			mu.Lock()
			active++
			if active > peak {
				peak = active
			}
			mu.Unlock()
			started <- struct{}{}
			<-release
			mu.Lock()
			active--
			mu.Unlock()
			return nil
		})
	}()
	<-ready

	for i := 0; i < 3; i++ {
		msg := canvasimportjobs.QueueMessage{JobID: uuid.New(), CourseCode: fmt.Sprintf("course-%d", i)}
		if err := bus.Publish(ctx, msg); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected job %d to start", i)
		}
	}
	if peak < 3 {
		t.Fatalf("expected 3 concurrent handlers, peak was %d", peak)
	}
	close(release)
}
