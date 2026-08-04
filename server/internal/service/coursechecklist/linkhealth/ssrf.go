package linkhealth

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// BlockedReason identifies why a URL was not fetched (metrics label).
type BlockedReason string

const (
	BlockedPrivateRange BlockedReason = "private_range"
	BlockedRobots       BlockedReason = "robots"
	BlockedRateLimit    BlockedReason = "rate_limit"
	BlockedInvalidURL   BlockedReason = "invalid_url"
	BlockedScheme       BlockedReason = "scheme"
)

// IsBlockedHost reports whether host resolves to a private/link-local/loopback
// address that must never be contacted (SSRF defence).
func IsBlockedHost(host string) (BlockedReason, bool) {
	host = strings.TrimSpace(host)
	if host == "" {
		return BlockedInvalidURL, true
	}
	// Strip port.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return BlockedPrivateRange, true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ipBlocked(ip) {
			return BlockedPrivateRange, true
		}
		return "", false
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS failure is not a block — caller treats as error/unknown.
		return "", false
	}
	for _, resolved := range ips {
		if ipBlocked(resolved) {
			return BlockedPrivateRange, true
		}
	}
	return "", false
}

func ipBlocked(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// Extra: IPv4 link-local / CGNAT / metadata-ish ranges already covered by IsPrivate/IsLinkLocal.
	return false
}

// ValidateURL parses and rejects non-http(s) schemes and blocked hosts before dial.
func ValidateURL(raw string) (*url.URL, BlockedReason, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, BlockedInvalidURL, fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, BlockedInvalidURL, err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return nil, BlockedScheme, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, BlockedInvalidURL, fmt.Errorf("missing host")
	}
	if reason, blocked := IsBlockedHost(u.Hostname()); blocked {
		return nil, reason, fmt.Errorf("blocked host")
	}
	return u, "", nil
}
