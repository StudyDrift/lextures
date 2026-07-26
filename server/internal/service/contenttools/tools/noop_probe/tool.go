// Package noop_probe is the CT.1 test-only built-in tool used to exercise the
// registry and config validation. It is never offered in production UIs beyond
// the catalog when Content Tools is enabled (authors should prefer real tools).
package noop_probe

import (
	_ "embed"
	"encoding/json"
)

const ID = "noop_probe"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool (required by FR-4).
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("noop_probe: invalid i18n/en.json: " + err.Error())
	}
}
