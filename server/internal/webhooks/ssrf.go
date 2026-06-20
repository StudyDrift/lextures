package webhooks

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrSSRFPolicy is returned when a webhook URL targets a blocked address range.
var ErrSSRFPolicy = errors.New("webhook URL blocked by SSRF policy: private, loopback, or link-local addresses are not allowed")

// ValidateEndpointURL checks that url is HTTPS and resolves only to public IPs.
func ValidateEndpointURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("endpoint URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("endpoint URL must use https")
	}
	if u.User != nil {
		return errors.New("endpoint URL must not include userinfo")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return errors.New("endpoint URL must include a hostname")
	}
	if strings.EqualFold(host, "localhost") {
		return ErrSSRFPolicy
	}
	if ip := net.ParseIP(host); ip != nil {
		if BlockedIP(ip) {
			return ErrSSRFPolicy
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("could not resolve endpoint hostname: %w", err)
	}
	if len(ips) == 0 {
		return errors.New("endpoint hostname did not resolve")
	}
	for _, ip := range ips {
		if BlockedIP(ip) {
			return ErrSSRFPolicy
		}
	}
	return nil
}

// BlockedIP reports whether ip is in a disallowed range for outbound webhooks.
func BlockedIP(ip net.IP) bool {
	return blockedIP(ip)
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	// Unique local (fc00::/7) and link-local IPv6.
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}
