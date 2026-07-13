package httpserver

import (
	"reflect"
	"testing"
)

func TestPlatformFeatureAuditChanges_OnlyMaskedBooleans(t *testing.T) {
	raw := []byte(`{
		"ffFeedback":false,
		"ffPublicCatalog":true,
		"smtpPassword":"do-not-record",
		"updateMask":["ffFeedback","ffPublicCatalog","smtpPassword"]
	}`)
	want := map[string]bool{"ffFeedback": false, "ffPublicCatalog": true}
	if got := platformFeatureAuditChanges(raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("platformFeatureAuditChanges() = %#v, want %#v", got, want)
	}
}

func TestPlatformFeatureAuditChanges_RequiresUpdateMask(t *testing.T) {
	if got := platformFeatureAuditChanges([]byte(`{"ffFeedback":true}`)); len(got) != 0 {
		t.Fatalf("expected no audited changes without updateMask, got %#v", got)
	}
}
