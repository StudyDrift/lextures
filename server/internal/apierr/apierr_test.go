package apierr

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteJSON(rr, 400, CodeInvalidInput, "bad")
	if rr.Code != 400 {
		t.Fatalf("status: %d", rr.Code)
	}
	var b Body
	if err := json.NewDecoder(rr.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Error.Code != CodeInvalidInput || b.Error.Message != "bad" {
		t.Fatalf("body: %#v", b)
	}
}

func TestWritePaymentRequired(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WritePaymentRequired(rr, "Purchase required.", "/marketplace/paid")
	if rr.Code != 402 {
		t.Fatalf("status: %d", rr.Code)
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		CheckoutHint string `json:"checkoutHint"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != CodePaymentRequired {
		t.Fatalf("code: %q", payload.Error.Code)
	}
	if payload.CheckoutHint != "/marketplace/paid" {
		t.Fatalf("hint: %q", payload.CheckoutHint)
	}
}
