// Package class_pulse is the CT.21 Class Pulse Content Tool.
package class_pulse

import (
	_ "embed"
	"encoding/json"
)

const ID = "class_pulse"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("class_pulse: invalid i18n/en.json: " + err.Error())
	}
}
