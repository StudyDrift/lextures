// Package inline_discussion is the CT.22 Inline Discussion Content Tool.
package inline_discussion

import (
	_ "embed"
	"encoding/json"
)

const ID = "inline_discussion"

// HiddenForumName is the course forum that backs all inline discussion threads.
// It is filtered from the public forum index.
const HiddenForumName = "__ct_inline_discussion__"

// ThreadTitlePrefix identifies threads owned by a content-tool instance.
const ThreadTitlePrefix = "ct.inline:"

//go:embed manifest.json
var ManifestJSON []byte

//go:embed i18n/en.json
var i18nENRaw []byte

// I18nEN is the English bundle for this tool.
var I18nEN map[string]string

func init() {
	if err := json.Unmarshal(i18nENRaw, &I18nEN); err != nil {
		panic("inline_discussion: invalid i18n/en.json: " + err.Error())
	}
}

// ThreadTitleForInstance returns the discussion thread title for an instance.
func ThreadTitleForInstance(instanceID string) string {
	return ThreadTitlePrefix + instanceID
}
