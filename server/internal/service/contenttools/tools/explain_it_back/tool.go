// Package explain_it_back is the CT.20 self-explanation Content Tool.
package explain_it_back

import (
	_ "embed"
	"encoding/json"
)

const ID = "explain_it_back"

// FeatureID is the aigateway feature id for Explain It Back calls.
const FeatureID = "content_tool_explain_back"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("explain_it_back: invalid i18n/en.json: " + err.Error())
	}
}
