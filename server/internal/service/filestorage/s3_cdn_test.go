package filestorage

import (
	"strings"
	"testing"
)

func TestRewriteCDNHost(t *testing.T) {
	d := &S3Driver{cdnBaseURL: "https://cdn.example.com"}
	raw := "https://bucket.s3.amazonaws.com/path/file.pdf?X-Amz-Signature=abc"
	got := d.rewriteCDNHost(raw)
	if !strings.HasPrefix(got, "https://cdn.example.com/") {
		t.Fatalf("rewriteCDNHost=%q", got)
	}
	if !strings.Contains(got, "X-Amz-Signature=abc") {
		t.Fatalf("expected query preserved, got %q", got)
	}
}

func TestRewriteCDNHost_Empty(t *testing.T) {
	d := &S3Driver{}
	raw := "https://bucket.s3.amazonaws.com/file.pdf"
	if got := d.rewriteCDNHost(raw); got != raw {
		t.Fatalf("expected unchanged URL, got %q", got)
	}
}
