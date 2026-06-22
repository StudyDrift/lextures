package browsersaml

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"net/http/httptest"
	"testing"
)

func TestDecodeSAMLRedirectParam(t *testing.T) {
	xml := []byte(`<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="r1" Version="2.0" IssueInstant="2026-01-01T00:00:00Z"><saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example</saml:Issuer><saml:NameID xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">user@example.com</saml:NameID></samlp:LogoutRequest>`)
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(xml); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	enc := base64.StdEncoding.EncodeToString(buf.Bytes())
	got, err := decodeSAMLRedirectParam(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("LogoutRequest")) {
		t.Fatalf("got %s", got)
	}
}

func TestHasSAMLRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/auth/saml/slo?SAMLRequest=abc", nil)
	if !hasSAMLRequest(r) {
		t.Fatal("expected SAMLRequest in query")
	}
	r = httptest.NewRequest("POST", "/auth/saml/slo", nil)
	if hasSAMLRequest(r) {
		t.Fatal("expected no SAMLRequest")
	}
}