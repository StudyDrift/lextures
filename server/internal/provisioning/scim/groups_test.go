package scim

import "testing"

func TestNormalizeGroupDisplayName(t *testing.T) {
	if got := normalizeGroupDisplayName("  Teachers  "); got != "Teachers" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeGroupDisplayName(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestMappingKey(t *testing.T) {
	a := groupMapping{Kind: "app_role", AppRoleName: "Teacher"}
	b := groupMapping{Kind: "app_role", AppRoleName: "teacher"}
	if mappingKey(a) != mappingKey(b) {
		t.Fatal("expected case-insensitive app role key")
	}
}

func TestDiffMappings(t *testing.T) {
	before := []groupMapping{
		{Kind: "app_role", AppRoleName: "Teacher"},
		{Kind: "org_role", OrgRoleKey: "org_viewer"},
	}
	after := []groupMapping{
		{Kind: "app_role", AppRoleName: "Teacher"},
		{Kind: "app_role", AppRoleName: "Student"},
	}
	added, removed := diffMappings(before, after)
	if len(added) != 1 || added[0].AppRoleName != "Student" {
		t.Fatalf("added: %+v", added)
	}
	if len(removed) != 1 || removed[0].OrgRoleKey != "org_viewer" {
		t.Fatalf("removed: %+v", removed)
	}
}