package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitTransport struct {
	queueName   string
	concurrency int
	conn        *amqp.Connection
	ch          *amqp.Channel
}

func openRabbit(url, queueName string, concurrency int) (*rabbitTransport, error) {
	if queueName == "" {
		return nil, fmt.Errorf("mq/rabbit: queue name is required")
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("mq/rabbit dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("mq/rabbit channel: %w", err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("mq/rabbit queue declare: %w", err)
	}
	return &rabbitTransport{queueName: queueName, concurrency: concurrency, conn: conn, ch: ch}, nil
}

func (r *rabbitTransport) Publish(ctx context.Context, body []byte) error {
	return r.ch.PublishWithContext(ctx, "", r.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (r *rabbitTransport) Consume(ctx context.Context, concurrency int, handler func(context.Context, []byte) error) error {
	if concurrency < 1 {
		concurrency = r.concurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if err := r.ch.Qos(concurrency, 0, false); err != nil {
		return fmt.Errorf("mq/rabbit qos: %w", err)
	}
	deliveries, err := r.ch.Consume(r.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("mq/rabbit consume: %w", err)
	}
	sem := make(chan struct{}, concurrency)
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
				if err := handler(ctx, d.Body); err != nil {
					if errors.Is(err, ErrPoison) {
						slog.Warn("mq/rabbit: poison message discarded", "err", err)
						_ = d.Nack(false, false)
						return
					}
					slog.Warn("mq/rabbit: handler failed, requeue", "err", err)
					_ = d.Nack(false, true)
					return
				}
				_ = d.Ack(false)
			}(d)
		}
	}
}

func (r *rabbitTransport) Close() error {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
