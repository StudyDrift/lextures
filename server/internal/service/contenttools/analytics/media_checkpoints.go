package analytics

import (
	"encoding/json"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/media_checkpoints"
)

func init() {
	RegisterProjector("media_checkpoints", projectMediaCheckpoints, []FacetSchema{
		{Key: "checkpointId", Label: "contentTools.analytics.facets.checkpointId", Type: "string"},
		{Key: "correct", Label: "contentTools.analytics.facets.correct", Type: "boolean"},
		{Key: "checkpointResult", Label: "contentTools.analytics.facets.checkpointResult", Type: "string"},
		{Key: "watchedBin", Label: "contentTools.analytics.facets.watchedBin", Type: "string"},
		{Key: "usedTranscriptOnly", Label: "contentTools.analytics.facets.usedTranscriptOnly", Type: "boolean"},
	})
}

func projectMediaCheckpoints(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Answers map[string]struct {
			Attempts []struct {
				Correct bool `json:"correct"`
			} `json:"attempts"`
			Done bool `json:"done"`
		} `json:"answers"`
		WatchedSegments    [][2]float64 `json:"watchedSegments"`
		FurthestSec        float64      `json:"furthestSec"`
		UsedTranscriptOnly bool         `json:"usedTranscriptOnly"`
		CompletedAt        string       `json:"completedAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)

	checkpointIDs := make([]string, 0)
	results := make([]string, 0)
	anyCorrect := false
	engaged := st.UsedTranscriptOnly || len(st.WatchedSegments) > 0 || st.FurthestSec > 0
	for cid, ans := range st.Answers {
		if len(ans.Attempts) == 0 && !ans.Done {
			continue
		}
		engaged = true
		checkpointIDs = append(checkpointIDs, cid)
		lastCorrect := false
		if len(ans.Attempts) > 0 {
			lastCorrect = ans.Attempts[len(ans.Attempts)-1].Correct
		}
		if lastCorrect {
			anyCorrect = true
			results = append(results, cid+":true")
		} else {
			results = append(results, cid+":false")
		}
	}
	if engaged {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	bins := media_checkpoints.WatchedBins(st.WatchedSegments, media_checkpoints.DefaultGranularitySec)
	s.Facets["checkpointId"] = checkpointIDs
	s.Facets["correct"] = anyCorrect
	s.Facets["checkpointResult"] = results
	s.Facets["watchedBin"] = bins
	s.Facets["usedTranscriptOnly"] = st.UsedTranscriptOnly
	return s
}
