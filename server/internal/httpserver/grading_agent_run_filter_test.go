package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
)

func TestSectionAllowed(t *testing.T) {
	sec := uuid.New()
	if !sectionAllowed(nil, sec) {
		t.Fatal("nil visible should allow any section")
	}
	if !sectionAllowed([]uuid.UUID{}, sec) {
		t.Fatal("empty visible should allow any section")
	}
	if sectionAllowed([]uuid.UUID{uuid.New()}, sec) {
		t.Fatal("expected deny when section not in visible set")
	}
	if !sectionAllowed([]uuid.UUID{sec}, sec) {
		t.Fatal("expected allow when section in visible set")
	}
}

func TestFormatGraderAgentRunTargetSummary(t *testing.T) {
	section := "Section B"
	group := "Team Alpha"
	tests := []struct {
		name  string
		scope gradingagentrepo.RunScope
		meta  *graderAgentRunFilterContext
		count int
		want  string
	}{
		{
			name:  "ungraded section",
			scope: gradingagentrepo.RunScopeUngraded,
			meta:  &graderAgentRunFilterContext{SectionLabel: &section},
			count: 24,
			want:  "Ungraded in Section B: 24 submissions",
		},
		{
			name:  "all group",
			scope: gradingagentrepo.RunScopeAll,
			meta:  &graderAgentRunFilterContext{GroupLabel: &group},
			count: 5,
			want:  "All in Team Alpha: 5 submissions",
		},
		{
			name:  "no filter",
			scope: gradingagentrepo.RunScopeUngraded,
			meta:  nil,
			count: 3,
			want:  "Ungraded: 3 submissions",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatGraderAgentRunTargetSummary(tc.scope, tc.meta, tc.count); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestParseGraderAgentRunFilterBody(t *testing.T) {
	sid := uuid.New().String()
	gid := uuid.New().String()
	subID := uuid.New().String()
	filter, err := parseGraderAgentRunFilterBody(&graderAgentRunFilterBody{
		SectionID:     &sid,
		GroupID:       &gid,
		SubmissionIDs: []string{subID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filter == nil || filter.SectionID == nil || filter.GroupID == nil || len(filter.SubmissionIDs) != 1 {
		t.Fatalf("unexpected filter: %+v", filter)
	}
}

func TestGraderAgentRunFilters_DisabledReturns404(t *testing.T) {
	d := Deps{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/assignments/00000000-0000-0000-0000-000000000001/grader-agent/run-target", nil)
	rec := httptest.NewRecorder()
	d.handleGetGraderAgentRunTarget()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
