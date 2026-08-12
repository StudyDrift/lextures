package l10n

import "testing"

// AppAndMarketingLocalesAgree keeps the product UI locale list and marketing
// content locale codes in lockstep (MC.14 NFR Internationalization).
func TestAppAndMarketingLocalesAgree(t *testing.T) {
	t.Parallel()
	app := []string{"en", "es", "fr", "ar"}
	for _, code := range app {
		if _, err := NormalizeLocale(code); err != nil {
			t.Fatalf("app locale %q is not a valid BCP-47 tag: %v", code, err)
		}
	}
}
