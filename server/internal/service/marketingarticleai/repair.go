package marketingarticleai

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const MaxFindings = 40

// Finding is a lint issue the repair pass should resolve.
type Finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Path     string `json:"path,omitempty"`
}

// RepairInput is the current article plus the findings to fix.
type RepairInput struct {
	Kind            string
	Title           string
	BodyMD          string
	Description     string
	PrimaryQuestion string
	Cluster         string
	Pillar          string
	Keywords        []string
	KnownPaths      []string
	Findings        []Finding
}

// RepairSystemPrompt instructs the model to walk every finding and return a complete revised article.
const RepairSystemPrompt = `You revise existing marketing content for Lextures, a learning management system.
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

Task:
- Go through EACH listed finding in order, including warnings, and apply a fix for every one.
- Return the complete revised article after all findings are addressed. Do not omit unresolved items.
- Preserve the author's topic, voice, and accurate claims. Change only what the findings require, plus the minimum surrounding edits so the article still reads as one piece.
- Do not invent product features, statistics, customer quotes, or sources you cannot support.

Field rules:
- title: specific, human, no clickbait. No trailing period.
- description: 1–2 sentences, at most 160 characters, suitable for search and social cards.
- primaryQuestion: the search-style question this article answers, ending with ?.
- cluster: short topic cluster label (2–4 words).
- pillar: short editorial pillar (2–4 words).
- keywords: 3–8 lowercase phrases, no hashtags.
- bodyMd: CommonMark only. No raw HTML, JSX, scripts, or javascript:/data: URLs.

Body contract (required after repair):
1. :::key-takeaways with 3–5 Markdown bullets.
2. :::answer with a 40–60 word direct answer to primaryQuestion.
3. 4–7 level-two headings that are questions (end with ?). Each section is a self-contained passage of 120–180 words (or 80–160 words if the draft is already close and only needs a small expansion).
4. At least one internal link to a real Lextures path. Hub pages that always resolve include /blog, /docs, /platform, and /pricing — keep those links; they are not broken. For any other internal link, use only a path listed as valid. Do not invent unknown article slugs.
5. Close with :::faq containing 3–5 entries, each a ### Question? heading plus a short paragraph.
6. Every [^n] citation must have a definition at the end as [^n]: https://…. If a numeric claim cannot be sourced, rephrase it so a citation is not required. Never invent a fake URL.
7. Optional :::callout tip or :::stat blocks are fine; never invent unknown directives.

Tone:
- For kind=blog: clear, expert, skeptical of hype. Write for school leaders and teachers.
- For kind=doc: practical how-to. Lead with what to click and what happens next.`

// RepairFromFindings asks the model to revise the article so every listed finding is resolved.
func RepairFromFindings(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model string,
	in RepairInput,
) (Draft, aiprovider.CallMeta, error) {
	if client == nil {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("marketingarticleai: nil completer")
	}
	in.Title = strings.TrimSpace(in.Title)
	in.BodyMD = strings.TrimSpace(in.BodyMD)
	if in.Title == "" && in.BodyMD == "" {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("title or body is required")
	}
	if utf8.RuneCountInString(in.BodyMD) > MaxExistingRunes {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("existing markdown is too long (max %d characters)", MaxExistingRunes)
	}
	findings := normalizeRepairFindings(in.Findings)
	if len(findings) == 0 {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("at least one finding is required")
	}
	if in.Kind != "doc" {
		in.Kind = "blog"
	}

	var user strings.Builder
	fmt.Fprintf(&user, "Article kind: %s\n\nGo through each finding below and apply a fix for every one, including warnings.\n", in.Kind)
	user.WriteString("\nFindings:\n")
	for i, finding := range findings {
		sev := strings.TrimSpace(finding.Severity)
		if sev == "" {
			sev = "warning"
		}
		fmt.Fprintf(&user, "%d. [%s] %s — %s", i+1, sev, strings.TrimSpace(finding.Rule), strings.TrimSpace(finding.Message))
		if finding.Path != "" {
			fmt.Fprintf(&user, " (path: %s)", finding.Path)
		}
		if finding.Line > 0 {
			fmt.Fprintf(&user, " (line %d)", finding.Line)
		}
		user.WriteByte('\n')
	}
	if in.Title != "" {
		fmt.Fprintf(&user, "\nCurrent title:\n%s\n", in.Title)
	}
	writeRepairMeta(&user, "description", in.Description)
	writeRepairMeta(&user, "primaryQuestion", in.PrimaryQuestion)
	writeRepairMeta(&user, "cluster", in.Cluster)
	writeRepairMeta(&user, "pillar", in.Pillar)
	if len(in.Keywords) > 0 {
		fmt.Fprintf(&user, "\nCurrent keywords: %s\n", strings.Join(in.Keywords, ", "))
	}
	if paths := repairKnownPaths(in.KnownPaths); len(paths) > 0 {
		fmt.Fprintf(&user, "\nValid internal paths (use only these for new or replacement links; /blog /docs /platform /pricing always resolve):\n%s\n", strings.Join(paths, ", "))
	}
	if in.BodyMD != "" {
		fmt.Fprintf(&user, "\nCurrent draft:\n%s", in.BodyMD)
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: RepairSystemPrompt},
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

const maxRepairKnownPaths = 80

func repairKnownPaths(in []string) []string {
	seen := map[string]struct{}{
		"/blog": {}, "/docs": {}, "/platform": {}, "/pricing": {},
	}
	out := []string{"/blog", "/docs", "/platform", "/pricing"}
	for _, path := range in {
		path = strings.TrimSpace(path)
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
		if len(out) >= maxRepairKnownPaths {
			break
		}
	}
	return out
}

func writeRepairMeta(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		fmt.Fprintf(b, "\nCurrent %s: (empty)\n", label)
		return
	}
	fmt.Fprintf(b, "\nCurrent %s:\n%s\n", label, value)
}

func normalizeRepairFindings(in []Finding) []Finding {
	out := make([]Finding, 0, len(in))
	for _, finding := range in {
		rule := strings.TrimSpace(finding.Rule)
		message := strings.TrimSpace(finding.Message)
		if rule == "" && message == "" {
			continue
		}
		sev := strings.ToLower(strings.TrimSpace(finding.Severity))
		switch sev {
		case "error", "info":
		default:
			sev = "warning"
		}
		out = append(out, Finding{
			Rule:     rule,
			Severity: sev,
			Message:  message,
			Line:     finding.Line,
			Path:     strings.TrimSpace(finding.Path),
		})
		if len(out) >= MaxFindings {
			break
		}
	}
	return out
}
