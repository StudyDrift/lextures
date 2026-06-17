// Package plagiarism implements originality scan providers and orchestration (plan 14.8).
package plagiarism

import (
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const (
	ProviderInternal  = "internal"
	ProviderTurnitin  = "turnitin"
	ProviderCopyleaks = "copyleaks"
	ProviderGPTZero   = "gptzero"
)

// ScanResult is the outcome of a provider scan.
type ScanResult struct {
	SimilarityPct    *float64
	AIProbability    *float64
	ReportURL        *string
	ReportToken      *string
	ProviderReportID *string
}

// Provider scans submission text for similarity or AI authorship signals.
type Provider interface {
	Name() string
	Scan(ctx context.Context, text string) (ScanResult, error)
}

// InternalAIProvider scores AI-authorship probability via OpenRouter.
type InternalAIProvider struct {
	Client *openrouter.Client
	Model  string
}

func (p InternalAIProvider) Name() string { return ProviderInternal }

func (p InternalAIProvider) Scan(ctx context.Context, text string) (ScanResult, error) {
	_ = ctx
	if p.Client == nil {
		return ScanResult{}, fmt.Errorf("internal ai: missing openrouter client")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ScanResult{}, fmt.Errorf("internal ai: empty text")
	}
	if len(trimmed) > 16000 {
		trimmed = trimmed[:16000]
	}
	model := p.Model
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	out, err := p.Client.ChatCompletion(model, []openrouter.Message{
		{
			Role: "system",
			Content: "You classify whether student writing was likely AI-generated. " +
				"Reply with a single number from 0 to 100 representing AI-authorship probability. No other text.",
		},
		{Role: "user", Content: trimmed},
	})
	if err != nil {
		return ScanResult{}, err
	}
	score := parseScorePercent(out.Text)
	if score == nil {
		return ScanResult{}, fmt.Errorf("internal ai: could not parse score")
	}
	return ScanResult{AIProbability: score}, nil
}

// StubExternalProvider simulates an external plagiarism provider for dev / stub mode.
type StubExternalProvider struct {
	Name_ string
}

func (p StubExternalProvider) Name() string {
	if p.Name_ == "" {
		return ProviderTurnitin
	}
	return p.Name_
}

func (p StubExternalProvider) Scan(ctx context.Context, text string) (ScanResult, error) {
	_ = ctx
	n := stubSimilarityScore(text)
	reportID := fmt.Sprintf("stub-%s-%d", p.Name(), len(text))
	url := fmt.Sprintf("https://example.com/originality/%s", reportID)
	token := "stub-embed-token"
	return ScanResult{
		SimilarityPct:    &n,
		ReportURL:        &url,
		ReportToken:      &token,
		ProviderReportID: &reportID,
	}, nil
}

func stubSimilarityScore(text string) float64 {
	words := len(strings.Fields(text))
	base := float64(words%37) + float64(len(text)%23)
	score := math.Mod(base, 85)
	if score < 5 {
		score = 5
	}
	return math.Round(score*10) / 10
}

var scoreRE = regexp.MustCompile(`(\d{1,3})(?:\.\d+)?`)

func parseScorePercent(raw string) *float64 {
	m := scoreRE.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) < 2 {
		return nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 {
		return nil
	}
	if n > 100 {
		n = 100
	}
	f := float64(n)
	return &f
}

// ReadTextFromReader loads up to maxBytes of UTF-8 text from r.
func ReadTextFromReader(r io.Reader, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 512 << 10
	}
	limited := io.LimitReader(r, maxBytes)
	b, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
