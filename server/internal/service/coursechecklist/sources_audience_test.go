package coursechecklist

import (
	"reflect"
	"testing"
)

func TestSourcesForAudience_HidesQmOscqrForHomeschoolAndK12(t *testing.T) {
	in := []string{"QM 1.1", "OSCQR 2", "WCAG 1.4.3", "NSQ A", "UDL Engagement", "Product"}
	want := []string{"WCAG 1.4.3", "NSQ A", "UDL Engagement", "Product"}

	gotHS := sourcesForAudience(in, CourseSnapshot{HomeschoolMode: true})
	if !reflect.DeepEqual(gotHS, want) {
		t.Fatalf("homeschool: got %#v want %#v", gotHS, want)
	}
	gotK12 := sourcesForAudience(in, CourseSnapshot{OrgIsK12: true})
	if !reflect.DeepEqual(gotK12, want) {
		t.Fatalf("k12: got %#v want %#v", gotK12, want)
	}
}

func TestSourcesForAudience_HidesQmOscqrForParentCreator(t *testing.T) {
	in := []string{"QM 1.1", "Product"}
	got := sourcesForAudience(in, CourseSnapshot{CreatorIsParent: true})
	if len(got) != 1 || got[0] != "Product" {
		t.Fatalf("parent creator: got %#v", got)
	}
}

func TestSourcesForAudience_KeepsCampusForHigherEd(t *testing.T) {
	in := []string{"QM 1.1", "OSCQR 2", "WCAG 1.4.3"}
	got := sourcesForAudience(in, CourseSnapshot{})
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("higher-ed: got %#v want %#v", got, in)
	}
}

func TestIsCampusRubricSource(t *testing.T) {
	cases := map[string]bool{
		"QM 1.1":         true,
		"QM 8.x":         true,
		"OSCQR 2":        true,
		"OSCQR 44":       true,
		"WCAG 1.4.3":     false,
		"NSQ A":          false,
		"Product":        false,
		"UDL Engagement": false,
	}
	for src, want := range cases {
		if got := isCampusRubricSource(src); got != want {
			t.Errorf("%q: got %v want %v", src, got, want)
		}
	}
}
