// Package mq provides a small publish/consume transport for background job
// buses. Supported backends:
//
//   - RabbitMQ when the URL is amqp:// or amqps://
//   - AWS SQS when the URL is an https://sqs…amazonaws.com/… queue URL
//
// Callers keep typed JSON messages; this package only moves raw bytes.
// Empty URL is not handled here — queue packages fall back to in-process memory.
package mq

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrPoison signals a non-retryable message (bad payload). Consumers delete /
// nack-without-requeue instead of returning the message to the queue.
var ErrPoison = errors.New("mq: poison message")

// Transport is a durable byte bus.
type Transport interface {
	Publish(ctx context.Context, body []byte) error
	// Consume blocks until ctx is cancelled. concurrency is the max number of
	// in-flight handler invocations. Handler errors requeue (or leave for SQS
	// visibility timeout) unless they wrap ErrPoison.
	Consume(ctx context.Context, concurrency int, handler func(context.Context, []byte) error) error
	Close() error
}

// Open returns a RabbitMQ or SQS transport based on url.
// queueName is only used for RabbitMQ (SQS URLs already identify the queue).
func Open(url, queueName string, concurrency int) (Transport, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("mq: empty url")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if IsSQSURL(url) {
		return openSQS(url, concurrency)
	}
	if isAMQPURL(url) {
		return openRabbit(url, queueName, concurrency)
	}
	return nil, fmt.Errorf("mq: unrecognized queue url scheme (want amqp(s):// or https://sqs…): %q", redactURL(url))
}

// IsSQSURL reports whether url looks like an AWS SQS queue URL.
func IsSQSURL(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	if !strings.HasPrefix(u, "https://") {
		return false
	}
	// Standard: https://sqs.<region>.amazonaws.com/<account>/<name>
	// VPC endpoint / china variants still contain "sqs" + "amazonaws".
	return strings.Contains(u, "sqs") && strings.Contains(u, "amazonaws")
}

func isAMQPURL(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	return strings.HasPrefix(u, "amqp://") || strings.HasPrefix(u, "amqps://")
}

func redactURL(url string) string {
	if len(url) > 48 {
		return url[:48] + "…"
	}
	return url
}
