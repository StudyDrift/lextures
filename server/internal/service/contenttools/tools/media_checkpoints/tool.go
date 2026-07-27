// Package media_checkpoints is the CT.19 Media Checkpoints Content Tool.
package media_checkpoints

import (
	_ "embed"
	"encoding/json"
)

const ID = "media_checkpoints"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("media_checkpoints: invalid i18n/en.json: " + err.Error())
	}
}
