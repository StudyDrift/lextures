package alttextai

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

type mockVisionCompleter struct {
	fn func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error)
}

func (m mockVisionCompleter) CompleteVision(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	return m.fn(ctx, model, messages, opts...)
}

func TestSuggest(t *testing.T) {
	ai := mockVisionCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		return aiprovider.ChatResult{Text: "A red circle on a white background."}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	suggestion, confidence, meta, err := Suggest(context.Background(), ai, DefaultModel, "https://example.com/img.png", "English")
	if err != nil {
		t.Fatal(err)
	}
	if suggestion == "" {
		t.Fatal("expected suggestion")
	}
	if confidence <= 0 {
		t.Fatalf("expected confidence > 0, got %v", confidence)
	}
	if meta.ModelID != DefaultModel {
		t.Fatalf("expected model id propagated, got %q", meta.ModelID)
	}
}

func TestSuggest_NilCompleter(t *testing.T) {
	if _, _, _, err := Suggest(context.Background(), nil, DefaultModel, "https://example.com/img.png", ""); err == nil {
		t.Fatal("expected error for nil completer")
	}
}

func TestSuggest_MissingImageURL(t *testing.T) {
	ai := mockVisionCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call CompleteVision without an image URL")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, _, err := Suggest(context.Background(), ai, DefaultModel, "  ", ""); err == nil {
		t.Fatal("expected error for missing image URL")
	}
}
