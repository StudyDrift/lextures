package analytics

import (
	"encoding/json"
)

func init() {
	RegisterProjector("class_pulse", projectClassPulse, []FacetSchema{
		{Key: "optionId", Label: "contentTools.analytics.facets.optionId", Type: "string"},
		{Key: "round2OptionId", Label: "contentTools.analytics.facets.round2OptionId", Type: "string"},
		{Key: "correct", Label: "contentTools.analytics.facets.correct", Type: "boolean"},
		{Key: "votedRounds", Label: "contentTools.analytics.facets.votedRounds", Type: "number"},
	})
}

func projectClassPulse(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Votes []struct {
			Round    int    `json:"round"`
			OptionID string `json:"optionId"`
		} `json:"votes"`
		CompletedAt string `json:"completedAt"`
		Correct     *bool  `json:"correct"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	rounds := 0
	var round1, round2 string
	for _, v := range st.Votes {
		if v.Round == 1 && v.OptionID != "" {
			round1 = v.OptionID
			rounds++
		}
		if v.Round == 2 && v.OptionID != "" {
			round2 = v.OptionID
			rounds++
		}
	}
	if rounds > 0 {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	if round1 != "" {
		s.Facets["optionId"] = round1
	}
	if round2 != "" {
		s.Facets["round2OptionId"] = round2
	}
	s.Facets["votedRounds"] = rounds
	if st.Correct != nil {
		s.Facets["correct"] = *st.Correct
	}
	return s
}
