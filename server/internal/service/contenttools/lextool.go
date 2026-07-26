package contenttools

import (
	"encoding/json"
	"regexp"
	"strings"
)

// lexToolFenceRE matches ```lex-tool ... ``` fences (CT body pointer blocks).
var lexToolFenceRE = regexp.MustCompile("(?s)```lex-tool\\s*\\n(.*?)\\n```")

// LexToolFencePayload is the JSON object inside a ```lex-tool fence (plan CT.2 FR-5).
type LexToolFencePayload struct {
	InstanceID string `json:"instanceId"`
	ToolID     string `json:"toolId"`
	V          int    `json:"v"`
}

// ParseLexToolFences extracts all ```lex-tool fence payloads from markdown.
// Invalid JSON fences are skipped (not included).
func ParseLexToolFences(markdown string) []LexToolFencePayload {
	if markdown == "" {
		return nil
	}
	matches := lexToolFenceRE.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]LexToolFencePayload, 0, len(matches))
	for _, sub := range matches {
		if len(sub) < 2 {
			continue
		}
		p, ok := parseLexToolPayload(sub[1])
		if !ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseLexToolPayload(raw string) (LexToolFencePayload, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return LexToolFencePayload{}, false
	}
	var p LexToolFencePayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return LexToolFencePayload{}, false
	}
	if strings.TrimSpace(p.InstanceID) == "" {
		return LexToolFencePayload{}, false
	}
	if p.V == 0 {
		p.V = 1
	}
	return p, true
}

// SerializeLexToolFence returns the exact fence text with stable key order
// {"instanceId":"...","toolId":"...","v":1} (plan CT.2 FR-5).
func SerializeLexToolFence(p LexToolFencePayload) string {
	if p.V == 0 {
		p.V = 1
	}
	b, err := json.Marshal(p)
	if err != nil {
		// Struct marshal cannot fail for this shape; keep a safe fallback.
		return "```lex-tool\n{}\n```"
	}
	return "```lex-tool\n" + string(b) + "\n```"
}

// StripInvalidLexToolFences removes fences whose instanceId is not in validInstanceIDs
// (or whose JSON payload is invalid). Valid fences are left byte-identical.
func StripInvalidLexToolFences(markdown string, validInstanceIDs map[string]bool) string {
	if markdown == "" {
		return markdown
	}
	if validInstanceIDs == nil {
		validInstanceIDs = map[string]bool{}
	}
	return lexToolFenceRE.ReplaceAllStringFunc(markdown, func(block string) string {
		sub := lexToolFenceRE.FindStringSubmatch(block)
		if len(sub) < 2 {
			return ""
		}
		p, ok := parseLexToolPayload(sub[1])
		if !ok || !validInstanceIDs[p.InstanceID] {
			return ""
		}
		return block
	})
}

// RewriteLexToolFences replaces instanceId values in ```lex-tool fences using idMap.
// Unmapped ids are left unchanged. Used by course copy (CT.1/CT.2).
func RewriteLexToolFences(markdown string, idMap map[string]string) string {
	if len(idMap) == 0 || markdown == "" {
		return markdown
	}
	return lexToolFenceRE.ReplaceAllStringFunc(markdown, func(block string) string {
		sub := lexToolFenceRE.FindStringSubmatch(block)
		if len(sub) < 2 {
			return block
		}
		p, ok := parseLexToolPayload(sub[1])
		if !ok {
			return block
		}
		newID, mapped := idMap[p.InstanceID]
		if !mapped || newID == "" {
			return block
		}
		p.InstanceID = newID
		return SerializeLexToolFence(p)
	})
}
