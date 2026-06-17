package openrouter

// UsageInfo is token and cost metadata from an OpenRouter chat completion response.
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
}

// HasData reports whether the provider returned any usage metadata.
func (u UsageInfo) HasData() bool {
	return u.TotalTokens > 0 || u.PromptTokens > 0 || u.CompletionTokens > 0 || u.CostUSD > 0
}

// ChatResult is the assistant text plus optional usage metadata.
type ChatResult struct {
	Text  string
	Usage UsageInfo
}