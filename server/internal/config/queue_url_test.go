package config

import "testing"

func TestMessageQueueURL_RabbitDefault(t *testing.T) {
	t.Parallel()
	c := Config{RabbitMQURL: "amqp://localhost:5672/"}
	if got := c.MessageQueueURL("https://sqs.us-east-1.amazonaws.com/1/q"); got != "amqp://localhost:5672/" {
		t.Fatalf("got %q", got)
	}
}

func TestMessageQueueURL_SQSBackend(t *testing.T) {
	t.Parallel()
	sqs := "https://sqs.us-east-1.amazonaws.com/123/lextures-staging-canvas-course-import"
	c := Config{
		QueueBackend:       "sqs",
		RabbitMQURL:        "amqp://localhost:5672/",
		SQSCanvasImportURL: sqs,
	}
	if got := c.CanvasImportQueueURL(); got != sqs {
		t.Fatalf("got %q want %q", got, sqs)
	}
}

func TestMessageQueueURL_AutoDetectSQS(t *testing.T) {
	t.Parallel()
	sqs := "https://sqs.us-east-1.amazonaws.com/123/q"
	c := Config{SQSCanvasImportURL: sqs}
	if c.resolvedQueueBackend() != "sqs" {
		t.Fatalf("backend = %q", c.resolvedQueueBackend())
	}
	if got := c.CanvasImportQueueURL(); got != sqs {
		t.Fatalf("got %q", got)
	}
}

func TestMessageQueueURL_Memory(t *testing.T) {
	t.Parallel()
	c := Config{QueueBackend: "memory", RabbitMQURL: "amqp://x"}
	if got := c.MessageQueueURL("https://sqs.example"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}
