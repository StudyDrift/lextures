package kernel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEncodeJSON_Envelope(t *testing.T) {
	t.Parallel()
	type out struct {
		ID string `json:"id"`
	}
	rr := httptest.NewRecorder()
	EncodeJSON(rr, http.StatusCreated, out{ID: "1"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != jsonContentType {
		t.Fatalf("ct=%q", ct)
	}
	var got out
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestEncodeJSON_MatchesHandRolled(t *testing.T) {
	t.Parallel()
	type out struct {
		Title string `json:"title"`
	}
	v := out{Title: "A"}

	hand := httptest.NewRecorder()
	hand.Header().Set("Content-Type", jsonContentType)
	hand.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(hand).Encode(v)

	kit := httptest.NewRecorder()
	EncodeJSON(kit, http.StatusOK, v)

	if hand.Body.String() != kit.Body.String() {
		t.Fatalf("body %q vs %q", hand.Body.String(), kit.Body.String())
	}
	if hand.Header().Get("Content-Type") != kit.Header().Get("Content-Type") {
		t.Fatalf("ct mismatch")
	}
}

func TestEncodeJSON_NoContent(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	EncodeJSON(rr, http.StatusNoContent, struct{}{})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body=%q", rr.Body.String())
	}
}
