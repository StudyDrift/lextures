package mq

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsSQSURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		want bool
	}{
		{"https://sqs.us-east-1.amazonaws.com/123456789012/lextures-staging-canvas-course-import", true},
		{"https://sqs.eu-west-1.amazonaws.com/1/q", true},
		{"amqp://user:pass@localhost:5672/", false},
		{"amqps://mq.example.com:5671", false},
		{"", false},
		{"https://example.com/sqs", false},
	}
	for _, tc := range cases {
		if got := IsSQSURL(tc.url); got != tc.want {
			t.Errorf("IsSQSURL(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestRegionFromSQSURL(t *testing.T) {
	t.Parallel()
	got := regionFromSQSURL("https://sqs.us-west-2.amazonaws.com/123/my-queue")
	if got != "us-west-2" {
		t.Fatalf("region = %q, want us-west-2", got)
	}
	if regionFromSQSURL("amqp://localhost") != "" {
		t.Fatal("expected empty region for non-sqs url")
	}
}

func TestErrorsIsPoison(t *testing.T) {
	t.Parallel()
	if !errors.Is(ErrPoison, ErrPoison) {
		t.Fatal("expected ErrPoison")
	}
	if !errors.Is(fmt.Errorf("wrap: %w", ErrPoison), ErrPoison) {
		t.Fatal("expected wrapped poison")
	}
	if errors.Is(errors.New("retry me"), ErrPoison) {
		t.Fatal("expected non-poison")
	}
}
