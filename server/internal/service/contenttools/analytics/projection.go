package analytics

import (
	"encoding/json"
	"math"
	"strings"
	"time"
)

// DefaultSmallN is the default suppression threshold (FR-6). Override per org when configured.
const DefaultSmallN = 5

// ProjectionVersion bumps when a tool's Project function changes; triggers rebuild.
const ProjectionVersion = 1

// FacetSchema describes aggregatable dimensions for a tool.
type FacetSchema struct {
	Key   string `json:"key"`
	Label string `json:"label"` // i18n key
	Type  string `json:"type"`  // string | number | boolean
}

// Summary is the typed projection of one learner's state_json (FR-1).
type Summary struct {
	Engaged            bool                   `json:"engaged"`
	Completed          bool                   `json:"completed"`
	ScorePct           *float64               `json:"scorePct,omitempty"`
	DurationMs         *int                   `json:"durationMs,omitempty"`
	Facets             map[string]any         `json:"facets"`
	ProjectionVersion  int                    `json:"projectionVersion"`
}

// ProjectInput is everything a projection needs from persisted state.
type ProjectInput struct {
	ToolID            string
	StateJSON         json.RawMessage
	Status            string
	ScoreRaw          *float64
	ScoreMax          *float64
	FirstInteractedAt *time.Time
	LastInteractedAt  *time.Time
	CompletedAt       *time.Time
}

// Projector projects state to a Summary.
type Projector func(in ProjectInput) Summary

var projectors = map[string]Projector{}

var facetSchemas = map[string][]FacetSchema{}

// RegisterProjector registers a tool's summary projection and facet schema.
func RegisterProjector(toolID string, p Projector, facets []FacetSchema) {
	if toolID == "" || p == nil {
		return
	}
	projectors[toolID] = p
	if facets == nil {
		facets = []FacetSchema{}
	}
	facetSchemas[toolID] = facets
}

// FacetsForTool returns the facet schema for a tool (may be empty).
func FacetsForTool(toolID string) []FacetSchema {
	if s, ok := facetSchemas[toolID]; ok {
		out := make([]FacetSchema, len(s))
		copy(out, s)
		return out
	}
	return []FacetSchema{}
}

// Project computes a Summary for the given tool state.
func Project(in ProjectInput) Summary {
	if p, ok := projectors[in.ToolID]; ok {
		s := p(in)
		if s.Facets == nil {
			s.Facets = map[string]any{}
		}
		if s.ProjectionVersion == 0 {
			s.ProjectionVersion = ProjectionVersion
		}
		return s
	}
	return defaultProject(in)
}

func defaultProject(in ProjectInput) Summary {
	engaged := in.Status != "" && in.Status != "not_started"
	completed := in.Status == "submitted" || in.Status == "completed"
	var scorePct *float64
	if in.ScoreRaw != nil && in.ScoreMax != nil && *in.ScoreMax > 0 {
		v := math.Round((*in.ScoreRaw / *in.ScoreMax) * 10000) / 100
		scorePct = &v
	}
	var durationMs *int
	if in.FirstInteractedAt != nil {
		end := in.LastInteractedAt
		if in.CompletedAt != nil {
			end = in.CompletedAt
		}
		if end != nil && !end.Before(*in.FirstInteractedAt) {
			d := int(end.Sub(*in.FirstInteractedAt).Milliseconds())
			durationMs = &d
		}
	}
	return Summary{
		Engaged:           engaged,
		Completed:         completed,
		ScorePct:          scorePct,
		DurationMs:        durationMs,
		Facets:            map[string]any{},
		ProjectionVersion: ProjectionVersion,
	}
}

func init() {
	RegisterProjector("noop_probe", projectNoopProbe, []FacetSchema{
		{Key: "attempts", Label: "contentTools.analytics.facets.attempts", Type: "number"},
		{Key: "correct", Label: "contentTools.analytics.facets.correct", Type: "boolean"},
		{Key: "hasResponse", Label: "contentTools.analytics.facets.hasResponse", Type: "boolean"},
	})
	RegisterProjector("sandbox_probe", defaultProject, nil)
}

func projectNoopProbe(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Response string `json:"response"`
		Attempts int    `json:"attempts"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	hasResp := strings.TrimSpace(st.Response) != ""
	if hasResp || st.Attempts > 0 {
		s.Engaged = true
	}
	s.Facets["attempts"] = st.Attempts
	s.Facets["hasResponse"] = hasResp
	correct := in.ScoreRaw != nil && in.ScoreMax != nil && *in.ScoreMax > 0 && *in.ScoreRaw >= *in.ScoreMax
	s.Facets["correct"] = correct
	return s
}
