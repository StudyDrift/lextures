// Package canvassubmissionsyncqueue publishes and consumes Canvas submission sync jobs via
// RabbitMQ, AWS SQS, or an in-memory fallback.
package canvassubmissionsyncqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lextures/lextures/server/internal/mq"
)

const defaultQueueName = "canvas.submission.sync"

// Publisher enqueues Canvas submission sync jobs.
type Publisher interface {
	Publish(ctx context.Context, msg QueueMessage) error
	Close() error
}

// Consumer receives Canvas submission sync jobs.
type Consumer interface {
	Consume(ctx context.Context, handler func(QueueMessage) error) error
	Close() error
}

// Bus combines publish and consume for wiring in app startup.
type Bus struct {
	pub         Publisher
	con         Consumer
	concurrency int
}

// NewBus returns a durable bus when url is non-empty, otherwise an in-process memory bus.
// url is either an AMQP URL (RabbitMQ) or a full AWS SQS queue URL.
func NewBus(url, queueName string, concurrency int) (*Bus, error) {
	if queueName == "" {
		queueName = defaultQueueName
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if url == "" {
		mem := newMemoryBus(concurrency)
		return &Bus{pub: mem, con: mem, concurrency: concurrency}, nil
	}
	tr, err := mq.Open(url, queueName, concurrency)
	if err != nil {
		return nil, err
	}
	adapter := &transportAdapter{tr: tr, concurrency: concurrency}
	return &Bus{pub: adapter, con: adapter, concurrency: concurrency}, nil
}

// Concurrency returns how many sync jobs are processed in parallel.
func (b *Bus) Concurrency() int {
	if b == nil || b.concurrency < 1 {
		return 1
	}
	return b.concurrency
}

func (b *Bus) Publish(ctx context.Context, msg QueueMessage) error {
	if b == nil || b.pub == nil {
		return fmt.Errorf("canvas submission sync queue not configured")
	}
	return b.pub.Publish(ctx, msg)
}

func (b *Bus) Consume(ctx context.Context, handler func(QueueMessage) error) error {
	if b == nil || b.con == nil {
		return fmt.Errorf("canvas submission sync queue not configured")
	}
	return b.con.Consume(ctx, handler)
}

func (b *Bus) Close() error {
	if b == nil {
		return nil
	}
	if b.pub != nil {
		return b.pub.Close()
	}
	return nil
}

type transportAdapter struct {
	tr          mq.Transport
	concurrency int
}

func (a *transportAdapter) Publish(ctx context.Context, msg QueueMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return a.tr.Publish(ctx, body)
}

func (a *transportAdapter) Consume(ctx context.Context, handler func(QueueMessage) error) error {
	return a.tr.Consume(ctx, a.concurrency, func(ctx context.Context, body []byte) error {
		var msg QueueMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			slog.Warn("canvas_submission_sync_queue: bad message", "err", err)
			return fmt.Errorf("%w: %v", mq.ErrPoison, err)
		}
		if err := handler(msg); err != nil {
			slog.Warn("canvas_submission_sync_queue: handler failed", "job_id", msg.JobID, "err", err)
			return err
		}
		return nil
	})
}

func (a *transportAdapter) Close() error {
	return a.tr.Close()
}

type memoryBus struct {
	mu          sync.Mutex
	ch          chan QueueMessage
	closed      bool
	closeCh     chan struct{}
	concurrency int
}

func newMemoryBus(concurrency int) *memoryBus {
	return &memoryBus{
		ch:          make(chan QueueMessage, 64),
		closeCh:     make(chan struct{}),
		concurrency: concurrency,
	}
}

func (m *memoryBus) Publish(_ context.Context, msg QueueMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return fmt.Errorf("memory queue closed")
	}
	select {
	case m.ch <- msg:
		return nil
	default:
		return fmt.Errorf("memory queue full")
	}
}

func (m *memoryBus) Consume(ctx context.Context, handler func(QueueMessage) error) error {
	sem := make(chan struct{}, m.concurrency)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.closeCh:
			return nil
		case msg := <-m.ch:
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			wg.Add(1)
			go func(msg QueueMessage) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if err := handler(msg); err != nil {
					slog.Warn("canvas_submission_sync_queue: memory handler failed", "job_id", msg.JobID, "err", err)
				}
			}(msg)
		}
	}
}

func (m *memoryBus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.closeCh)
	return nil
}
