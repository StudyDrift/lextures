// Package highlight_annotate is the CT.13 Highlight & Annotate Content Tool.
package highlight_annotate

import (
	_ "embed"
	"encoding/json"
)

const ID = "highlight_annotate"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("highlight_annotate: invalid i18n/en.json: " + err.Error())
	}
}
