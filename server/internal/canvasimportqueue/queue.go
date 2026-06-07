// Package canvasimportqueue publishes and consumes Canvas import jobs via RabbitMQ or an in-memory fallback.
package canvasimportqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
)

const defaultQueueName = "canvas.course.import"

// Publisher enqueues Canvas import jobs.
type Publisher interface {
	Publish(ctx context.Context, msg canvasimportjobs.QueueMessage) error
	Close() error
}

// Consumer receives Canvas import jobs.
type Consumer interface {
	Consume(ctx context.Context, handler func(canvasimportjobs.QueueMessage) error) error
	Close() error
}

// Bus combines publish and consume for wiring in app startup.
type Bus struct {
	pub Publisher
	con Consumer
}

// NewBus returns a RabbitMQ-backed bus when url is non-empty, otherwise an in-process memory bus.
func NewBus(url, queueName string) (*Bus, error) {
	if queueName == "" {
		queueName = defaultQueueName
	}
	if url == "" {
		mem := newMemoryBus()
		return &Bus{pub: mem, con: mem}, nil
	}
	rmq, err := newRabbitBus(url, queueName)
	if err != nil {
		return nil, err
	}
	return &Bus{pub: rmq, con: rmq}, nil
}

func (b *Bus) Publish(ctx context.Context, msg canvasimportjobs.QueueMessage) error {
	if b == nil || b.pub == nil {
		return fmt.Errorf("canvas import queue not configured")
	}
	return b.pub.Publish(ctx, msg)
}

func (b *Bus) Consume(ctx context.Context, handler func(canvasimportjobs.QueueMessage) error) error {
	if b == nil || b.con == nil {
		return fmt.Errorf("canvas import queue not configured")
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

type rabbitBus struct {
	url       string
	queueName string
	conn      *amqp.Connection
	ch        *amqp.Channel
}

func newRabbitBus(url, queueName string) (*rabbitBus, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq queue declare: %w", err)
	}
	return &rabbitBus{url: url, queueName: queueName, conn: conn, ch: ch}, nil
}

func (r *rabbitBus) Publish(_ context.Context, msg canvasimportjobs.QueueMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.ch.PublishWithContext(context.Background(), "", r.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (r *rabbitBus) Consume(ctx context.Context, handler func(canvasimportjobs.QueueMessage) error) error {
	deliveries, err := r.ch.Consume(r.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			var msg canvasimportjobs.QueueMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				slog.Warn("canvas_import_queue: bad message", "err", err)
				_ = d.Nack(false, false)
				continue
			}
			if err := handler(msg); err != nil {
				slog.Warn("canvas_import_queue: handler failed", "job_id", msg.JobID, "err", err)
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

func (r *rabbitBus) Close() error {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

type memoryBus struct {
	mu      sync.Mutex
	ch      chan canvasimportjobs.QueueMessage
	closed  bool
	closeCh chan struct{}
}

func newMemoryBus() *memoryBus {
	return &memoryBus{
		ch:      make(chan canvasimportjobs.QueueMessage, 32),
		closeCh: make(chan struct{}),
	}
}

func (m *memoryBus) Publish(_ context.Context, msg canvasimportjobs.QueueMessage) error {
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

func (m *memoryBus) Consume(ctx context.Context, handler func(canvasimportjobs.QueueMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.closeCh:
			return nil
		case msg := <-m.ch:
			if err := handler(msg); err != nil {
				slog.Warn("canvas_import_queue: memory handler failed", "job_id", msg.JobID, "err", err)
			}
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
