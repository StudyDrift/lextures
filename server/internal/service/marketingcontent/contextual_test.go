package marketingcontent

import (
	"testing"

	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

func TestNormalizeViewerRoles(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "empty defaults to learner", in: nil, want: []string{"learner"}},
		{name: "student maps to learner", in: []string{"student"}, want: []string{"student", "learner"}},
		{name: "teacher maps to instructor", in: []string{"teacher"}, want: []string{"teacher", "instructor"}},
		{name: "global admin maps to admin", in: []string{"Global Admin"}, want: []string{"Global Admin", "admin"}},
		{name: "dedupes aliases", in: []string{"student", "learner"}, want: []string{"student", "learner"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeViewerRoles(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestFilterByRole_StudentSeesLearnerArticles(t *testing.T) {
	t.Parallel()
	arts := []repo.PublicArticle{
		{Article: repo.Article{Slug: "for-learners", Roles: []string{"learner", "instructor"}}},
		{Article: repo.Article{Slug: "instructor-only", Roles: []string{"instructor"}}},
		{Article: repo.Article{Slug: "anyone"}},
	}
	roles := NormalizeViewerRoles([]string{"student"})
	got := filterByRole(arts, roles)
	if len(got) != 2 {
		t.Fatalf("expected learner + unrestricted, got %#v", got)
	}
	if got[0].Slug != "for-learners" || got[1].Slug != "anyone" {
		t.Fatalf("unexpected order/slugs: %#v", got)
	}
}

func TestRouteKeywordsDropsIDs(t *testing.T) {
	t.Parallel()
	if got := routeKeywords("/courses/123/modules"); got != "courses modules" {
		t.Fatalf("got %q", got)
	}
	if got := routeKeywords("/courses/abc123/modules"); got != "courses abc123 modules" {
		t.Fatalf("got %q", got)
	}
	if got := routeKeywords("/dashboard"); got != "dashboard" {
		t.Fatalf("got %q", got)
	}
}
