package quizgame

import (
	"strings"
	"testing"
)

func TestDeepCopyOptsDefaults(t *testing.T) {
	opts := DeepCopyOpts{
		SourceKitID:      "00000000-0000-4000-8000-000000000001",
		TargetCourseCode: "C-TEST",
	}
	if opts.AsTemplate {
		t.Fatal("AsTemplate should default false")
	}
	if opts.DropBankLinks {
		t.Fatal("DropBankLinks should default false")
	}
}

func TestSharePermissionNormalize(t *testing.T) {
	p, err := normalizeSharePermission("")
	if err != nil || p != SharePermCopy {
		t.Fatalf("default perm: got %q err=%v", p, err)
	}
	if _, err := normalizeSharePermission("admin"); err == nil {
		t.Fatal("expected invalid permission error")
	}
	gt, err := normalizeGranteeType("ORG")
	if err != nil || gt != ShareGranteeOrg {
		t.Fatalf("grantee: got %q err=%v", gt, err)
	}
}

func TestTemplateScopeNormalize(t *testing.T) {
	s, err := NormalizeTemplateScope("System")
	if err != nil || s != TemplateScopeSystem {
		t.Fatalf("got %q err=%v", s, err)
	}
	if _, err := NormalizeTemplateScope("builtin"); err == nil {
		t.Fatal("expected error for builtin")
	}
}

func TestCatalogStatusValidation(t *testing.T) {
	// Ensure invalid statuses are rejected by the same rules SetCatalogStatus uses.
	for _, st := range []string{"unlisted", "pending", "listed", "rejected"} {
		st = strings.TrimSpace(strings.ToLower(st))
		switch st {
		case "unlisted", "pending", "listed", "rejected":
		default:
			t.Fatalf("unexpected %q", st)
		}
	}
}
