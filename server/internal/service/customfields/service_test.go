package customfields

import (
	"testing"

	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"student_id", true},
		{"grade_level", true},
		{"StudentId", false},
		{"1bad", false},
		{"email", false},
		{"", false},
	}
	for _, tc := range tests {
		err := ValidateKey(tc.key)
		if tc.valid && err != nil {
			t.Errorf("ValidateKey(%q) = %v, want nil", tc.key, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("ValidateKey(%q) = nil, want error", tc.key)
		}
	}
}

func TestValidateValuesSelect(t *testing.T) {
	defs := []cfrepo.Definition{{
		Key: "department", FieldType: cfrepo.FieldSelect, SelectOptions: []string{"Math", "Science"},
	}}
	errs := ValidateValues(defs, map[string]any{"department": "History"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	errs = ValidateValues(defs, map[string]any{"department": "Math"})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestFilterByVisibility(t *testing.T) {
	defs := []cfrepo.Definition{
		{Key: "title_one", Visibility: cfrepo.VisibilityAdminOnly},
		{Key: "section", Visibility: cfrepo.VisibilityInstructor},
		{Key: "nickname", Visibility: cfrepo.VisibilityStudent},
	}
	values := map[string]any{"title_one": true, "section": "A", "nickname": "Sam"}

	student := FilterByVisibility(defs, values, ViewerStudent)
	if _, ok := student["title_one"]; ok {
		t.Error("student should not see admin_only field")
	}
	if _, ok := student["section"]; ok {
		t.Error("student should not see instructor field")
	}
	if student["nickname"] != "Sam" {
		t.Error("student should see student field")
	}

	instructor := FilterByVisibility(defs, values, ViewerInstructor)
	if _, ok := instructor["title_one"]; ok {
		t.Error("instructor should not see admin_only field")
	}
	if instructor["section"] != "A" {
		t.Error("instructor should see instructor field")
	}

	admin := FilterByVisibility(defs, values, ViewerAdmin)
	if admin["title_one"] != true {
		t.Error("admin should see admin_only field")
	}
}

func TestMergePatch(t *testing.T) {
	existing := map[string]any{"a": "1", "b": "2"}
	patch := map[string]any{"b": "updated", "c": "3", "d": nil}
	merged := MergePatch(existing, patch)
	if merged["a"] != "1" || merged["b"] != "updated" || merged["c"] != "3" {
		t.Fatalf("unexpected merge: %v", merged)
	}
	if _, ok := merged["d"]; ok {
		t.Error("nil patch value should remove key")
	}
}

func TestCoerceCSVValue(t *testing.T) {
	boolDef := cfrepo.Definition{FieldType: cfrepo.FieldBoolean}
	v, err := CoerceCSVValue(boolDef, "yes")
	if err != nil || v != true {
		t.Fatalf("bool yes: %v %v", v, err)
	}
	numDef := cfrepo.Definition{FieldType: cfrepo.FieldNumber}
	v, err = CoerceCSVValue(numDef, "42")
	if err != nil || v != int64(42) {
		t.Fatalf("number 42: %v %v", v, err)
	}
}
