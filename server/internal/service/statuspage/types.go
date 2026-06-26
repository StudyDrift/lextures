package statuspage

// Summary is the normalized response for GET /api/v1/status-summary.
type Summary struct {
	PageURL    string            `json:"pageUrl"`
	Status     string            `json:"status"`
	Incidents  []SummaryIncident `json:"incidents"`
	Configured bool              `json:"configured"`
}

// SummaryIncident is an active incident surfaced to the web app banner.
type SummaryIncident struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Impact string `json:"impact"`
}

// upstreamSummary mirrors Statuspage GET /pages/{id}/summary.json.
type upstreamSummary struct {
	Page struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"page"`
	Incidents []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Impact string `json:"impact"`
	} `json:"incidents"`
	ScheduledMaintenances []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Status      string `json:"status"`
		Impact      string `json:"impact"`
		ScheduledFor string `json:"scheduled_for"`
	} `json:"scheduled_maintenances"`
}