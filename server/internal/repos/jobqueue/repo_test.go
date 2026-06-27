package jobqueue

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalPayload(t *testing.T) {
	t.Run("nil becomes empty object", func(t *testing.T) {
		b, err := marshalPayload(nil)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "{}" {
			t.Fatalf("got %q", b)
		}
	})
	t.Run("empty raw becomes empty object", func(t *testing.T) {
		b, err := marshalPayload(json.RawMessage(nil))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "{}" {
			t.Fatalf("got %q", b)
		}
	})
	t.Run("struct marshals", func(t *testing.T) {
		b, err := marshalPayload(struct {
			A int `json:"a"`
		}{A: 7})
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != `{"a":7}` {
			t.Fatalf("got %q", b)
		}
	})
	t.Run("raw passthrough", func(t *testing.T) {
		b, err := marshalPayload(json.RawMessage(`{"x":1}`))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != `{"x":1}` {
			t.Fatalf("got %q", b)
		}
	})
}

func TestTruncateErr(t *testing.T) {
	if got := truncateErr("short"); got != "short" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("x", 5000)
	if got := truncateErr(long); len(got) != 4000 {
		t.Fatalf("want 4000 got %d", len(got))
	}
}
