package marketingcontent

import (
	"encoding/json"
	"testing"
)

func TestSocialFromExtra(t *testing.T) {
	t.Parallel()
	title, desc := SocialFromExtra(json.RawMessage(`{"socialTitle":"Share me","socialDescription":"One line.","other":1}`))
	if title != "Share me" || desc != "One line." {
		t.Fatalf("got title=%q desc=%q", title, desc)
	}
	title, desc = SocialFromExtra(nil)
	if title != "" || desc != "" {
		t.Fatalf("empty extra: %q %q", title, desc)
	}
}

func TestMergeSocialIntoExtraPreservesOtherKeys(t *testing.T) {
	t.Parallel()
	got := MergeSocialIntoExtra(json.RawMessage(`{"keep":true}`), "Title", "Desc")
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["keep"] != true || payload["socialTitle"] != "Title" || payload["socialDescription"] != "Desc" {
		t.Fatalf("got %#v", payload)
	}
	got = MergeSocialIntoExtra(got, "", "")
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["socialTitle"]; ok {
		t.Fatalf("expected social keys removed: %#v", payload)
	}
	if payload["keep"] != true {
		t.Fatalf("lost keep: %#v", payload)
	}
}
