package paymentprovider

import (
	"testing"

	"github.com/google/uuid"
)

func TestResolveProvider_DefaultStripe(t *testing.T) {
	name, err := ResolveProvider("", Config{StripeSecretKey: "sk_test_x"})
	if err != nil {
		t.Fatal(err)
	}
	if name != ProviderStripe {
		t.Fatalf("provider: %s", name)
	}
}

func TestResolveProvider_PayPal(t *testing.T) {
	name, err := ResolveProvider(ProviderPayPal, Config{
		PayPalClientID:     "id",
		PayPalClientSecret: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != ProviderPayPal {
		t.Fatalf("provider: %s", name)
	}
}

func TestFactory_BuildStripe(t *testing.T) {
	p, err := Factory{}.Build(ProviderStripe, Config{StripeSecretKey: "sk_test_x"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != ProviderStripe {
		t.Fatalf("name: %s", p.Name())
	}
}

func TestParsePayPalCustomID(t *testing.T) {
	uid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	custom := paypalCustomID(uid, &cid, "key-1")
	gotUID, gotCID, key := parsePayPalCustomID(custom)
	if gotUID != uid || gotCID == nil || *gotCID != cid || key != "key-1" {
		t.Fatalf("parse: uid=%v course=%v key=%q", gotUID, gotCID, key)
	}
}

func TestStripeCheckoutPaymentMethods_NL(t *testing.T) {
	methods := stripeCheckoutPaymentMethods("NL")
	if len(methods) != 2 || methods[1] != "ideal" {
		t.Fatalf("methods: %v", methods)
	}
}

func TestMockProvider_Checkout(t *testing.T) {
	mock := &MockProvider{
		CheckoutResult: &CheckoutResult{SessionID: "sess_1", CheckoutURL: "https://example.com"},
	}
	got, err := mock.CreateCheckoutSession(t.Context(), CheckoutRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "sess_1" {
		t.Fatalf("session: %s", got.SessionID)
	}
}
