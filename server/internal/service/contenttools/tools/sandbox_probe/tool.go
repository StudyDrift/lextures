// Package sandbox_probe is the CT.5 iframe-sandbox canary tool.
package sandbox_probe

import (
	_ "embed"
	"encoding/json"
)

const ID = "sandbox_probe"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("sandbox_probe: invalid i18n/en.json: " + err.Error())
	}
}
