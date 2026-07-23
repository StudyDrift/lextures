package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWantsSameOriginFileBody(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mod  func(*http.Request)
		want bool
	}{
		{
			name: "plain get redirects ok",
			mod:  func(r *http.Request) {},
			want: false,
		},
		{
			name: "cors alone still allows redirect (large media)",
			mod: func(r *http.Request) {
				r.Header.Set("Sec-Fetch-Mode", "cors")
			},
			want: false,
		},
		{
			name: "prefer representation",
			mod: func(r *http.Request) {
				r.Header.Set("Prefer", "return=representation")
			},
			want: true,
		},
		{
			name: "inline query",
			mod: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("inline", "1")
				r.URL.RawQuery = q.Encode()
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-X/course-files/00000000-0000-0000-0000-000000000001/content", nil)
			tc.mod(r)
			if got := wantsSameOriginFileBody(r); got != tc.want {
				t.Fatalf("wantsSameOriginFileBody = %v want %v", got, tc.want)
			}
		})
	}
}
