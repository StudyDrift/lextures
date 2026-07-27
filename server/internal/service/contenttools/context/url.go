package context

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NormalizeURL canonicalizes a URL for cache keys (scheme/host lowercased, fragment stripped).
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrSSRFBlocked
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrSSRFBlocked
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	// Drop default ports.
	if (u.Scheme == "http" && strings.HasSuffix(u.Host, ":80")) ||
		(u.Scheme == "https" && strings.HasSuffix(u.Host, ":443")) {
		host, _, splitErr := net.SplitHostPort(u.Host)
		if splitErr == nil {
			u.Host = host
		}
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

// HashURL returns sha256 hex of the normalized URL.
func HashURL(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// HostOf returns the hostname (no port) for a URL string.
func HostOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// parseLiteralIP expands decimal/octal/hex IPv4 forms that net.ParseIP rejects
// (e.g. 2130706433, 0177.0.0.1, 0x7f.0.0.1) for SSRF defence (AC-2).
func parseLiteralIP(host string) net.IP {
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}
	// Dotted forms with octal/hex components.
	if strings.Contains(host, ".") {
		parts := strings.Split(host, ".")
		if len(parts) != 4 {
			return nil
		}
		var b [4]byte
		for i, p := range parts {
			n, err := parseIPv4Component(p)
			if err != nil || n > 255 {
				return nil
			}
			b[i] = byte(n)
		}
		return net.IPv4(b[0], b[1], b[2], b[3])
	}
	// Single integer (decimal / hex / octal) → IPv4.
	n, err := parseIPv4Component(host)
	if err != nil || n > 0xffffffff {
		return nil
	}
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func parseIPv4Component(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	base := 10
	switch {
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		base = 16
		s = s[2:]
	case len(s) > 1 && s[0] == '0':
		base = 8
		s = s[1:]
	}
	return strconv.ParseUint(s, base, 64)
}
