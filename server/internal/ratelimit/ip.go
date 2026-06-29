package ratelimit

import (
	"net"
	"net/http"
	"strings"
)

// ParseCIDRs parses a list of CIDR strings or bare IPs into networks, skipping
// malformed entries. A bare IP (e.g. "203.0.113.5") becomes a /32 or /128.
func ParseCIDRs(entries []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, raw := range entries {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			if ip := net.ParseIP(s); ip != nil {
				if ip.To4() != nil {
					s += "/32"
				} else {
					s += "/128"
				}
			}
		}
		if _, n, err := net.ParseCIDR(s); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// ipInNets reports whether ip (a bare address string) falls within any network.
func ipInNets(ip string, nets []*net.IPNet) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// hostOnly strips the :port from an address, returning the bare host/IP.
func hostOnly(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// ClientIP extracts the caller's IP, honouring X-Real-IP / X-Forwarded-For only
// when the immediate peer (remoteAddr) is a trusted proxy (plan 17.6 NFR
// security). Requests from untrusted peers always use the raw connection
// address, so a forged X-Forwarded-For cannot spoof the rate-limit identity.
func ClientIP(remoteAddr string, hdr http.Header, trusted []*net.IPNet) string {
	peer := hostOnly(remoteAddr)
	if peer == "" || !ipInNets(peer, trusted) {
		return peer
	}
	if xri := strings.TrimSpace(hdr.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := hdr.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Walk right-to-left and return the first address that is not itself a
		// trusted proxy — that is the real client behind the proxy chain.
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip != "" && !ipInNets(ip, trusted) {
				return ip
			}
		}
		if first := strings.TrimSpace(parts[0]); first != "" {
			return first
		}
	}
	return peer
}
