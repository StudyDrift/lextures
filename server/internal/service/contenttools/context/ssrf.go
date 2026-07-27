package context

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/lextures/lextures/server/internal/webhooks"
)

// LookupIPs is overridable in tests (DNS rebinding simulations).
var LookupIPs = net.LookupIP

// ValidateFetchURL enforces SSRF policy: http(s) only, no credentials, no private
// / link-local / metadata ranges, including decimal/octal/hex IP forms (FR-4, AC-2).
func ValidateFetchURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: empty url", ErrSSRFBlocked)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid url", ErrSSRFBlocked)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme not allowed", ErrSSRFBlocked)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials not allowed", ErrSSRFBlocked)
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrSSRFBlocked)
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("%w: localhost", ErrSSRFBlocked)
	}
	if ip := parseLiteralIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: private or link-local address", ErrSSRFBlocked)
		}
		return nil
	}
	ips, err := LookupIPs(host)
	if err != nil {
		return fmt.Errorf("contenttools/context: resolve host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("contenttools/context: host did not resolve")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: resolves to private network", ErrSSRFBlocked)
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Unwrap IPv4-mapped IPv6 (::ffff:x.x.x.x).
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if webhooks.BlockedIP(ip) {
		return true
	}
	// Cloud metadata / documentation ranges beyond stdlib helpers.
	if v4 := ip.To4(); v4 != nil {
		// 100.64.0.0/10 CGNAT
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		// 192.0.0.0/24 IETF protocol assignments (includes some metadata)
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 0 {
			return true
		}
	}
	return false
}

// PinDial resolves host once and dials a single public IP (mitigates DNS rebinding).
func PinDial(network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if ip := parseLiteralIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, ErrSSRFBlocked
		}
		return net.Dial(network, net.JoinHostPort(ip.String(), port))
	}
	ips, err := LookupIPs(host)
	if err != nil {
		return nil, err
	}
	var public net.IP
	for _, ip := range ips {
		if !isBlockedIP(ip) {
			public = ip
			break
		}
	}
	if public == nil {
		return nil, ErrSSRFBlocked
	}
	return net.Dial(network, net.JoinHostPort(public.String(), port))
}
