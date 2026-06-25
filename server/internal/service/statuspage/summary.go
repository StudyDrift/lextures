package statuspage

import "strings"

func emptySummary(pageURL string, configured bool) Summary {
	return Summary{
		PageURL:    pageURL,
		Status:     "none",
		Incidents:  []SummaryIncident{},
		Configured: configured,
	}
}

func normalizeSummary(pageURL string, raw upstreamSummary) Summary {
	out := Summary{
		PageURL:    pageURL,
		Status:     "none",
		Incidents:  make([]SummaryIncident, 0),
		Configured: true,
	}
	if u := strings.TrimSpace(raw.Page.URL); u != "" {
		out.PageURL = u
	}
	maxImpact := "none"
	for _, inc := range raw.Incidents {
		if !isActiveIncident(inc.Status) {
			continue
		}
		out.Incidents = append(out.Incidents, SummaryIncident{
			ID:     inc.ID,
			Name:   inc.Name,
			Status: inc.Status,
			Impact: inc.Impact,
		})
		maxImpact = maxImpactRank(maxImpact, inc.Impact)
	}
	for _, maint := range raw.ScheduledMaintenances {
		if !isActiveMaintenance(maint.Status) {
			continue
		}
		out.Incidents = append(out.Incidents, SummaryIncident{
			ID:     maint.ID,
			Name:   maint.Name,
			Status: maint.Status,
			Impact: maint.Impact,
		})
		maxImpact = maxImpactRank(maxImpact, maint.Impact)
	}
	out.Status = maxImpact
	return out
}

func isActiveIncident(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "postmortem":
		return false
	default:
		return status != ""
	}
}

func isActiveMaintenance(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return false
	default:
		return status != ""
	}
}

func maxImpactRank(current, next string) string {
	rank := func(v string) int {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "critical":
			return 4
		case "major":
			return 3
		case "minor":
			return 2
		case "none", "":
			return 0
		default:
			return 1
		}
	}
	if rank(next) > rank(current) {
		return strings.ToLower(strings.TrimSpace(next))
	}
	return current
}