package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type sqsTransport struct {
	client      *sqs.Client
	queueURL    string
	concurrency int
}

func openSQS(queueURL string, concurrency int) (*sqsTransport, error) {
	region := regionFromSQSURL(queueURL)
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	}
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		return nil, fmt.Errorf("mq/sqs: cannot determine region from URL or AWS_REGION")
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("mq/sqs: load aws config: %w", err)
	}
	return &sqsTransport{
		client:      sqs.NewFromConfig(cfg),
		queueURL:    queueURL,
		concurrency: concurrency,
	}, nil
}

// regionFromSQSURL extracts the region from
// https://sqs.<region>.amazonaws.com/... or https://sqs.<region>.amazonaws.com.cn/...
func regionFromSQSURL(queueURL string) string {
	// Host form: sqs.us-east-1.amazonaws.com
	const prefix = "https://sqs."
	if !strings.HasPrefix(strings.ToLower(queueURL), prefix) {
		return ""
	}
	rest := queueURL[len(prefix):]
	dot := strings.IndexByte(rest, '.')
	if dot <= 0 {
		return ""
	}
	return rest[:dot]
}

func (s *sqsTransport) Publish(ctx context.Context, body []byte) error {
	_, err := s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("mq/sqs send: %w", err)
	}
	return nil
}

func (s *sqsTransport) Consume(ctx context.Context, concurrency int, handler func(context.Context, []byte) error) error {
	if concurrency < 1 {
		concurrency = s.concurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20, // long poll
			VisibilityTimeout:   900,
		})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("mq/sqs: receive failed", "err", err)
			continue
		}
		if len(out.Messages) == 0 {
			continue
		}

		for _, msg := range out.Messages {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			wg.Add(1)
			go func(m sqstypes.Message) {
				defer func() {
					<-sem
					wg.Done()
				}()
				s.handleOne(ctx, m, handler)
			}(msg)
		}
	}
}

func (s *sqsTransport) handleOne(ctx context.Context, m sqstypes.Message, handler func(context.Context, []byte) error) {
	body := ""
	if m.Body != nil {
		body = *m.Body
	}
	err := handler(ctx, []byte(body))
	if err != nil {
		if errors.Is(err, ErrPoison) {
			slog.Warn("mq/sqs: poison message deleted", "err", err, "message_id", aws.ToString(m.MessageId))
			s.delete(ctx, m)
			return
		}
		// Leave message for visibility timeout / redrive to DLQ after maxReceiveCount.
		slog.Warn("mq/sqs: handler failed, will retry after visibility timeout",
			"err", err, "message_id", aws.ToString(m.MessageId))
		return
	}
	s.delete(ctx, m)
}

func (s *sqsTransport) delete(ctx context.Context, m sqstypes.Message) {
	if m.ReceiptHandle == nil {
		return
	}
	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.queueURL),
		ReceiptHandle: m.ReceiptHandle,
	})
	if err != nil {
		slog.Warn("mq/sqs: delete failed", "err", err, "message_id", aws.ToString(m.MessageId))
	}
}

func (s *sqsTransport) Close() error {
	return nil
}
