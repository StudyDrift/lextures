package contenttools

import (
	"encoding/json"
	"regexp"
	"strings"
)

// lexToolFenceRE matches ```lex-tool ... ``` fences (CT body pointer blocks).
var lexToolFenceRE = regexp.MustCompile("(?s)```lex-tool\\s*\\n(.*?)\\n```")

// RewriteLexToolFences replaces instanceId values in ```lex-tool fences using idMap.
// Unmapped ids are left unchanged. Used by course copy (CT.1).
func RewriteLexToolFences(markdown string, idMap map[string]string) string {
	if len(idMap) == 0 || markdown == "" {
		return markdown
	}
	return lexToolFenceRE.ReplaceAllStringFunc(markdown, func(block string) string {
		sub := lexToolFenceRE.FindStringSubmatch(block)
		if len(sub) < 2 {
			return block
		}
		raw := strings.TrimSpace(sub[1])
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return block
		}
		id, _ := payload["instanceId"].(string)
		if id == "" {
			return block
		}
		newID, ok := idMap[id]
		if !ok || newID == "" {
			return block
		}
		payload["instanceId"] = newID
		b, err := json.Marshal(payload)
		if err != nil {
			return block
		}
		return "```lex-tool\n" + string(b) + "\n```"
	})
}
