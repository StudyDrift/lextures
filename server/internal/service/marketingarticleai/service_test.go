package marketingarticleai

import (
	"context"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

type mockCompleter struct {
	fn func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error)
}

func (m mockCompleter) Complete(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	return m.fn(ctx, model, messages, opts...)
}

func TestParseDraftJSON_Success(t *testing.T) {
	t.Parallel()
	raw := "```json\n{\"title\":\"  Hello \",\"description\":\"A desc.\",\"primaryQuestion\":\"What is it?\",\"cluster\":\"LMS\",\"pillar\":\"Product\",\"keywords\":[\"Courses\",\"courses\",\"  \"],\"bodyMd\":\":::key-takeaways\\n- One\\n:::\"}\n```"
	got, err := ParseDraftJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Hello" || got.Description != "A desc." || got.PrimaryQuestion != "What is it?" {
		t.Fatalf("got %#v", got)
	}
	if len(got.Keywords) != 1 || got.Keywords[0] != "courses" {
		t.Fatalf("keywords=%v", got.Keywords)
	}
	if !strings.Contains(got.BodyMD, "key-takeaways") {
		t.Fatalf("body=%q", got.BodyMD)
	}
}

func TestParseDraftJSON_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseDraftJSON("not json"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeDraft_CapsAndEmptyKeywords(t *testing.T) {
	t.Parallel()
	got := normalizeDraft(Draft{
		Title:       strings.Repeat("T", MaxTitleRunes+10),
		Description: strings.Repeat("D", MaxDescriptionRunes+10),
		Keywords:    nil,
	})
	if utf8Count(got.Title) != MaxTitleRunes {
		t.Fatalf("title runes=%d", utf8Count(got.Title))
	}
	if utf8Count(got.Description) != MaxDescriptionRunes {
		t.Fatalf("description runes=%d", utf8Count(got.Description))
	}
	if got.Keywords == nil || len(got.Keywords) != 0 {
		t.Fatalf("keywords=%#v", got.Keywords)
	}
}

func TestGenerateFromPrompt(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		if model != "test-model" {
			t.Fatalf("model=%q", model)
		}
		if len(messages) != 2 || messages[1].Role != "user" {
			t.Fatalf("messages=%#v", messages)
		}
		if !strings.Contains(messages[1].Content, "Article kind: blog") {
			t.Fatalf("user=%q", messages[1].Content)
		}
		if !strings.Contains(messages[1].Content, "Current title") {
			t.Fatalf("expected existing title in prompt")
		}
		if len(opts) == 0 || !opts[0].JSONMode {
			t.Fatal("expected JSONMode")
		}
		return aiprovider.ChatResult{Text: `{"title":"T","description":"D","primaryQuestion":"Q?","cluster":"C","pillar":"P","keywords":["k"],"bodyMd":"B"}`}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	got, meta, err := GenerateFromPrompt(context.Background(), ai, "test-model", "", "blog", "Write about grades", "Old title", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T" || got.BodyMD != "B" || meta.ModelID != "test-model" {
		t.Fatalf("got %#v meta=%#v", got, meta)
	}
}

func TestGenerateFromPrompt_EmptyPrompt(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call Complete")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, err := GenerateFromPrompt(context.Background(), ai, "m", "", "blog", "   ", "", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateFromPrompt_NilCompleter(t *testing.T) {
	if _, _, err := GenerateFromPrompt(context.Background(), nil, "m", "", "blog", "topic", "", ""); err == nil {
		t.Fatal("expected error")
	}
}

func utf8Count(s string) int {
	return len([]rune(s))
}
