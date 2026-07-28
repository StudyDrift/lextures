// Package tools is the generated-style index of first-party Content Tools.
// Adding a tool = drop a folder here and register it in All() (CT.1 contract:
// no migration, no new route, no Deps change).
package tools

import (
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/ask_questions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/code_sandbox"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/diagram_hotspot"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/explain_it_back"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/highlight_annotate"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/media_checkpoints"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/noop_probe"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/parameter_explorer"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/predict_reveal"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sandbox_probe"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sort_sequence"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/worked_example"
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
			ID:           code_sandbox.ID,
			ManifestJSON: code_sandbox.ManifestJSON,
			I18nEN:       code_sandbox.I18nEN,
		},
		{
			ID:           diagram_hotspot.ID,
			ManifestJSON: diagram_hotspot.ManifestJSON,
			I18nEN:       diagram_hotspot.I18nEN,
		},
		{
			ID:           explain_it_back.ID,
			ManifestJSON: explain_it_back.ManifestJSON,
			I18nEN:       explain_it_back.I18nEN,
		},
		{
			ID:           highlight_annotate.ID,
			ManifestJSON: highlight_annotate.ManifestJSON,
			I18nEN:       highlight_annotate.I18nEN,
		},
		{
			ID:           inline_questions.ID,
			ManifestJSON: inline_questions.ManifestJSON,
			I18nEN:       inline_questions.I18nEN,
		},
		{
			ID:           media_checkpoints.ID,
			ManifestJSON: media_checkpoints.ManifestJSON,
			I18nEN:       media_checkpoints.I18nEN,
		},
		{
			ID:           noop_probe.ID,
			ManifestJSON: noop_probe.ManifestJSON,
			I18nEN:       noop_probe.I18nEN,
		},
		{
			ID:           parameter_explorer.ID,
			ManifestJSON: parameter_explorer.ManifestJSON,
			I18nEN:       parameter_explorer.I18nEN,
		},
		{
			ID:           predict_reveal.ID,
			ManifestJSON: predict_reveal.ManifestJSON,
			I18nEN:       predict_reveal.I18nEN,
		},
		{
			ID:           sandbox_probe.ID,
			ManifestJSON: sandbox_probe.ManifestJSON,
			I18nEN:       sandbox_probe.I18nEN,
		},
		{
			ID:           sort_sequence.ID,
			ManifestJSON: sort_sequence.ManifestJSON,
			I18nEN:       sort_sequence.I18nEN,
		},
		{
			ID:           worked_example.ID,
			ManifestJSON: worked_example.ManifestJSON,
			I18nEN:       worked_example.I18nEN,
		},
	}
}
