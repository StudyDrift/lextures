package paymentprovider

import "fmt"

// Factory builds concrete payment providers.
type Factory struct{}

// Build returns a provider implementation for the given name.
func (Factory) Build(name ProviderName, cfg Config) (Provider, error) {
	switch name {
	case ProviderStripe:
		if !cfg.StripeConfigured() {
			return nil, fmt.Errorf("paymentprovider: stripe not configured")
		}
		return NewStripeProvider(cfg), nil
	case ProviderPayPal:
		if !cfg.PayPalConfigured() {
			return nil, fmt.Errorf("paymentprovider: paypal not configured")
		}
		return NewPayPalProvider(cfg), nil
	default:
		return nil, fmt.Errorf("paymentprovider: unknown provider %q", name)
	}
}

// ResolveProvider picks the checkout backend from an explicit choice or platform defaults.
func ResolveProvider(requested ProviderName, cfg Config) (ProviderName, error) {
	switch requested {
	case "":
		if cfg.StripeConfigured() {
			return ProviderStripe, nil
		}
		if cfg.PayPalConfigured() {
			return ProviderPayPal, nil
		}
		return "", fmt.Errorf("paymentprovider: no provider configured")
	case ProviderStripe, ProviderPayPal:
		var factory Factory
		if _, err := factory.Build(requested, cfg); err != nil {
			return "", err
		}
		return requested, nil
	default:
		return "", fmt.Errorf("paymentprovider: unsupported provider %q", requested)
	}
}
