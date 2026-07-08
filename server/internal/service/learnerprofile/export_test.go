package learnerprofile

import (
	"encoding/json"
	"testing"
)

func TestExportDocument_IncludesDisclosure(t *testing.T) {
	doc := ExportDocument{
		UserID: "00000000-0000-0000-0000-000000000001",
		Status: "active",
		Disclosure: ExportDisclosure{
			ProfilingNotice: "notice",
			Art22Posture:    "advisory only",
		},
		ExportKind: "learner-profile",
		ExportedAt: "2026-01-01T00:00:00Z",
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	disclosure, ok := parsed["disclosure"].(map[string]any)
	if !ok {
		t.Fatalf("missing disclosure: %+v", parsed)
	}
	if disclosure["art22Posture"] != "advisory only" {
		t.Fatalf("art22Posture: %+v", disclosure)
	}
	if parsed["exportKind"] != "learner-profile" {
		t.Fatalf("exportKind: %+v", parsed["exportKind"])
	}
}