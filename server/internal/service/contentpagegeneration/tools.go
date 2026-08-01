package contentpagegeneration

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// DefaultAIToolIDs are the text-first tools Build with AI may propose.
// Asset-heavy tools (diagram, media, code, parameter explorer) are excluded.
var DefaultAIToolIDs = []string{
	"inline_questions",
	"flashcards",
	"predict_reveal",
	"explain_it_back",
	"class_pulse",
	"sort_sequence",
	"worked_example",
	"highlight_annotate",
	"inline_discussion",
	"ask_questions",
}

var defaultAIToolSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(DefaultAIToolIDs))
	for _, id := range DefaultAIToolIDs {
		m[id] = struct{}{}
	}
	return m
}()

// NormalizeAllowedToolIDs trims, de-dupes, and keeps only known AI-safe tool IDs.
func NormalizeAllowedToolIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := defaultAIToolSet[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// IntersectAllowedToolIDs returns ids that appear in both slices (order of a preserved).
// If courseAllow is empty, it means "all tools allowed" at the course level.
func IntersectAllowedToolIDs(aiAllow, courseAllow []string) []string {
	ai := NormalizeAllowedToolIDs(aiAllow)
	if len(ai) == 0 {
		ai = append([]string{}, DefaultAIToolIDs...)
	}
	if len(courseAllow) == 0 {
		return ai
	}
	course := make(map[string]struct{}, len(courseAllow))
	for _, id := range courseAllow {
		id = strings.TrimSpace(id)
		if id != "" {
			course[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(ai))
	for _, id := range ai {
		if _, ok := course[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// StripAllTools removes tools from every section (prose-only path).
func StripAllTools(sections []DraftSection) []DraftSection {
	if len(sections) == 0 {
		return sections
	}
	out := make([]DraftSection, len(sections))
	for i, s := range sections {
		out[i] = DraftSection{Heading: s.Heading, Markdown: s.Markdown}
	}
	return out
}

// ToolConfigValidator validates raw config for a toolId. Nil means always accept.
// Callers that can import contenttools should pass ValidateConfigJSON against the registry.
// (This package deliberately does not import contenttools to avoid an import cycle.)
type ToolConfigValidator func(toolID string, config json.RawMessage) error

// NormalizeDraftSectionTools caps tools and applies allowlist + ID/default fixups.
// Without a schema validator, configs may still fail at instance create time — prefer
// NormalizeDraftSectionToolsWith from the HTTP layer with ValidateConfigJSON.
func NormalizeDraftSectionTools(sections []DraftSection, allowed []string) []DraftSection {
	return NormalizeDraftSectionToolsWith(sections, allowed, nil)
}

// NormalizeDraftSectionToolsWith is the testable core of tool normalization.
// Invalid tools (validator error) are dropped; prose is never rejected because of a bad tool.
func NormalizeDraftSectionToolsWith(sections []DraftSection, allowed []string, validate ToolConfigValidator) []DraftSection {
	if len(sections) == 0 {
		return sections
	}
	allow := make(map[string]struct{})
	if len(allowed) == 0 {
		for _, id := range DefaultAIToolIDs {
			allow[id] = struct{}{}
		}
	} else {
		for _, id := range NormalizeAllowedToolIDs(allowed) {
			allow[id] = struct{}{}
		}
	}
	if validate == nil {
		validate = func(string, json.RawMessage) error { return nil }
	}

	totalTools := 0
	out := make([]DraftSection, 0, len(sections))
	for _, s := range sections {
		if totalTools >= MaxToolsPerDraft || len(s.Tools) == 0 {
			out = append(out, DraftSection{Heading: s.Heading, Markdown: s.Markdown})
			continue
		}
		kept := make([]DraftTool, 0, len(s.Tools))
		for _, t := range s.Tools {
			if totalTools >= MaxToolsPerDraft || len(kept) >= MaxToolsPerSection {
				break
			}
			id := strings.TrimSpace(t.ToolID)
			if _, ok := allow[id]; !ok {
				continue
			}
			cfg, ok := prepareToolConfig(id, t.Config)
			if !ok {
				continue
			}
			if err := validate(id, cfg); err != nil {
				continue
			}
			kept = append(kept, DraftTool{ToolID: id, Config: cfg})
			totalTools++
		}
		sec := DraftSection{Heading: s.Heading, Markdown: s.Markdown}
		if len(kept) > 0 {
			sec.Tools = kept
		}
		// Drop empty prose-only sections that lost all tools and had no content.
		if sec.Heading == "" && sec.Markdown == "" && len(sec.Tools) == 0 {
			continue
		}
		out = append(out, sec)
	}
	return out
}

func prepareToolConfig(toolID string, raw json.RawMessage) (json.RawMessage, bool) {
	base := defaultConfigMap(toolID)
	overlay := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &overlay); err != nil {
			// fall back to defaults only
			overlay = map[string]any{}
		}
	}
	// Overlay AI fields onto defaults (AI wins for keys it provides).
	for k, v := range overlay {
		if v == nil {
			continue
		}
		base[k] = v
	}
	ensureNestedIDs(toolID, base)
	// Tool-specific required fixes
	fixupToolConfig(toolID, base)
	out, err := json.Marshal(base)
	if err != nil {
		return nil, false
	}
	return out, true
}

func defaultConfigMap(toolID string) map[string]any {
	// Mirrors clients/web defaultContentToolConfig for AI-safe tools.
	switch toolID {
	case "ask_questions":
		return map[string]any{
			"intro":       "Ask a question about this section.",
			"placeholder": "Type your question…",
		}
	case "inline_discussion":
		return map[string]any{
			"prompt":           "What stands out to you in this section?",
			"postBeforeYouSee": true,
			"allowReplies":     true,
			"requiredPosts":    1,
			"requiredReplies":  0,
			"anonymity":        "named",
			"sort":             "oldest",
			"pageSize":         20,
		}
	case "inline_questions":
		return map[string]any{
			"label":              "Quick check",
			"attempts":           2,
			"revealCorrectAfter": "last_attempt",
			"sequential":         true,
			"shuffleOptions":     false,
			"scorePolicy":        "best",
			"questions": []any{
				map[string]any{
					"id":     "q1",
					"type":   "single",
					"prompt": "Replace this sample question with your own.",
					"options": []any{
						map[string]any{"id": "a", "text": "Option A", "correct": true},
						map[string]any{"id": "b", "text": "Option B", "correct": false},
					},
					"points": 1,
				},
			},
		}
	case "flashcards":
		return map[string]any{
			"title": "New deck",
			"cards": []any{
				map[string]any{"id": "c1", "front": "Front of card 1", "back": "Back of card 1"},
				map[string]any{"id": "c2", "front": "Front of card 2", "back": "Back of card 2"},
			},
			"reversePractice":  false,
			"sessionCap":       20,
			"shuffle":          true,
			"requireFirstPass": true,
		}
	case "class_pulse":
		return map[string]any{
			"question": "Which option best matches the idea?",
			"options": []any{
				map[string]any{"id": "a", "text": "Option A"},
				map[string]any{"id": "b", "text": "Option B"},
			},
			"allowSecondVote": false,
			"revealCorrect":   "never",
			"showPercentages": true,
		}
	case "predict_reveal":
		return map[string]any{
			"question":           "What do you predict will happen?",
			"mode":               "choice",
			"confidenceScale":    "three",
			"confidenceRequired": true,
			"showPeerResults":    true,
			"outcomes": []any{
				map[string]any{"id": "o1", "text": "Outcome A", "correct": true},
				map[string]any{"id": "o2", "text": "Outcome B", "correct": false},
			},
			"reveal": map[string]any{
				"markdown": "Edit this reveal text with the explanation students see after predicting.",
			},
		}
	case "explain_it_back":
		return map[string]any{
			"prompt":   "In your own words, explain the main idea from this section.",
			"minWords": 10,
			"maxWords": 200,
			"keyPoints": []any{
				map[string]any{"id": "kp1", "label": "Idea 1", "description": "First concept the response should cover"},
			},
			"revealKeyPointsAfterSubmit": true,
			"aiFeedback":                 true,
			"feedbackStyle":              "encouraging",
			"attempts":                   3,
		}
	case "highlight_annotate":
		return map[string]any{
			"prompt":          "Highlight the key ideas in the passage.",
			"passageSource":   "inline",
			"passageMarkdown": "Replace this sample passage with the text students should annotate.",
			"unitGranularity": "sentence",
			"tags": []any{
				map[string]any{"id": "key", "label": "Key idea", "color": "#0f766e"},
				map[string]any{"id": "question", "label": "Question", "color": "#b45309"},
			},
			"minAnnotations": 1,
			"maxAnnotations": 10,
			"requireNote":    false,
		}
	case "sort_sequence":
		return map[string]any{
			"mode":   "categorize",
			"prompt": "Sort these items into the correct categories.",
			"items": []any{
				map[string]any{"id": "item_a", "text": "Item A"},
				map[string]any{"id": "item_b", "text": "Item B"},
			},
			"buckets": []any{
				map[string]any{"id": "bucket_a", "label": "Category A"},
				map[string]any{"id": "bucket_b", "label": "Category B"},
			},
			"correctBucketByItem":    map[string]any{"item_a": "bucket_a", "item_b": "bucket_b"},
			"attempts":               3,
			"showPerItemCorrectness": true,
			"lockCorrect":            true,
			"scoreMode":              "per_item",
			"shuffleItems":           true,
		}
	case "worked_example":
		return map[string]any{
			"title":           "Worked example",
			"problem":         "Replace with the problem statement students will work through.",
			"variables":       []any{},
			"blankPolicy":     "author",
			"attemptsPerStep": 3,
			"practiceOnly":    true,
			"showAllSteps":    false,
			"steps": []any{
				map[string]any{
					"id":    "s1",
					"label": "Step 1",
					"text":  "Describe the first step.",
					"blank": map[string]any{
						"type":            "text",
						"acceptedAnswers": []any{"answer"},
					},
				},
			},
		}
	default:
		return map[string]any{}
	}
}

func ensureNestedIDs(toolID string, cfg map[string]any) {
	switch toolID {
	case "inline_questions":
		ensureObjectArrayIDs(cfg, "questions", "q")
		if qs, ok := cfg["questions"].([]any); ok {
			for _, q := range qs {
				qm, ok := q.(map[string]any)
				if !ok {
					continue
				}
				ensureObjectArrayIDs(qm, "options", "opt")
			}
		}
	case "flashcards":
		ensureObjectArrayIDs(cfg, "cards", "c")
	case "class_pulse":
		ensureObjectArrayIDs(cfg, "options", "opt")
	case "predict_reveal":
		ensureObjectArrayIDs(cfg, "outcomes", "o")
	case "explain_it_back":
		ensureObjectArrayIDs(cfg, "keyPoints", "kp")
	case "highlight_annotate":
		ensureObjectArrayIDs(cfg, "tags", "tag")
	case "sort_sequence":
		ensureObjectArrayIDs(cfg, "items", "item")
		ensureObjectArrayIDs(cfg, "buckets", "bucket")
	case "worked_example":
		ensureObjectArrayIDs(cfg, "steps", "s")
	}
}

func ensureObjectArrayIDs(parent map[string]any, key, prefix string) {
	arr, ok := parent[key].([]any)
	if !ok {
		return
	}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if strings.TrimSpace(id) == "" {
			m["id"] = prefix + "_" + shortID()
		}
		arr[i] = m
	}
	parent[key] = arr
}

func fixupToolConfig(toolID string, cfg map[string]any) {
	switch toolID {
	case "inline_questions":
		// Drop empty prompts; ensure at least one question remains (else validation fails → dropped).
		if qs, ok := cfg["questions"].([]any); ok {
			kept := make([]any, 0, len(qs))
			for _, q := range qs {
				qm, ok := q.(map[string]any)
				if !ok {
					continue
				}
				prompt, _ := qm["prompt"].(string)
				if strings.TrimSpace(prompt) == "" {
					continue
				}
				// Cap questions (manifest maxItems is typically 3).
				if len(kept) >= 3 {
					break
				}
				// Ensure type
				typ, _ := qm["type"].(string)
				typ = strings.TrimSpace(typ)
				if typ == "" {
					qm["type"] = "single"
				}
				kept = append(kept, qm)
			}
			cfg["questions"] = kept
		}
	case "flashcards":
		if cards, ok := cfg["cards"].([]any); ok {
			kept := make([]any, 0, len(cards))
			for _, c := range cards {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				front, _ := cm["front"].(string)
				back, _ := cm["back"].(string)
				if strings.TrimSpace(front) == "" || strings.TrimSpace(back) == "" {
					continue
				}
				if len(kept) >= 30 {
					break
				}
				kept = append(kept, cm)
			}
			cfg["cards"] = kept
		}
	case "predict_reveal":
		// Ensure reveal is an object with markdown
		if rev, ok := cfg["reveal"].(map[string]any); ok {
			if md, _ := rev["markdown"].(string); strings.TrimSpace(md) == "" {
				rev["markdown"] = "See the explanation above."
			}
			cfg["reveal"] = rev
		} else if revStr, ok := cfg["reveal"].(string); ok {
			cfg["reveal"] = map[string]any{"markdown": revStr}
		}
	}
}

func shortID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
}
