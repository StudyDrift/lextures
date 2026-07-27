// Package inline_questions is the CT.11 formative check Content Tool.
package inline_questions

import (
	_ "embed"
	"encoding/json"
)

const ID = "inline_questions"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("inline_questions: invalid i18n/en.json: " + err.Error())
	}
}
