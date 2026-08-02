package contentpagegeneration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// MaxPromptRunes caps the instructor prompt sent to the model.
const MaxPromptRunes = 8_000

// MaxExistingMarkdownRunes caps optional existing draft markdown sent for revision context.
const MaxExistingMarkdownRunes = 80_000

// MaxSections caps how many draft sections are returned.
const MaxSections = 20

// MaxToolsPerDraft caps interactive tools across the whole page draft.
const MaxToolsPerDraft = 6

// MaxToolsPerSection caps tools attached to a single section.
const MaxToolsPerSection = 2

// MaxHeadingRunes / MaxMarkdownRunes limit individual draft fields.
const (
	MaxHeadingRunes  = 200
	MaxMarkdownRunes = 20_000
)

// DraftTool is a proposed content-tool placement (not persisted; no instanceId).
type DraftTool struct {
	ToolID string          `json:"toolId"`
	Config json.RawMessage `json:"config"`
}

// DraftSection is a proposed content-page section (not persisted).
type DraftSection struct {
	Heading  string      `json:"heading"`
	Markdown string      `json:"markdown"`
	Tools    []DraftTool `json:"tools,omitempty"`
}

// GenerateOpts controls optional interactive tool generation.
type GenerateOpts struct {
	// IncludeTools asks the model to interleave content-tool drafts when pedagogically useful.
	IncludeTools bool
	// AllowedToolIDs is the allowlist sent to the model and used when normalizing output.
	// Empty with IncludeTools=false means prose only; empty with IncludeTools=true uses DefaultAIToolIDs.
	AllowedToolIDs []string
}

// DefaultSystemPrompt instructs the model to return structured section JSON only (prose).
const DefaultSystemPrompt = `You write course content pages for an LMS block editor.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"sections":[{"heading":"...","markdown":"..."}]}.

Rules:
- Produce learner-facing instructional content that matches the instructor's topic description.
- heading: short section title without markdown # prefixes; use "" for a lead-in block with no heading.
- markdown: body content in Markdown only (paragraphs, lists, emphasis, links). Do NOT put ## headings inside markdown — use separate section objects instead.
- Prefer 2–8 clear sections when the topic supports it; return between 1 and 20 sections.
- Write in a professional, accessible tone suitable for students.
- If existing draft content is provided, revise or restructure it to fit the prompt rather than ignoring it.
- If the prompt has no usable topic, return {"sections":[]}.
- Do NOT include interactive tools, tool fences, or a "tools" field.`

// ToolsSystemPromptAppendix is appended when IncludeTools is true (after the prose rules).
const ToolsSystemPromptAppendix = `
Interactive content tools (optional):
You MAY attach zero or more interactive tools to a section via a "tools" array.
Shape: {"sections":[{"heading":"...","markdown":"...","tools":[{"toolId":"...","config":{...}}]}]}.

Pedagogy:
- Prefer 1–4 tools total for a typical page (hard max 6). At most 2 tools per section.
- Place checks after the conceptual content they assess (tool-only sections with empty heading/markdown are fine).
- Use flashcards for vocabulary; inline_questions for comprehension checks; predict_reveal for prediction moments;
  explain_it_back for reflection; class_pulse for quick polls; sort_sequence for categorize/order; worked_example for multi-step procedures;
  highlight_annotate when a short passage should be marked up; inline_discussion / ask_questions sparingly.
- Never invent instanceIds or markdown lex-tool fences — only structured tools entries.
- Only use toolIds from the allowed list provided in the user message.
- Each config must be complete and self-contained (include required nested ids as short strings when needed).

Config shapes (camelCase; omit fields you do not need when defaults exist):
- inline_questions: {"label":"string","attempts":2,"revealCorrectAfter":"last_attempt","sequential":true,"questions":[{"id":"q1","type":"single|multi|true_false|short_text|numeric","prompt":"…","options":[{"id":"a","text":"…","correct":true}],"acceptedAnswers":["…"],"correctValue":0,"explanation":"…","points":1}]}
- flashcards: {"title":"string","cards":[{"id":"c1","front":"…","back":"…"}],"shuffle":true,"sessionCap":20}
- predict_reveal: {"question":"…","mode":"choice","confidenceScale":"three","confidenceRequired":true,"showPeerResults":true,"outcomes":[{"id":"o1","text":"…","correct":true}],"reveal":{"markdown":"…"}}
- explain_it_back: {"prompt":"…","minWords":10,"maxWords":200,"keyPoints":[{"id":"kp1","label":"…","description":"…"}],"revealKeyPointsAfterSubmit":true,"aiFeedback":true,"feedbackStyle":"encouraging","attempts":3}
- class_pulse: {"question":"…","options":[{"id":"a","text":"…"}],"allowSecondVote":false,"revealCorrect":"never","showPercentages":true}
- sort_sequence: {"mode":"categorize|order","prompt":"…","items":[{"id":"i1","text":"…"}],"buckets":[{"id":"b1","label":"…"}],"correctBucketByItem":{"i1":"b1"},"correctOrder":["i1"],"attempts":3,"showPerItemCorrectness":true,"lockCorrect":true,"scoreMode":"per_item","shuffleItems":true}
- worked_example: {"title":"…","problem":"…","blankPolicy":"author","attemptsPerStep":3,"practiceOnly":true,"showAllSteps":false,"steps":[{"id":"s1","label":"Step 1","text":"…","blank":{"type":"text","acceptedAnswers":["…"]}}]}
- highlight_annotate: {"prompt":"…","passageSource":"inline","passageMarkdown":"…","unitGranularity":"sentence","tags":[{"id":"key","label":"Key idea","color":"#0f766e"}],"minAnnotations":1,"maxAnnotations":10,"requireNote":false}
- inline_discussion: {"prompt":"…","postBeforeYouSee":true,"allowReplies":true,"requiredPosts":1,"requiredReplies":0,"anonymity":"named","sort":"oldest","pageSize":20}
- ask_questions: {"intro":"…","placeholder":"Type your question…"}
`

// GenerateFromPrompt asks the model for draft content-page sections.
func GenerateFromPrompt(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, prompt, existingMarkdown, pageTitle string,
	opts GenerateOpts,
) ([]DraftSection, aiprovider.CallMeta, error) {
	p := strings.TrimSpace(prompt)
	if p == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(p) > MaxPromptRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("prompt is too long (max %d characters)", MaxPromptRunes)
	}
	existing := strings.TrimSpace(existingMarkdown)
	if utf8.RuneCountInString(existing) > MaxExistingMarkdownRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("existing markdown is too long (max %d characters)", MaxExistingMarkdownRunes)
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	allowed := NormalizeAllowedToolIDs(opts.AllowedToolIDs)
	if opts.IncludeTools {
		if len(allowed) == 0 {
			allowed = append([]string{}, DefaultAIToolIDs...)
		}
		if !strings.Contains(sys, "Interactive content tools") {
			sys = sys + "\n" + ToolsSystemPromptAppendix
		}
	} else {
		allowed = nil
	}

	var user strings.Builder
	if title := strings.TrimSpace(pageTitle); title != "" {
		fmt.Fprintf(&user, "Page title: %s\n\n", title)
	}
	fmt.Fprintf(&user, "Instructor description of the content:\n%s", p)
	if existing != "" {
		fmt.Fprintf(&user, "\n\nExisting draft content to revise or replace:\n%s", existing)
	}
	if opts.IncludeTools && len(allowed) > 0 {
		fmt.Fprintf(&user, "\n\nInteractive tools are ENABLED. Allowed toolIds: %s\nInclude tools only when they improve learning; omit tools when pure prose is better.", strings.Join(allowed, ", "))
	} else {
		user.WriteString("\n\nInteractive tools are DISABLED. Do not include a tools field.")
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	sections, err := ParseDraftSectionsJSON(res.Text)
	if err != nil {
		return nil, meta, err
	}
	if opts.IncludeTools {
		// Schema validation is applied by the HTTP layer (avoids import cycle with contenttools).
		sections = NormalizeDraftSectionTools(sections, allowed)
	} else {
		sections = StripAllTools(sections)
	}
	return sections, meta, nil
}

// ParseDraftSectionsJSON parses and normalizes model JSON into draft sections.
func ParseDraftSectionsJSON(raw string) ([]DraftSection, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Sections []DraftSection `json:"sections"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse content page sections JSON: %w", err)
	}
	return normalizeDraftSections(payload.Sections), nil
}

func stripJSONFences(raw string) string {
	text := strings.TrimSpace(raw)
	// Only strip markdown fences when they wrap the whole payload (leading fence).
	// Do not scan for ``` inside JSON string values (e.g. fenced code in markdown fields).
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if strings.HasPrefix(strings.ToLower(text), "json") {
			text = text[4:]
			// optional language tag newline
			text = strings.TrimPrefix(text, "\n")
			text = strings.TrimPrefix(text, "\r\n")
		} else if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			// language tag on first line (e.g. ```json or ```)
			text = text[nl+1:]
		}
		// Drop a single trailing fence if present at end.
		if endIdx := strings.LastIndex(text, "```"); endIdx != -1 {
			// Prefer closing fence near the end of the payload.
			after := strings.TrimSpace(text[endIdx+3:])
			if after == "" {
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

func normalizeDraftSections(in []DraftSection) []DraftSection {
	out := make([]DraftSection, 0, len(in))
	for _, s := range in {
		heading := strings.TrimSpace(s.Heading)
		heading = strings.TrimLeft(heading, "#")
		heading = strings.TrimSpace(heading)
		if utf8.RuneCountInString(heading) > MaxHeadingRunes {
			heading = string([]rune(heading)[:MaxHeadingRunes])
		}
		md := strings.TrimSpace(s.Markdown)
		if utf8.RuneCountInString(md) > MaxMarkdownRunes {
			md = string([]rune(md)[:MaxMarkdownRunes])
		}
		tools := normalizeRawTools(s.Tools)
		if heading == "" && md == "" && len(tools) == 0 {
			continue
		}
		out = append(out, DraftSection{Heading: heading, Markdown: md, Tools: tools})
		if len(out) >= MaxSections {
			break
		}
	}
	return out
}

func normalizeRawTools(in []DraftTool) []DraftTool {
	if len(in) == 0 {
		return nil
	}
	out := make([]DraftTool, 0, len(in))
	for _, t := range in {
		id := strings.TrimSpace(t.ToolID)
		if id == "" {
			continue
		}
		cfg := t.Config
		if len(cfg) == 0 {
			cfg = json.RawMessage(`{}`)
		}
		out = append(out, DraftTool{ToolID: id, Config: cfg})
		if len(out) >= MaxToolsPerSection {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
