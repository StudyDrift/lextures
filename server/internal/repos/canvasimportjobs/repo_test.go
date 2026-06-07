package canvasimportjobs

import (
	"encoding/json"
	"testing"
)

func TestIncludeJSONRoundTrip(t *testing.T) {
	inc := Include{Modules: true, Files: true}
	b, err := json.Marshal(inc)
	if err != nil {
		t.Fatal(err)
	}
	var out Include
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out != inc {
		t.Fatalf("got %+v want %+v", out, inc)
	}
}
