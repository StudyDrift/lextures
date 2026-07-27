package analytics

import (
	"encoding/json"
)

func init() {
	RegisterProjector("worked_example", projectWorkedExample, []FacetSchema{
		{Key: "stepId", Label: "contentTools.analytics.facets.stepId", Type: "string"},
		{Key: "result", Label: "contentTools.analytics.facets.result", Type: "string"},
		{Key: "hintsUsed", Label: "contentTools.analytics.facets.hintsUsed", Type: "number"},
		{Key: "revealed", Label: "contentTools.analytics.facets.revealed", Type: "boolean"},
	})
}

func projectWorkedExample(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Steps map[string]struct {
			Attempts []struct {
				Result string `json:"result"`
			} `json:"attempts"`
			HintsUsed   int    `json:"hintsUsed"`
			Revealed    bool   `json:"revealed"`
			CompletedAt string `json:"completedAt"`
		} `json:"steps"`
		CompletedAt string `json:"completedAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)

	stepIDs := make([]string, 0, len(st.Steps))
	results := make([]string, 0, len(st.Steps))
	hintsTotal := 0
	anyRevealed := false
	engaged := false
	for sid, sp := range st.Steps {
		if len(sp.Attempts) == 0 && sp.HintsUsed == 0 && !sp.Revealed {
			continue
		}
		engaged = true
		stepIDs = append(stepIDs, sid)
		hintsTotal += sp.HintsUsed
		if sp.Revealed {
			anyRevealed = true
			results = append(results, sid+":revealed")
		} else if len(sp.Attempts) > 0 {
			last := sp.Attempts[len(sp.Attempts)-1]
			results = append(results, sid+":"+last.Result)
		}
	}
	if engaged {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	s.Facets["stepId"] = stepIDs
	s.Facets["result"] = results
	s.Facets["hintsUsed"] = hintsTotal
	s.Facets["revealed"] = anyRevealed
	return s
}
