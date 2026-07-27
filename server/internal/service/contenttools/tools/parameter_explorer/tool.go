package parameter_explorer

import (
	_ "embed"
	"encoding/json"
)

const ID = "parameter_explorer"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("parameter_explorer: invalid i18n/en.json: " + err.Error())
	}
}
