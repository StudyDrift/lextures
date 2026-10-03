package httpserver

import (
	"net/url"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/models/search"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

func managedLearnerQueryItem(hit managedlearners.SearchHit) search.QueryResultItem {
	subtitle := "Learner"
	if hit.GradeLevel != nil {
		grade := strings.TrimSpace(*hit.GradeLevel)
		if grade != "" {
			subtitle = "Learner · Grade " + grade
		}
	}
	id := hit.UserID.String()
	return search.QueryResultItem{
		ID:       "person:" + id + ":learner",
		Type:     "person",
		Title:    hit.DisplayName,
		Subtitle: subtitle,
		Path:     "/learners?learner=" + url.QueryEscape(id),
		Score:    hit.Score,
	}
}

func personResultUserID(id string) string {
	parts := strings.Split(id, ":")
	if len(parts) >= 2 && parts[0] == "person" {
		return parts[1]
	}
	return ""
}

// mergeSearchPersonResults appends managed-learner hits that are not already in the roster page.
// total is the roster match count plus managed learners that were not in that page.
func mergeSearchPersonResults(
	roster []search.QueryResultItem,
	rosterTotal int,
	extra []search.QueryResultItem,
	limit int,
) ([]search.QueryResultItem, int) {
	seen := map[string]struct{}{}
	for _, item := range roster {
		if uid := personResultUserID(item.ID); uid != "" {
			seen[uid] = struct{}{}
		}
	}
	merged := append([]search.QueryResultItem{}, roster...)
	added := 0
	for _, item := range extra {
		if uid := personResultUserID(item.ID); uid != "" {
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
		}
		merged = append(merged, item)
		added++
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].Score != merged[j].Score {
			return merged[i].Score > merged[j].Score
		}
		if merged[i].Title != merged[j].Title {
			return merged[i].Title < merged[j].Title
		}
		return merged[i].ID < merged[j].ID
	})
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	if merged == nil {
		merged = []search.QueryResultItem{}
	}
	if rosterTotal < 0 {
		rosterTotal = 0
	}
	return merged, rosterTotal + added
}
