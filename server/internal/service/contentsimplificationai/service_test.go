package contentsimplificationai

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

type mockCompleter struct {
	fn func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error)
}

func (m mockCompleter) Complete(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	return m.fn(ctx, model, messages, opts...)
}

func TestSimplify(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		return aiprovider.ChatResult{Text: "  Simplified text.  "}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	text, meta, err := Simplify(context.Background(), ai, "test-model", "Some complex text.", 5)
	if err != nil {
		t.Fatal(err)
	}
	if text != "Simplified text." {
		t.Fatalf("unexpected text: %q", text)
	}
	if meta.ModelID != "test-model" {
		t.Fatalf("expected model id propagated, got %q", meta.ModelID)
	}
}

func TestSimplify_NilCompleter(t *testing.T) {
	if _, _, err := Simplify(context.Background(), nil, "test-model", "text", 5); err == nil {
		t.Fatal("expected error for nil completer")
	}
}

func TestSimplify_EmptyText(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call Complete for empty text")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, err := Simplify(context.Background(), ai, "test-model", "   ", 5); err == nil {
		t.Fatal("expected error for empty text")
	}
}
