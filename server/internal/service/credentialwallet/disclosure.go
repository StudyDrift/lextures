package credentialwallet

import (
	"time"

	walletrepo "github.com/lextures/lextures/server/internal/repos/wallet"
)

// PublicItem is a disclosure-filtered projection for shared collections.
type PublicItem struct {
	Kind       string  `json:"kind"`
	Valid      bool    `json:"valid"`
	Revoked    bool    `json:"revoked"`
	Title      *string `json:"title,omitempty"`
	Issuer     *string `json:"issuer,omitempty"`
	IssuedAt   *string `json:"issuedAt,omitempty"`
	VerifyURL  *string `json:"verifyUrl,omitempty"`
	VerifyStatus *string `json:"verifyStatus,omitempty"`
}

// FilterItem applies collection disclosure to a wallet item.
func FilterItem(it walletrepo.Item, disclosure walletrepo.Disclosure, webOrigin string) PublicItem {
	out := PublicItem{
		Kind:    string(it.Kind),
		Valid:   !it.Revoked,
		Revoked: it.Revoked,
	}
	switch disclosure {
	case walletrepo.DisclosureValidity:
		return out
	case walletrepo.DisclosureSummary:
		title := it.Title
		out.Title = &title
		out.Issuer = it.Issuer
		if it.IssuedAt != nil {
			s := it.IssuedAt.UTC().Format(time.RFC3339)
			out.IssuedAt = &s
		}
		status := VerifyStatus(it)
		out.VerifyStatus = &status
		return out
	case walletrepo.DisclosureFull:
		title := it.Title
		out.Title = &title
		out.Issuer = it.Issuer
		if it.IssuedAt != nil {
			s := it.IssuedAt.UTC().Format(time.RFC3339)
			out.IssuedAt = &s
		}
		status := VerifyStatus(it)
		out.VerifyStatus = &status
		if url := VerifyURL(webOrigin, it); url != "" {
			out.VerifyURL = &url
		}
		return out
	default:
		return out
	}
}

// NormalizeDisclosure returns a valid disclosure, defaulting to minimal (validity).
func NormalizeDisclosure(d string) walletrepo.Disclosure {
	switch walletrepo.Disclosure(d) {
	case walletrepo.DisclosureValidity, walletrepo.DisclosureSummary, walletrepo.DisclosureFull:
		return walletrepo.Disclosure(d)
	default:
		return walletrepo.DisclosureValidity
	}
}
