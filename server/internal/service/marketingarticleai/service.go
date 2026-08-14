// Package marketingarticleai drafts marketing blog/help articles from an author prompt via the AI provider.
package marketingarticleai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const (
	MaxPromptRunes      = 8_000
	MaxExistingRunes    = 80_000
	MaxTitleRunes       = 200
	MaxDescriptionRunes = 160
	MaxFieldRunes       = 200
	MaxBodyRunes        = 80_000
	MaxKeywords         = 12
	MaxSlugRunes        = 100
)

// Draft is a proposed article (not persisted).
type Draft struct {
	Title           string   `json:"title"`
	Slug            string   `json:"slug,omitempty"`
	Description     string   `json:"description"`
	BodyMD          string   `json:"bodyMd"`
	PrimaryQuestion string   `json:"primaryQuestion"`
	Cluster         string   `json:"cluster"`
	Pillar          string   `json:"pillar"`
	Keywords        []string `json:"keywords"`
}

// DefaultSystemPrompt instructs the model to return structured marketing-dialect JSON only.
const DefaultSystemPrompt = `You write marketing content for Lextures, a learning management system.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object:
{
  "title": "...",
  "description": "...",
  "primaryQuestion": "...",
  "cluster": "...",
  "pillar": "...",
  "keywords": ["..."],
  "bodyMd": "..."
}

Rules:
- title: specific, human, no clickbait. No trailing period.
- description: 1–2 sentences, at most 160 characters, suitable for search and social cards.
- primaryQuestion: the search-style question this article answers, ending with ?.
- cluster: short topic cluster label (2–4 words).
- pillar: short editorial pillar (2–4 words).
- keywords: 3–8 lowercase phrases, no hashtags.
- bodyMd: CommonMark only. No raw HTML, JSX, scripts, or javascript:/data: URLs.

Body structure (required):
1. Open with a :::key-takeaways block containing 3–5 Markdown bullets.
2. Then a :::answer block: 40–60 words that directly answer primaryQuestion.
3. Then 4–7 level-two headings that are questions (end with ?). Each section is a self-contained passage of 80–160 words.
4. Use at least one internal link to a real Lextures path. /blog, /docs, /platform, and /pricing always resolve.
5. Close with a :::faq block containing 3–5 entries, each a ### Question? heading plus a short paragraph.
6. Cite numeric claims with [^1] and define them at the end as [^1]: https://….
7. Optional :::callout tip or :::stat blocks are fine; never invent unknown directives.

Tone:
- For kind=blog: clear, expert, skeptical of hype. Write for school leaders and teachers.
- For kind=doc: practical how-to. Lead with what to click and what happens next.
- Do not invent product features, statistics, or customer quotes you cannot support.
- If the prompt has no usable topic, return empty strings and an empty keywords array.`

// MetadataSystemPrompt instructs the model to return essentials metadata only.
const MetadataSystemPrompt = `You write marketing metadata for Lextures, a learning management system.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object:
{
  "slug": "...",
  "description": "...",
  "primaryQuestion": "...",
  "cluster": "...",
  "pillar": "...",
  "keywords": ["..."]
}

Rules:
- slug: lowercase kebab-case ASCII, 3–8 words, no leading or trailing hyphens.
- description: 1–2 sentences, at most 160 characters, suitable for search and social cards.
- primaryQuestion: the search-style question this article answers, ending with ?.
- cluster: short topic cluster label (2–4 words).
- pillar: short editorial pillar (2–4 words).
- keywords: 3–8 lowercase phrases, no hashtags.
- Do not invent product features, statistics, or customer quotes.
- If there is no usable topic, return empty strings and an empty keywords array.`

// GenerateFromPrompt asks the model for a draft marketing article.
func GenerateFromPrompt(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, kind, prompt, existingTitle, existingBody string,
) (Draft, aiprovider.CallMeta, error) {
	if client == nil {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("marketingarticleai: nil completer")
	}
	p := strings.TrimSpace(prompt)
	if p == "" {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(p) > MaxPromptRunes {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("prompt is too long (max %d characters)", MaxPromptRunes)
	}
	existingTitle = strings.TrimSpace(existingTitle)
	existingBody = strings.TrimSpace(existingBody)
	if utf8.RuneCountInString(existingBody) > MaxExistingRunes {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("existing markdown is too long (max %d characters)", MaxExistingRunes)
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}
	if kind != "doc" {
		kind = "blog"
	}

	var user strings.Builder
	fmt.Fprintf(&user, "Article kind: %s\n\nAuthor description of the article:\n%s", kind, p)
	if existingTitle != "" {
		fmt.Fprintf(&user, "\n\nCurrent title (revise or replace):\n%s", existingTitle)
	}
	if existingBody != "" {
		fmt.Fprintf(&user, "\n\nCurrent draft (revise or replace):\n%s", existingBody)
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true, MaxTokens: 8_000})
	if err != nil {
		return Draft{}, meta, err
	}
	draft, err := ParseDraftJSON(res.Text)
	if err != nil {
		return Draft{}, meta, err
	}
	return draft, meta, nil
}

// GenerateMetadataFromContent asks the model for essentials metadata only (no body).
func GenerateMetadataFromContent(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, kind, title, body string,
) (Draft, aiprovider.CallMeta, error) {
	if client == nil {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("marketingarticleai: nil completer")
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("title or body is required")
	}
	if utf8.RuneCountInString(body) > MaxExistingRunes {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("existing markdown is too long (max %d characters)", MaxExistingRunes)
	}
	if kind != "doc" {
		kind = "blog"
	}

	var user strings.Builder
	fmt.Fprintf(&user, "Article kind: %s\nFill the essentials metadata from this article.", kind)
	if title != "" {
		fmt.Fprintf(&user, "\n\nTitle:\n%s", title)
	}
	if body != "" {
		fmt.Fprintf(&user, "\n\nBody:\n%s", body)
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: MetadataSystemPrompt},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true, MaxTokens: 800})
	if err != nil {
		return Draft{}, meta, err
	}
	draft, err := ParseDraftJSON(res.Text)
	if err != nil {
		return Draft{}, meta, err
	}
	draft.Title = ""
	draft.BodyMD = ""
	if draft.Slug == "" && title != "" {
		draft.Slug = slugifySlug(title)
	}
	return draft, meta, nil
}

// ParseDraftJSON parses and normalizes model JSON into a draft article.
func ParseDraftJSON(raw string) (Draft, error) {
	text := stripJSONFences(raw)
	var draft Draft
	if err := json.Unmarshal([]byte(text), &draft); err != nil {
		return Draft{}, fmt.Errorf("parse marketing article JSON: %w", err)
	}
	return normalizeDraft(draft), nil
}

func normalizeDraft(in Draft) Draft {
	out := Draft{
		Title:           clipRunes(strings.TrimSpace(in.Title), MaxTitleRunes),
		Slug:            slugifySlug(in.Slug),
		Description:     clipRunes(strings.TrimSpace(in.Description), MaxDescriptionRunes),
		BodyMD:          clipRunes(strings.TrimSpace(in.BodyMD), MaxBodyRunes),
		PrimaryQuestion: clipRunes(strings.TrimSpace(in.PrimaryQuestion), MaxFieldRunes),
		Cluster:         clipRunes(strings.TrimSpace(in.Cluster), MaxFieldRunes),
		Pillar:          clipRunes(strings.TrimSpace(in.Pillar), MaxFieldRunes),
	}
	seen := map[string]struct{}{}
	for _, kw := range in.Keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}
		if utf8.RuneCountInString(kw) > MaxFieldRunes {
			kw = string([]rune(kw)[:MaxFieldRunes])
		}
		if _, ok := seen[kw]; ok {
			continue
		}
		seen[kw] = struct{}{}
		out.Keywords = append(out.Keywords, kw)
		if len(out.Keywords) >= MaxKeywords {
			break
		}
	}
	if out.Keywords == nil {
		out.Keywords = []string{}
	}
	return out
}

func slugifySlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return clipRunes(out, MaxSlugRunes)
}

func clipRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func stripJSONFences(raw string) string {
	text := strings.TrimSpace(raw)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if strings.HasPrefix(strings.ToLower(text), "json") {
			text = text[4:]
			text = strings.TrimPrefix(text, "\n")
			text = strings.TrimPrefix(text, "\r\n")
		} else if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			text = text[nl+1:]
		}
		if endIdx := strings.LastIndex(text, "```"); endIdx != -1 {
			if strings.TrimSpace(text[endIdx+3:]) == "" {
				text = text[:endIdx]
			}
		}
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		if start := strings.Index(text, "{"); start != -1 {
			if end := strings.LastIndex(text, "}"); end > start {
				text = text[start : end+1]
			}
		}
	}
	return strings.TrimSpace(text)
}
