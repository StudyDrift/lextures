package httpserver

import (
	"net/http/httptest"
	"testing"
)

func TestWritePublicMarketingMediaWritesExactBodyLength(t *testing.T) {
	t.Parallel()
	body := []byte("image bytes")
	response := httptest.NewRecorder()

	writePublicMarketingMedia(response, "image/png", body)

	result := response.Result()
	defer func() { _ = result.Body.Close() }()
	if got := response.Body.Bytes(); string(got) != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}
	if got := result.Header.Get("Content-Length"); got != "11" {
		t.Fatalf("Content-Length = %q, want 11", got)
	}
	if got := result.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
}
