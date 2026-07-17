package engine

import (
	"math"
	"sort"
	"strconv"
)

// TeamMemberScore is one player's contribution for aggregation.
type TeamMemberScore struct {
	PlayerID   string
	TeamID     string
	TotalScore int
	ResponseMs int // sum of response_ms for tie-break
}

// TeamLeaderboardEntry is one ranked team row.
type TeamLeaderboardEntry struct {
	Rank       int
	TeamID     string
	Name       string
	Color      string
	Score      int // aggregated (rounded for average)
	MemberCount int
	RawSum     int
}

// AggregateTeamScores ranks teams by sum or average of member total_score (AC-1).
// Tie-break: lower total response_ms sum, then team name ascending.
func AggregateTeamScores(
	members []TeamMemberScore,
	teamMeta map[string]struct{ Name, Color string },
	agg TeamAggregate,
) []TeamLeaderboardEntry {
	type acc struct {
		sum        int
		ms         int
		n          int
		name       string
		color      string
	}
	byTeam := map[string]*acc{}
	for _, m := range members {
		if m.TeamID == "" {
			continue
		}
		a := byTeam[m.TeamID]
		if a == nil {
			meta := teamMeta[m.TeamID]
			a = &acc{name: meta.Name, color: meta.Color}
			byTeam[m.TeamID] = a
		}
		a.sum += m.TotalScore
		a.ms += m.ResponseMs
		a.n++
	}
	out := make([]TeamLeaderboardEntry, 0, len(byTeam))
	for id, a := range byTeam {
		score := a.sum
		if agg == TeamAggregateAverage {
			if a.n == 0 {
				score = 0
			} else {
				score = int(math.Round(float64(a.sum) / float64(a.n)))
			}
		}
		out = append(out, TeamLeaderboardEntry{
			TeamID:      id,
			Name:        a.name,
			Color:       a.color,
			Score:       score,
			MemberCount: a.n,
			RawSum:      a.sum,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		// Prefer fewer total response ms among members (fairer for speed).
		ai := byTeam[out[i].TeamID]
		aj := byTeam[out[j].TeamID]
		if ai != nil && aj != nil && ai.ms != aj.ms {
			return ai.ms < aj.ms
		}
		return out[i].Name < out[j].Name
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

// AutoBalanceAssign round-robins player IDs across team IDs (host-controlled).
func AutoBalanceAssign(playerIDs, teamIDs []string) map[string]string {
	out := make(map[string]string, len(playerIDs))
	if len(teamIDs) == 0 {
		return out
	}
	for i, pid := range playerIDs {
		out[pid] = teamIDs[i%len(teamIDs)]
	}
	return out
}

// DefaultTeamNames returns "Team A", "Team B", …
func DefaultTeamNames(n int) []string {
	if n < 1 {
		n = 1
	}
	names := make([]string, n)
	for i := 0; i < n; i++ {
		if i < 26 {
			names[i] = "Team " + string(rune('A'+i))
		} else {
			names[i] = "Team " + strconv.Itoa(i+1)
		}
	}
	return names
}

// DefaultTeamColors pairs names with accessible palette (not colour-only in UI).
func DefaultTeamColors(n int) []string {
	palette := []string{"#2563eb", "#dc2626", "#16a34a", "#ca8a04", "#9333ea", "#0891b2", "#ea580c", "#4f46e5"}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = palette[i%len(palette)]
	}
	return out
}
