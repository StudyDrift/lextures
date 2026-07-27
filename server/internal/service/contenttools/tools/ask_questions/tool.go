// Package ask_questions is the CT.10 grounded Q&A Content Tool.
package ask_questions

import (
	_ "embed"
	"encoding/json"
)

const ID = "ask_questions"

// FeatureID is the aigateway feature id for Ask Questions calls.
const FeatureID = "content_tool_ask"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("ask_questions: invalid i18n/en.json: " + err.Error())
	}
}
