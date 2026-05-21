package clamav_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/clamav"
)

func TestStubDetectsEICAR(t *testing.T) {
	c := clamav.NewClient("", true)
	payload := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	res, err := c.ScanStream(t.Context(), strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if res.Clean {
		t.Fatal("expected infected")
	}
}

func TestStubCleanFile(t *testing.T) {
	c := clamav.NewClient("", true)
	res, err := c.ScanStream(t.Context(), bytes.NewReader([]byte("hello world pdf content")))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Clean {
		t.Fatalf("expected clean, got %+v", res)
	}
}

func TestQuarantineKey(t *testing.T) {
	if got := clamav.QuarantineKey("files/C101/uuid.pdf"); got != "quarantine/files/C101/uuid.pdf" {
		t.Fatalf("got %q", got)
	}
}
