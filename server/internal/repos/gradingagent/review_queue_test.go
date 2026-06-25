package gradingagent

import "testing"

func TestReviewQueueLatestCTE_isStable(t *testing.T) {
	if reviewQueueLatestCTE == "" {
		t.Fatal("reviewQueueLatestCTE must be defined")
	}
	if !containsAll(reviewQueueLatestCTE, "DISTINCT ON (submission_id)", "is_dry_run = false", "created_at DESC") {
		t.Fatal("review queue CTE must dedupe by submission with latest created_at")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !contains(s, part) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}