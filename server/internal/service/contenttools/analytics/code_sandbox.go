package analytics

import (
	"encoding/json"
	"strings"
)

func init() {
	RegisterProjector("code_sandbox", projectCodeSandbox, []FacetSchema{
		{Key: "testId", Label: "contentTools.analytics.facets.testId", Type: "string"},
		{Key: "passed", Label: "contentTools.analytics.facets.passed", Type: "boolean"},
		{Key: "runCount", Label: "contentTools.analytics.facets.runCount", Type: "number"},
	})
}

func projectCodeSandbox(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		Code string `json:"code"`
		Runs []struct {
			Action string `json:"action"`
			Tests  []struct {
				ID     string `json:"id"`
				Passed bool   `json:"passed"`
			} `json:"tests"`
		} `json:"runs"`
		Best *struct {
			Passed int `json:"passed"`
			Total  int `json:"total"`
		} `json:"best"`
		CompletedAt string `json:"completedAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	if strings.TrimSpace(st.Code) != "" || len(st.Runs) > 0 {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" || in.Status == "submitted" {
		s.Completed = true
	}

	testIDs := make([]string, 0)
	anyFail := false
	anyPass := false
	for i := len(st.Runs) - 1; i >= 0; i-- {
		r := st.Runs[i]
		if r.Action != "check" || len(r.Tests) == 0 {
			continue
		}
		for _, tr := range r.Tests {
			testIDs = appendUnique(testIDs, tr.ID)
			if tr.Passed {
				anyPass = true
			} else {
				anyFail = true
			}
		}
		break
	}
	s.Facets["testId"] = testIDs
	s.Facets["runCount"] = len(st.Runs)
	if anyPass && !anyFail {
		s.Facets["passed"] = true
	} else if anyFail || anyPass {
		s.Facets["passed"] = anyPass && !anyFail
	}
	if st.Best != nil && st.Best.Total > 0 {
		pct := float64(st.Best.Passed) / float64(st.Best.Total) * 100
		s.ScorePct = &pct
		s.Facets["passed"] = st.Best.Passed >= st.Best.Total
	}
	return s
}
