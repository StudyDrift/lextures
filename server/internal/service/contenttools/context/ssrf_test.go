package context

import (
	"errors"
	"net"
	"testing"
)

func TestValidateFetchURL_blocksPrivateAndLiteralForms(t *testing.T) {
	cases := []string{
		"http://169.254.169.254/latest/meta-data",
		"http://localhost/admin",
		"http://127.0.0.1/",
		"http://[::1]/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://172.16.0.1/",
		"http://0x7f.0.0.1/",
		"http://0177.0.0.1/",
		"http://2130706433/",
		"http://user:pass@example.com/",
		"ftp://example.com/",
	}
	for _, u := range cases {
		err := ValidateFetchURL(u)
		if err == nil {
			t.Fatalf("expected block for %s", u)
		}
		if !errors.Is(err, ErrSSRFBlocked) {
			t.Fatalf("%s: want ErrSSRFBlocked, got %v", u, err)
		}
	}
}

func TestValidateFetchURL_allowsPublicHostShape(t *testing.T) {
	orig := LookupIPs
	t.Cleanup(func() { LookupIPs = orig })
	LookupIPs = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	if err := ValidateFetchURL("https://example.com/page"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateFetchURL_blocksDNSRebindingToPrivate(t *testing.T) {
	orig := LookupIPs
	t.Cleanup(func() { LookupIPs = orig })
	LookupIPs = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3")}, nil
	}
	err := ValidateFetchURL("https://evil.example/rebind")
	if !errors.Is(err, ErrSSRFBlocked) {
		t.Fatalf("want ErrSSRFBlocked, got %v", err)
	}
}

func TestValidateFetchURL_blocksIPv6Mapped(t *testing.T) {
	err := ValidateFetchURL("http://[::ffff:127.0.0.1]/")
	if !errors.Is(err, ErrSSRFBlocked) {
		t.Fatalf("want ErrSSRFBlocked, got %v", err)
	}
}

func TestHostBreakerOpensAfterThreeFailures(t *testing.T) {
	host := "flaky-" + t.Name() + ".example"
	if HostBreakerOpen(host) {
		t.Fatal("should start closed")
	}
	HostBreakerRecordFailure(host)
	HostBreakerRecordFailure(host)
	if HostBreakerOpen(host) {
		t.Fatal("should still be closed after 2 failures")
	}
	HostBreakerRecordFailure(host)
	if !HostBreakerOpen(host) {
		t.Fatal("should be open after 3 failures")
	}
	HostBreakerRecordSuccess(host)
	if HostBreakerOpen(host) {
		t.Fatal("success should clear breaker")
	}
}
