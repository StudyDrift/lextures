// Package worked_example is the CT.18 step-through worked example Content Tool.
package worked_example

import (
	_ "embed"
	"encoding/json"
)

const ID = "worked_example"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("worked_example: invalid i18n/en.json: " + err.Error())
	}
}
