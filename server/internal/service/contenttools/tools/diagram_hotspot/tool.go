// Package diagram_hotspot is the CT.15 Labeled Diagram & Hotspot Content Tool.
package diagram_hotspot

import (
	_ "embed"
	"encoding/json"
)

const ID = "diagram_hotspot"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("diagram_hotspot: invalid i18n/en.json: " + err.Error())
	}
}
