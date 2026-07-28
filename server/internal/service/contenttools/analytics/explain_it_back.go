package analytics

import (
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/explain_it_back"
)

func init() {
	RegisterProjector("explain_it_back", projectExplainItBack, []FacetSchema{
		{Key: "keyPointId", Label: "contentTools.analytics.facets.keyPointId", Type: "string"},
		{Key: "covered", Label: "contentTools.analytics.facets.covered", Type: "boolean"},
		{Key: "attemptCount", Label: "contentTools.analytics.facets.attemptCount", Type: "number"},
		{Key: "reviewMode", Label: "contentTools.analytics.facets.reviewMode", Type: "boolean"},
	})
}

func projectExplainItBack(in ProjectInput) Summary {
	s := defaultProject(in)
	st := explain_it_back.ParseState(in.StateJSON)
	attemptCount := len(st.Attempts)
	if attemptCount > 0 || strings.TrimSpace(st.Draft) != "" {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" || attemptCount > 0 {
		s.Completed = true
	}

	coveredIDs := []string{}
	reviewMode := false
	if attemptCount > 0 {
		last := st.Attempts[attemptCount-1]
		if last.Feedback != nil {
			coveredIDs = append([]string{}, last.Feedback.Covered...)
			reviewMode = last.Feedback.Mode == explain_it_back.FeedbackModeReview
		}
	}

	// Facet keyPointId: covered ids from the latest attempt (CT.7 / FR-9).
	s.Facets["keyPointId"] = coveredIDs
	s.Facets["covered"] = len(coveredIDs) > 0
	s.Facets["attemptCount"] = attemptCount
	s.Facets["reviewMode"] = reviewMode
	return s
}
