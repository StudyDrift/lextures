package kernel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/apierr"
)

func TestValidation_WritesUX6Envelope(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteError(rr, httptest.NewRequest(http.MethodPost, "/", nil), Validation(
		"Fix the highlighted fields",
		FieldError("title", "required", "Title is required."),
		FieldError("sections[0].code", "already_taken", "This code is already in use."),
	))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", rr.Code)
	}
	var body apierr.ValidationFailedBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "validation_failed" || body.Message != "Fix the highlighted fields" {
		t.Fatalf("body=%+v", body)
	}
	if len(body.Fields) != 2 || body.Fields[0].Path != "title" || body.Fields[1].Path != "sections[0].code" {
		t.Fatalf("fields=%+v", body.Fields)
	}
}

func TestCollectValidation_Empty(t *testing.T) {
	t.Parallel()
	if err := CollectValidation("x", nil); err != nil {
		t.Fatal(err)
	}
}

func TestFieldError_Defaults(t *testing.T) {
	t.Parallel()
	f := FieldError("name", "", "")
	if f.Code != "custom" || f.Message != "Enter a valid value for this field." {
		t.Fatalf("%+v", f)
	}
}
