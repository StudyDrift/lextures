package httpserver

import "testing"

func TestGradingAgentCellPosting(t *testing.T) {
	tests := []struct {
		assign string
		agent  string
		want   string
	}{
		{"automatic", "auto_post", "automatic"},
		{"automatic", "draft", "manual"},
		{"automatic", "unposted", "manual"},
		{"manual", "auto_post", "manual"},
		{"", "auto_post", "manual"},
	}
	for _, tc := range tests {
		if got := gradingAgentCellPosting(tc.assign, tc.agent); got != tc.want {
			t.Fatalf("gradingAgentCellPosting(%q, %q) = %q, want %q", tc.assign, tc.agent, got, tc.want)
		}
	}
}