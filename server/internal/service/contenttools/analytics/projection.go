package analytics

import (
	"encoding/json"
	"math"
	"strconv"
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

// HasProjector reports whether a tool registered an explicit summary projection (CT.8 gate).
func HasProjector(toolID string) bool {
	_, ok := projectors[toolID]
	return ok
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
	RegisterProjector("ask_questions", projectAskQuestions, []FacetSchema{
		{Key: "questionCount", Label: "contentTools.analytics.facets.questionCount", Type: "number"},
		{Key: "hasConversation", Label: "contentTools.analytics.facets.hasConversation", Type: "boolean"},
	})
	RegisterProjector("inline_questions", projectInlineQuestions, []FacetSchema{
		{Key: "questionId", Label: "contentTools.analytics.facets.questionId", Type: "string"},
		{Key: "optionId", Label: "contentTools.analytics.facets.optionId", Type: "string"},
		{Key: "correct", Label: "contentTools.analytics.facets.correct", Type: "boolean"},
	})
	RegisterProjector("predict_reveal", projectPredictReveal, []FacetSchema{
		{Key: "outcomeId", Label: "contentTools.analytics.facets.outcomeId", Type: "string"},
		{Key: "confidenceBucket", Label: "contentTools.analytics.facets.confidenceBucket", Type: "string"},
		{Key: "correct", Label: "contentTools.analytics.facets.correct", Type: "boolean"},
	})
	RegisterProjector("highlight_annotate", projectHighlightAnnotate, []FacetSchema{
		{Key: "tagId", Label: "contentTools.analytics.facets.tagId", Type: "string"},
		{Key: "unitIndex", Label: "contentTools.analytics.facets.unitIndex", Type: "string"},
	})
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

func projectAskQuestions(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Turns []struct {
			Role string `json:"role"`
		} `json:"turns"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	qCount := 0
	for _, t := range st.Turns {
		if t.Role == "user" {
			qCount++
		}
	}
	if qCount > 0 {
		s.Engaged = true
	}
	s.Facets["questionCount"] = qCount
	s.Facets["hasConversation"] = qCount > 0
	return s
}

func projectInlineQuestions(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Answers map[string]struct {
			Attempts []struct {
				Value   any  `json:"value"`
				Correct bool `json:"correct"`
			} `json:"attempts"`
		} `json:"answers"`
		ScoreRaw *float64 `json:"scoreRaw"`
		ScoreMax *float64 `json:"scoreMax"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	questionIDs := make([]string, 0, len(st.Answers))
	optionIDs := make([]string, 0, len(st.Answers))
	anyCorrect := false
	answered := 0
	for qid, ans := range st.Answers {
		if len(ans.Attempts) == 0 {
			continue
		}
		answered++
		questionIDs = append(questionIDs, qid)
		last := ans.Attempts[len(ans.Attempts)-1]
		if last.Correct {
			anyCorrect = true
		}
		switch v := last.Value.(type) {
		case string:
			optionIDs = append(optionIDs, qid+":"+v)
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					optionIDs = append(optionIDs, qid+":"+s)
				}
			}
		}
	}
	if answered > 0 {
		s.Engaged = true
	}
	s.Facets["questionId"] = questionIDs
	s.Facets["optionId"] = optionIDs
	if answered > 0 {
		s.Facets["correct"] = anyCorrect && in.ScoreRaw != nil && in.ScoreMax != nil && *in.ScoreMax > 0 && *in.ScoreRaw >= *in.ScoreMax
	}
	return s
}

func projectPredictReveal(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Prediction *struct {
			OutcomeID string `json:"outcomeId"`
			Text      string `json:"text"`
		} `json:"prediction"`
		ConfidenceBucket string `json:"confidenceBucket"`
		CommittedAt      string `json:"committedAt"`
		Correct          *bool  `json:"correct"`
		RevealedAt       string `json:"revealedAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	if st.CommittedAt != "" {
		s.Engaged = true
		s.Completed = true
	}
	if st.Prediction != nil && st.Prediction.OutcomeID != "" {
		s.Facets["outcomeId"] = st.Prediction.OutcomeID
	}
	if st.ConfidenceBucket != "" {
		s.Facets["confidenceBucket"] = st.ConfidenceBucket
	}
	if st.Correct != nil {
		s.Facets["correct"] = *st.Correct
	}
	if st.CommittedAt != "" && st.RevealedAt != "" {
		if start, err1 := time.Parse(time.RFC3339, st.CommittedAt); err1 == nil {
			if end, err2 := time.Parse(time.RFC3339, st.RevealedAt); err2 == nil && !end.Before(start) {
				d := int(end.Sub(start).Milliseconds())
				s.DurationMs = &d
			}
		}
	}
	return s
}

func projectHighlightAnnotate(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Annotations []struct {
			TagID   string `json:"tagId"`
			Orphaned bool  `json:"orphaned"`
			Anchor  struct {
				UnitIndex *int `json:"unitIndex"`
			} `json:"anchor"`
		} `json:"annotations"`
		CompletedAt string `json:"completedAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	tagIDs := make([]string, 0, len(st.Annotations))
	unitIndexes := make([]string, 0, len(st.Annotations))
	for _, a := range st.Annotations {
		if a.TagID != "" {
			tagIDs = append(tagIDs, a.TagID)
		}
		if a.Anchor.UnitIndex != nil {
			unitIndexes = append(unitIndexes, strconv.Itoa(*a.Anchor.UnitIndex))
		}
	}
	if len(st.Annotations) > 0 {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	s.Facets["tagId"] = tagIDs
	s.Facets["unitIndex"] = unitIndexes
	return s
}
