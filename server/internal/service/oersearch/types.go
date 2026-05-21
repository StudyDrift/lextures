package oersearch

// Result is a normalized OER search hit returned to the API.
type Result struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	PreviewURL    string `json:"previewUrl,omitempty"`
	Provider      string `json:"provider"`
	LicenseSPDX   string `json:"licenseSpdx"`
	LicenseLabel  string `json:"licenseLabel"`
	GradeLevel    string `json:"gradeLevel,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Attribution   string `json:"attribution,omitempty"`
}

// SearchParams are query inputs shared across providers.
type SearchParams struct {
	Query    string
	Subject  string
	Level    string
	License  string // filter token, e.g. "CC-BY"
	Provider string // empty = all enabled
}

// SearchResponse is the API payload for a search.
type SearchResponse struct {
	Results    []Result `json:"results"`
	Provider   string   `json:"provider,omitempty"`
	FromCache  bool     `json:"fromCache"`
	CacheAsOf  string   `json:"cacheAsOf,omitempty"`
	StaleCache bool     `json:"staleCache,omitempty"`
}
