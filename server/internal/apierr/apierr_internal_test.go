package apierr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

)

func TestWriteInternal_RecordsServerError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req = WithServerErrorTracking(req)
	rr := httptest.NewRecorder()

	err := errors.New("query failed")
	WriteInternal(rr, req, "Failed to load submissions.", err)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rr.Code, http.StatusInternalServerError)
	}
	msg, got := ServerErrorFromRequest(req)
	if msg != "Failed to load submissions." {
		t.Fatalf("message: got %q want %q", msg, "Failed to load submissions.")
	}
	if got != err {
		t.Fatalf("err: got %v want %v", got, err)
	}

	var b Body
	if err := json.NewDecoder(rr.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Error.Code != CodeInternal || b.Error.Message != "Failed to load submissions." {
		t.Fatalf("body: %#v", b)
	}
}