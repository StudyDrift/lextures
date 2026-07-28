package analytics

import (
	"encoding/json"
)

func init() {
	RegisterProjector("flashcards", projectFlashcards, []FacetSchema{
		{Key: "cardId", Label: "contentTools.analytics.facets.cardId", Type: "string"},
		{Key: "firstRating", Label: "contentTools.analytics.facets.firstRating", Type: "string"},
		{Key: "ratedCards", Label: "contentTools.analytics.facets.ratedCards", Type: "number"},
		{Key: "firstPassComplete", Label: "contentTools.analytics.facets.firstPassComplete", Type: "boolean"},
	})
}

func projectFlashcards(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Cards map[string]struct {
			Seen        int     `json:"seen"`
			FirstRating *string `json:"firstRating"`
			LastRating  *string `json:"lastRating"`
		} `json:"cards"`
		FirstPassCompletedAt string `json:"firstPassCompletedAt"`
		ActiveSession        *struct {
			Reviewed int `json:"reviewed"`
		} `json:"activeSession"`
		Sessions []struct {
			Reviewed int `json:"reviewed"`
		} `json:"sessions"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)

	rated := 0
	var hardestCard string
	hardestRank := 99
	rank := map[string]int{"again": 0, "hard": 1, "good": 2, "easy": 3}
	var sampleFirst string
	for id, c := range st.Cards {
		if c.Seen > 0 {
			rated++
			s.Engaged = true
		}
		if c.FirstRating != nil {
			sampleFirst = *c.FirstRating
			if r, ok := rank[*c.FirstRating]; ok && r < hardestRank {
				hardestRank = r
				hardestCard = id
			}
		}
	}
	if st.FirstPassCompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	if len(st.Sessions) > 0 || (st.ActiveSession != nil && st.ActiveSession.Reviewed > 0) {
		s.Engaged = true
	}
	s.Facets["ratedCards"] = rated
	s.Facets["firstPassComplete"] = s.Completed
	if hardestCard != "" {
		s.Facets["cardId"] = hardestCard
		if sample := st.Cards[hardestCard].FirstRating; sample != nil {
			s.Facets["firstRating"] = *sample
		}
	} else if sampleFirst != "" {
		s.Facets["firstRating"] = sampleFirst
	}
	return s
}
