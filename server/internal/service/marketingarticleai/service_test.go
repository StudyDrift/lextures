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

func TestGenerateMetadataFromContent(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		if model != "test-model" {
			t.Fatalf("model=%q", model)
		}
		if len(messages) != 2 || messages[0].Content != MetadataSystemPrompt {
			t.Fatalf("messages=%#v", messages)
		}
		if !strings.Contains(messages[1].Content, "Title:\nHomeschool advice") {
			t.Fatalf("user=%q", messages[1].Content)
		}
		if strings.Contains(messages[1].Content, "bodyMd") {
			t.Fatalf("metadata prompt should not ask for a body")
		}
		if len(opts) == 0 || !opts[0].JSONMode || opts[0].MaxTokens != 800 {
			t.Fatalf("opts=%#v", opts)
		}
		return aiprovider.ChatResult{Text: `{"slug":"Homeschool Student Advice!","description":"Practical advice for homeschool families.","primaryQuestion":"How do I support a homeschool student?","cluster":"Homeschool","pillar":"Advice","keywords":["homeschool"],"title":"Ignore","bodyMd":"Ignore"}`}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	got, meta, err := GenerateMetadataFromContent(context.Background(), ai, "test-model", "blog", "Homeschool advice", "A draft body")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "homeschool-student-advice" || got.Description != "Practical advice for homeschool families." {
		t.Fatalf("got %#v", got)
	}
	if got.SocialTitle != "Homeschool advice" || got.SocialDescription != got.Description {
		t.Fatalf("social fallback %#v", got)
	}
	if got.Title != "" || got.BodyMD != "" {
		t.Fatalf("metadata draft should clear title/body: %#v", got)
	}
	if got.PrimaryQuestion == "" || meta.ModelID != "test-model" {
		t.Fatalf("got %#v meta=%#v", got, meta)
	}
}

func TestRepairFromFindings(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		if model != "test-model" {
			t.Fatalf("model=%q", model)
		}
		if len(messages) != 2 || messages[0].Content != RepairSystemPrompt {
			t.Fatalf("messages=%#v", messages)
		}
		user := messages[1].Content
		if !strings.Contains(user, "Article kind: blog") {
			t.Fatalf("user=%q", user)
		}
		if !strings.Contains(user, "[warning] passage.length — Direct answer is 69 words") {
			t.Fatalf("expected warning finding in prompt: %q", user)
		}
		if !strings.Contains(user, "(line 10)") {
			t.Fatalf("expected line in prompt: %q", user)
		}
		if !strings.Contains(user, "[error] fm.cluster — Required metadata field is missing. (path: cluster)") {
			t.Fatalf("expected error finding in prompt: %q", user)
		}
		if !strings.Contains(user, "Current draft:\n:::answer") {
			t.Fatalf("expected body in prompt: %q", user)
		}
		if !strings.Contains(user, "/blog") || !strings.Contains(user, "always resolve") {
			t.Fatalf("expected hub paths to be treated as valid: %q", user)
		}
		if len(opts) == 0 || !opts[0].JSONMode || opts[0].MaxTokens != 12_000 {
			t.Fatalf("opts=%#v", opts)
		}
		return aiprovider.ChatResult{Text: `{"title":"Fixed","description":"D","primaryQuestion":"Q?","cluster":"C","pillar":"P","keywords":["k"],"bodyMd":":::answer\nYes.\n:::"}`}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	got, meta, err := RepairFromFindings(context.Background(), ai, "test-model", RepairInput{
		Kind:    "blog",
		Title:   "Old",
		BodyMD:  ":::answer\nToo long answer text.\n:::",
		Cluster: "",
		Findings: []Finding{
			{Rule: "passage.length", Severity: "warning", Message: "Direct answer is 69 words; target is 40-60.", Line: 10},
			{Rule: "fm.cluster", Severity: "error", Message: "Required metadata field is missing.", Path: "cluster"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Fixed" || got.BodyMD == "" || meta.ModelID != "test-model" {
		t.Fatalf("got %#v meta=%#v", got, meta)
	}
}

func TestRepairFromFindings_RetriesInvalidJSON(t *testing.T) {
	calls := 0
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		calls++
		if calls == 1 {
			return aiprovider.ChatResult{Text: "not json"}, aiprovider.CallMeta{ModelID: model}, nil
		}
		return aiprovider.ChatResult{Text: `{"title":"Fixed","description":"D","primaryQuestion":"Q?","cluster":"C","pillar":"P","keywords":["k"],"bodyMd":":::answer\nYes.\n:::"}`}, aiprovider.CallMeta{ModelID: model}, nil
	}}
	got, _, err := RepairFromFindings(context.Background(), ai, "test-model", RepairInput{
		Title:    "Old",
		BodyMD:   ":::answer\nToo long.\n:::",
		Findings: []Finding{{Rule: "passage.length", Message: "too long"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || got.Title != "Fixed" {
		t.Fatalf("calls=%d got=%#v", calls, got)
	}
}

func TestRepairFromFindings_EmptyFindings(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call Complete")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, err := RepairFromFindings(context.Background(), ai, "m", RepairInput{Title: "T", Findings: nil}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRepairFromFindings_EmptySource(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call Complete")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, err := RepairFromFindings(context.Background(), ai, "m", RepairInput{
		Findings: []Finding{{Rule: "x", Message: "y"}},
	}); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateMetadataFromContent_EmptySource(t *testing.T) {
	ai := mockCompleter{fn: func(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
		t.Fatal("should not call Complete")
		return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
	}}
	if _, _, err := GenerateMetadataFromContent(context.Background(), ai, "m", "blog", "  ", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestSlugifySlug(t *testing.T) {
	if got := slugifySlug("  Homeschooling Student Advice!  "); got != "homeschooling-student-advice" {
		t.Fatalf("got %q", got)
	}
}

func utf8Count(s string) int {
	return len([]rune(s))
}
