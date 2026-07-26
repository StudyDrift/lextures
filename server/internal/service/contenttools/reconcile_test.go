package contenttools

import (
	"strings"
	"testing"
)

func TestStripCrossCourseFenceLogic(t *testing.T) {
	// Unit-level stand-in for reconcile: only course-owned ids are valid.
	inCourse := "11111111-1111-1111-1111-111111111111"
	foreign := "22222222-2222-2222-2222-222222222222"
	md := "keep\n" +
		SerializeLexToolFence(LexToolFencePayload{InstanceID: inCourse, ToolID: "noop_probe", V: 1}) +
		"\n" +
		SerializeLexToolFence(LexToolFencePayload{InstanceID: foreign, ToolID: "noop_probe", V: 1}) +
		"\nend"
	cleaned := StripInvalidLexToolFences(md, map[string]bool{inCourse: true})
	if !strings.Contains(cleaned, inCourse) {
		t.Fatal("in-course fence should remain")
	}
	if strings.Contains(cleaned, foreign) {
		t.Fatal("cross-course fence should be stripped")
	}
	refs := ParseLexToolFences(cleaned)
	if len(refs) != 1 || refs[0].InstanceID != inCourse {
		t.Fatalf("unexpected refs: %+v", refs)
	}
}
