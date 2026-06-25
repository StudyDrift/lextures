package httpserver

import "strings"

// gradingAgentCellPosting returns the gradebook posting policy for an agent-written cell.
// Grades stay unposted (manual) unless the assignment posts automatically and the agent is
// configured for auto_post.
func gradingAgentCellPosting(assignPostingPolicy, agentPostPolicy string) string {
	posting := "manual"
	policy := strings.TrimSpace(agentPostPolicy)
	if policy == "" || policy == "unposted" || policy == "draft" {
		policy = "draft"
	}
	if strings.TrimSpace(assignPostingPolicy) == "automatic" && policy == "auto_post" {
		posting = "automatic"
	}
	return posting
}