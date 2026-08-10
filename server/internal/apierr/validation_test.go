package apierr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/apierr"
)

func TestWriteValidationFailed_Envelope(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	apierr.WriteValidationFailed(rr, "Fix the highlighted fields", []apierr.FieldViolation{
		{Path: "firstName", Code: "required", Message: "Enter a first name."},
		{Path: "phoneNumber", Code: "invalid_phone", Message: "Enter a valid phone number.", Params: map[string]any{"example": "+1 555 0100"}},
	})

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type=%q", ct)
	}

	var body apierr.ValidationFailedBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error != apierr.ErrorValidationFailed {
		t.Fatalf("error=%q", body.Error)
	}
	if body.Message != "Fix the highlighted fields" {
		t.Fatalf("message=%q", body.Message)
	}
	if len(body.Fields) != 2 {
		t.Fatalf("fields len=%d", len(body.Fields))
	}
	if body.Fields[0].Path != "firstName" || body.Fields[0].Code != "required" {
		t.Fatalf("field0=%+v", body.Fields[0])
	}
	if body.Fields[1].Params["example"] != "+1 555 0100" {
		t.Fatalf("params=%v", body.Fields[1].Params)
	}
}

func TestWriteValidationFailed_EmptyFields(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	apierr.WriteValidationFailed(rr, "", nil)
	if rr.Code != 422 {
		t.Fatalf("status=%d", rr.Code)
	}
	var body apierr.ValidationFailedBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "Validation failed" {
		t.Fatalf("default message=%q", body.Message)
	}
	if body.Fields == nil {
		t.Fatal("fields should be empty slice not null")
	}
}
