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
		{"Email", false},
		{"1bad", false},
		{"email", false},
		{"", false},
	}
	for _, tc := range tests {
		err := ValidateKey(tc.key)
		if tc.valid && err != nil {
			t.Fatalf("ValidateKey(%q) unexpected error: %v", tc.key, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("ValidateKey(%q) expected error", tc.key)
		}
	}
}

func TestFilterValuesVisibility(t *testing.T) {
	defs := []cfrepo.Definition{
		{Key: "student_id", Visibility: cfrepo.VisibilityStudent},
		{Key: "title_one", Visibility: cfrepo.VisibilityAdminOnly},
		{Key: "department", Visibility: cfrepo.VisibilityInstructor},
	}
	values := map[string]any{
		"student_id": "123",
		"title_one":  true,
		"department": "Math",
		"unknown":    "x",
	}
	student := FilterValues(values, defs, AudienceStudent, false)
	if _, ok := student["student_id"]; !ok {
		t.Fatal("student should see student_id")
	}
	if _, ok := student["title_one"]; ok {
		t.Fatal("student should not see admin_only field")
	}
	if _, ok := student["department"]; ok {
		t.Fatal("student should not see instructor field")
	}

	instructor := FilterValues(values, defs, AudienceInstructor, false)
	if _, ok := instructor["department"]; !ok {
		t.Fatal("instructor should see department")
	}
	if _, ok := instructor["title_one"]; ok {
		t.Fatal("instructor should not see admin_only field")
	}

	admin := FilterValues(values, defs, AudienceAdmin, false)
	if len(admin) != 3 {
		t.Fatalf("admin should see 3 fields, got %d", len(admin))
	}
}

func TestNormalizeValueSelect(t *testing.T) {
	def := cfrepo.Definition{
		Key:           "department",
		FieldType:     cfrepo.FieldSelect,
		SelectOptions: []string{"Math", "Science"},
	}
	if _, err := normalizeValue(def, "History"); err == nil {
		t.Fatal("expected validation error for invalid select option")
	}
	val, err := normalizeValue(def, "Math")
	if err != nil || val != "Math" {
		t.Fatalf("expected Math, got %v err=%v", val, err)
	}
}

func TestNormalizeValueBoolean(t *testing.T) {
	def := cfrepo.Definition{Key: "flag", FieldType: cfrepo.FieldBoolean}
	val, err := normalizeValue(def, "yes")
	if err != nil || val != true {
		t.Fatalf("expected true, got %v err=%v", val, err)
	}
}
