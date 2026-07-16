package board

import (
	"encoding/json"

	"github.com/microcosm-cc/bluemonday"
)

// Strict subset for board post bodies: bold, italic, lists, links, code (FR-12).
var postBodyPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "ul", "ol", "li", "a", "code", "pre")
	p.AllowAttrs("href").OnElements("a")
	p.RequireParseableURLs(true)
	p.AllowURLSchemes("http", "https", "mailto")
	return p
}()

// SanitizePostHTML strips unsafe markup from a board post body.
func SanitizePostHTML(html string) string {
	return postBodyPolicy.Sanitize(html)
}

// NormalizeBody sanitizes a JSON body document. Accepted shapes:
//   - null / empty → nil
//   - string → {"html": sanitized} or {"text": plain} when no tags
//   - object with "html" and/or "text" fields
func NormalizeBody(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		clean := SanitizePostHTML(asString)
		if clean == "" && asString != "" {
			// Plain text without tags — keep as text.
			out, err := json.Marshal(map[string]string{"text": asString})
			return out, err
		}
		out, err := json.Marshal(map[string]string{"html": clean, "text": stripTagsApprox(clean)})
		return out, err
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if html, ok := obj["html"].(string); ok {
		obj["html"] = SanitizePostHTML(html)
	}
	if text, ok := obj["text"].(string); ok {
		// Text is plain; strip any accidental tags.
		obj["text"] = postBodyPolicy.Sanitize(text)
	}
	return json.Marshal(obj)
}

func stripTagsApprox(html string) string {
	return bluemonday.StrictPolicy().Sanitize(html)
}
