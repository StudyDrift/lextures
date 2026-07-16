package httpserver

import "testing"

func TestMaskSecret_BYOKNeverReturnsPlaintext(t *testing.T) {
	if got := maskSecret(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	plain := "sk-ant-super-secret-key"
	got := maskSecret(plain)
	if got == "" || got == plain {
		t.Fatalf("expected masked placeholder, got %q", got)
	}
	if got != placeholderSecretResponse {
		t.Fatalf("got %q want %q", got, placeholderSecretResponse)
	}
}
