// Package tools is the generated-style index of first-party Content Tools.
// Adding a tool = drop a folder here and register it in All() (CT.1 contract:
// no migration, no new route, no Deps change).
package tools

import (
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/ask_questions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/noop_probe"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sandbox_probe"
)

// Entry is one built-in tool's embeddable assets.
type Entry struct {
	ID           string
	ManifestJSON []byte
	I18nEN       map[string]string
}

// All returns every first-party tool. Keep sorted by id.
func All() []Entry {
	return []Entry{
		{
			ID:           ask_questions.ID,
			ManifestJSON: ask_questions.ManifestJSON,
			I18nEN:       ask_questions.I18nEN,
		},
		{
			ID:           inline_questions.ID,
			ManifestJSON: inline_questions.ManifestJSON,
			I18nEN:       inline_questions.I18nEN,
		},
		{
			ID:           noop_probe.ID,
			ManifestJSON: noop_probe.ManifestJSON,
			I18nEN:       noop_probe.I18nEN,
		},
		{
			ID:           sandbox_probe.ID,
			ManifestJSON: sandbox_probe.ManifestJSON,
			I18nEN:       sandbox_probe.I18nEN,
		},
	}
}
