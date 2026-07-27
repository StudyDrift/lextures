// Package code_sandbox is the CT.17 Code Sandbox Content Tool.
package code_sandbox

import (
	_ "embed"
	"encoding/json"
)

const ID = "code_sandbox"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("code_sandbox: invalid i18n/en.json: " + err.Error())
	}
}
