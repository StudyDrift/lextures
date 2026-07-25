package adaptivecontent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// stubCompleter is a ScopedCompleter test double.
type stubCompleter struct {
	responses []string
	calls     int
	lastMsgs  []aiprovider.Message
}

func (s *stubCompleter) Complete(ctx context.Context, modelOverride string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	s.calls++
	s.lastMsgs = messages
	idx := s.calls - 1
	text := `{"sections":[{"heading":"Intro","markdown":"body"}]}`
	if idx < len(s.responses) {
		text = s.responses[idx]
	} else if len(s.responses) > 0 {
		text = s.responses[len(s.responses)-1]
	}
	return aiprovider.ChatResult{
		Text: text,
		Usage: aiprovider.UsageInfo{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}, aiprovider.CallMeta{
		Provider: aiprovider.ProviderDryRun,
		ModelID:  modelOverride,
		Usage:    aiprovider.UsageInfo{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}, nil
}

func TestBuildUserPrompt_IncludesGuardrails(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	in := GenerateInput{
		BaseMarkdown: "Base facts about ATP synthase and $E=mc^2$.",
		BaseTitle:    "Cell Energy",
		Profile: ProfileResult{
			EmphasisMode:     EmphasisRemediate,
			TargetBloom:      "understand",
			ProfileSignature: "abc",
			AxisSet:          []string{"emphasis", "misconception"},
			Payload: ProfilePayload{
				ConceptGaps:    []ConceptGap{{ConceptID: cid, Gap: 0.7}},
				Misconceptions: []string{"m1"},
			},
		},
		AllowedAxes:         []string{"emphasis", "misconception"},
		KeyTerms:            []string{"ATP synthase"},
		ConceptLabels:       map[uuid.UUID]string{cid: "Respiration"},
		MisconceptionLabels: map[string]string{"m1": "Plants only respire at night"},
	}
	p := BuildUserPrompt(in)
	for _, want := range []string{
		"AUTHORITATIVE base content",
		"Emphasis mode: remediate",
		"ATP synthase",
		"Respiration",
		"Plants only respire at night",
		"Target Bloom level: understand",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
}

func TestGenerateVariant_NeutralShortCircuit(t *testing.T) {
	t.Parallel()
	stub := &stubCompleter{}
	in := GenerateInput{
		BaseMarkdown:   "Original lesson body.",
		BaseTitle:      "Lesson",
		Profile:        NeutralProfile(uuid.New(), nil, "default", "default"),
		GatewayAllowed: true,
		Model:          "test-model",
	}
	v, _, err := GenerateVariant(context.Background(), stub, in)
	if err != nil {
		t.Fatal(err)
	}
	if stub.calls != 0 {
		t.Fatalf("expected no model calls, got %d", stub.calls)
	}
	if !v.Fallback || v.FallbackReason != "neutral_profile" {
		t.Fatalf("variant=%+v", v)
	}
	if !strings.Contains(v.Markdown, "Original lesson body") {
		t.Fatalf("markdown=%q", v.Markdown)
	}
}

func TestGenerateVariant_GatewayDenied(t *testing.T) {
	t.Parallel()
	stub := &stubCompleter{}
	in := GenerateInput{
		BaseMarkdown: "Body",
		Profile: ProfileResult{
			EmphasisMode:     EmphasisCompress,
			ProfileSignature: "sig1",
			IsNeutral:        false,
		},
		GatewayAllowed: false,
		Model:          "m",
	}
	v, _, err := GenerateVariant(context.Background(), stub, in)
	if err != ErrGatewayDenied {
		t.Fatalf("err=%v", err)
	}
	if stub.calls != 0 {
		t.Fatal("model must not be called")
	}
	if !v.Fallback {
		t.Fatal("expected fallback")
	}
}

func TestGenerateVariant_FidelityRejectMissingKeyTerm(t *testing.T) {
	t.Parallel()
	// Model returns content that drops the required key term.
	stub := &stubCompleter{responses: []string{
		`{"sections":[{"heading":"H","markdown":"A short summary without the special term."}]}`,
	}}
	in := GenerateInput{
		BaseMarkdown: "Lesson about **Mitochondria** and ATP.\n\n```js\nconst x = 1;\n```\n\nFormula $E=mc^2$.",
		Profile: ProfileResult{
			EmphasisMode:     EmphasisCompress,
			ProfileSignature: "sig-compress",
			Payload:          ProfilePayload{},
		},
		KeyTerms:       []string{"Mitochondria"},
		GatewayAllowed: true,
		Model:          "m",
		SkipJudge:      true,
		MinFidelity:    0.85,
	}
	v, meta, err := GenerateVariant(context.Background(), stub, in)
	if err != ErrRejectedFidelity {
		t.Fatalf("err=%v v=%+v", err, v)
	}
	if v.Status != "rejected" || !v.Fallback {
		t.Fatalf("status=%s fallback=%v", v.Status, v.Fallback)
	}
	if meta.Usage.TotalTokens == 0 {
		t.Fatal("expected token usage")
	}
}

func TestGenerateVariant_SuccessWithHardChecks(t *testing.T) {
	t.Parallel()
	body := "Mitochondria produce ATP. Code:\n```js\nconst x = 1;\n```\nEnergy $E=mc^2$."
	payload, _ := json.Marshal(map[string]any{
		"sections": []map[string]string{
			{"heading": "Summary", "markdown": body},
		},
	})
	stub := &stubCompleter{responses: []string{string(payload)}}
	in := GenerateInput{
		BaseMarkdown: body,
		Profile: ProfileResult{
			EmphasisMode:     EmphasisCompress,
			ProfileSignature: "sig-ok",
			AxisSet:          []string{"emphasis"},
		},
		AllowedAxes:    []string{"emphasis"},
		KeyTerms:       []string{"Mitochondria", "ATP"},
		GatewayAllowed: true,
		Model:          "m",
		SkipJudge:      true,
		MinFidelity:    0.5, // lexical may be lower; hard checks pass
	}
	v, _, err := GenerateVariant(context.Background(), stub, in)
	if err != nil {
		t.Fatalf("err=%v flags=%v score=%v", err, v.SafetyFlags, v.FidelityScore)
	}
	if v.Status != "auto_served" {
		t.Fatalf("status=%s score=%v flags=%v", v.Status, v.FidelityScore, v.SafetyFlags)
	}
	if stub.calls != 1 {
		t.Fatalf("calls=%d", stub.calls)
	}
}

func TestGenerateVariant_RequireApproval(t *testing.T) {
	t.Parallel()
	body := "Mitochondria produce ATP for the cell through respiration processes."
	payload, _ := json.Marshal(map[string]any{
		"sections": []map[string]string{{"heading": "", "markdown": body}},
	})
	stub := &stubCompleter{responses: []string{string(payload)}}
	in := GenerateInput{
		BaseMarkdown: body,
		Profile: ProfileResult{
			EmphasisMode:     EmphasisReinforce,
			ProfileSignature: "sig-apr",
		},
		KeyTerms:                  []string{"Mitochondria"},
		GatewayAllowed:            true,
		Model:                     "m",
		SkipJudge:                 true,
		RequireInstructorApproval: true,
		MinFidelity:               0.5,
	}
	v, _, err := GenerateVariant(context.Background(), stub, in)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != "pending_review" {
		t.Fatalf("status=%s", v.Status)
	}
}

func TestSyntheticProfileFromRequest(t *testing.T) {
	t.Parallel()
	p := SyntheticProfileFromRequest(EmphasisCompress, "", "", "", nil, nil, nil)
	if p.EmphasisMode != EmphasisCompress {
		t.Fatal(p.EmphasisMode)
	}
	if p.ProfileSignature == "" || p.IsNeutral {
		t.Fatalf("%+v", p)
	}
	// Same inputs → same signature.
	p2 := SyntheticProfileFromRequest(EmphasisCompress, "", "", "", nil, nil, nil)
	if p.ProfileSignature != p2.ProfileSignature {
		t.Fatal("signature not stable")
	}
}
