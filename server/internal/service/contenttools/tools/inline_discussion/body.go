package inline_discussion

import (
	"encoding/json"
	"strings"
)

// TipTapDocFromText builds a TipTap doc carrying plain text and optional lex meta.
func TipTapDocFromText(text string, meta *PostMeta) json.RawMessage {
	text = strings.TrimSpace(text)
	doc := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	}
	if meta != nil {
		doc["attrs"] = map[string]any{"lex": meta}
	}
	raw, _ := json.Marshal(doc)
	return raw
}

// TextFromTipTap extracts plain text from a TipTap doc.
func TextFromTipTap(body json.RawMessage) string {
	if len(body) == 0 {
		return ""
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return ""
	}
	var b strings.Builder
	walkText(root["content"], &b)
	return strings.TrimSpace(b.String())
}

func walkText(node any, b *strings.Builder) {
	switch v := node.(type) {
	case []any:
		for _, child := range v {
			walkText(child, b)
		}
	case map[string]any:
		if t, ok := v["type"].(string); ok && t == "text" {
			if s, ok := v["text"].(string); ok {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(s)
			}
			return
		}
		walkText(v["content"], b)
	}
}

// MetaFromTipTap reads attrs.lex from a TipTap doc.
func MetaFromTipTap(body json.RawMessage) PostMeta {
	var root struct {
		Attrs struct {
			Lex PostMeta `json:"lex"`
		} `json:"attrs"`
	}
	_ = json.Unmarshal(body, &root)
	return root.Attrs.Lex
}

// WithMeta returns a TipTap body with updated lex meta, preserving text.
func WithMeta(body json.RawMessage, meta PostMeta) json.RawMessage {
	text := TextFromTipTap(body)
	return TipTapDocFromText(text, &meta)
}
