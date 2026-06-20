package webhooks

import "testing"

func TestValidateEndpointURL_blocksPrivateIPs(t *testing.T) {
	cases := []string{
		"http://example.com/hook",
		"https://127.0.0.1/hook",
		"https://10.0.0.1/hook",
		"https://192.168.1.5/hook",
		"https://169.254.169.254/latest/meta-data",
		"https://localhost/hook",
	}
	for _, u := range cases {
		err := ValidateEndpointURL(u)
		if err == nil {
			t.Fatalf("expected rejection for %q", u)
		}
	}
}

func TestValidateEndpointURL_acceptsPublicHTTPS(t *testing.T) {
	err := ValidateEndpointURL("https://1.1.1.1/hook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeEventTypes(t *testing.T) {
	out, ok := NormalizeEventTypes([]string{"grade.posted", "grade.posted", "enrollment.created"})
	if !ok || len(out) != 2 {
		t.Fatalf("got %v ok=%v", out, ok)
	}
	_, ok = NormalizeEventTypes([]string{"not.real"})
	if ok {
		t.Fatal("expected invalid event type")
	}
}
