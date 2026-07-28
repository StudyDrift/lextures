// Package flashcards is the CT.23 Flashcards & Spaced Recall Content Tool.
package flashcards

import (
	_ "embed"
	"encoding/json"
)

const ID = "flashcards"

// Source marks synthetic question-bank rows that back deck cards for SRS.
const Source = "content_tool_flashcards"

// SideForward / SideReverse identify independent SRS items per card.
const (
	SideForward = "forward"
	SideReverse = "reverse"
)

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("flashcards: invalid i18n/en.json: " + err.Error())
	}
}
