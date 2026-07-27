// Package sort_sequence is the CT.14 Sort & Sequence Content Tool.
package sort_sequence

import (
	_ "embed"
	"encoding/json"
)

const ID = "sort_sequence"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("sort_sequence: invalid i18n/en.json: " + err.Error())
	}
}
