// Package gradingagentqueue publishes and consumes grading-agent batch jobs via RabbitMQ or an in-memory fallback.
package gradingagentqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const defaultQueueName = "grading.agent.run"

// Publisher enqueues grading-agent jobs.
type Publisher interface {
	Publish(ctx context.Context, msg QueueMessage) error
	Close() error
}

// Consumer receives grading-agent jobs.
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

// NewBus returns a RabbitMQ-backed bus when url is non-empty, otherwise an in-process memory bus.
func NewBus(url, queueName string, concurrency int) (*Bus, error) {
	if queueName == "" {
		queueName = defaultQueueName
	}
	if concurrency < 1 {
		concurrency = 2
	}
	if url == "" {
		mem := newMemoryBus(concurrency)
		return &Bus{pub: mem, con: mem, concurrency: concurrency}, nil
	}
	rmq, err := newRabbitBus(url, queueName, concurrency)
	if err != nil {
		return nil, err
	}
	return &Bus{pub: rmq, con: rmq, concurrency: concurrency}, nil
}

// Concurrency returns how many jobs are processed in parallel.
func (b *Bus) Concurrency() int {
	if b == nil || b.concurrency < 1 {
		return 1
	}
	return b.concurrency
}

func (b *Bus) Publish(ctx context.Context, msg QueueMessage) error {
	if b == nil || b.pub == nil {
		return fmt.Errorf("grading agent queue not configured")
	}
	return b.pub.Publish(ctx, msg)
}

func (b *Bus) Consume(ctx context.Context, handler func(QueueMessage) error) error {
	if b == nil || b.con == nil {
		return fmt.Errorf("grading agent queue not configured")
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
	queueName   string
	concurrency int
	conn        *amqp.Connection
	ch          *amqp.Channel
}

func newRabbitBus(url, queueName string, concurrency int) (*rabbitBus, error) {
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
	return &rabbitBus{queueName: queueName, concurrency: concurrency, conn: conn, ch: ch}, nil
}

func (r *rabbitBus) Publish(_ context.Context, msg QueueMessage) error {
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

func (r *rabbitBus) Consume(ctx context.Context, handler func(QueueMessage) error) error {
	if err := r.ch.Qos(r.concurrency, 0, false); err != nil {
		return fmt.Errorf("rabbitmq qos: %w", err)
	}
	deliveries, err := r.ch.Consume(r.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}
	sem := make(chan struct{}, r.concurrency)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				_ = d.Nack(false, true)
				return ctx.Err()
			}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer func() {
					<-sem
					wg.Done()
				}()
				var msg QueueMessage
				if err := json.Unmarshal(d.Body, &msg); err != nil {
					slog.Warn("grading_agent_queue: bad message", "err", err)
					_ = d.Nack(false, false)
					return
				}
				if err := handler(msg); err != nil {
					slog.Warn("grading_agent_queue: handler failed", "run_id", msg.RunID, "err", err)
					_ = d.Nack(false, true)
					return
				}
				_ = d.Ack(false)
			}(d)
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
	mu          sync.Mutex
	ch          chan QueueMessage
	closed      bool
	closeCh     chan struct{}
	concurrency int
}

func newMemoryBus(concurrency int) *memoryBus {
	return &memoryBus{
		ch:          make(chan QueueMessage, 128),
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
					slog.Warn("grading_agent_queue: memory handler failed", "run_id", msg.RunID, "err", err)
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