// Package predict_reveal is the CT.12 Predict & Reveal Content Tool.
package predict_reveal

import (
	_ "embed"
	"encoding/json"
)

const ID = "predict_reveal"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("predict_reveal: invalid i18n/en.json: " + err.Error())
	}
}
