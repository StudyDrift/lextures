package httpserver

import (
	"testing"

	"github.com/google/uuid"
)

func TestCourseFileIDFromHeroURL(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	want := uuid.MustParse(id)
	cases := []struct {
		raw string
		ok  bool
	}{
		{raw: "/api/v1/courses/C-ABC/course-files/" + id + "/content", ok: true},
		{raw: "https://self.lextures.com/api/v1/courses/C-ABC/course-files/" + id + "/content", ok: true},
		{raw: "/api/v1/courses/C-ABC/course-files/" + id + "/content?w=1920&h=384", ok: true},
		{raw: "https://cdn.example.com/banner.jpg", ok: false},
		{raw: "/api/v1/courses/C-ABC/files/items/" + id + "/content", ok: false},
		{raw: "", ok: false},
	}
	for _, tc := range cases {
		got, ok := courseFileIDFromHeroURL(tc.raw)
		if ok != tc.ok {
			t.Fatalf("%q ok = %v want %v", tc.raw, ok, tc.ok)
		}
		if ok && got != want {
			t.Fatalf("%q id = %s", tc.raw, got)
		}
	}
}
