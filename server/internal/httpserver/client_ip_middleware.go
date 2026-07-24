package httpserver

import (
	"net"
	"net/http"

	"github.com/lextures/lextures/server/internal/ratelimit"
)

// clientIPMiddleware rewrites r.RemoteAddr to the caller IP when the TCP peer
// is a configured trusted proxy (RATE_LIMIT_TRUSTED_PROXIES). Unlike chi's
// deprecated middleware.RealIP, forged X-Forwarded-For / X-Real-IP from
// untrusted peers are ignored.
func (d Deps) clientIPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trusted := ratelimit.ParseCIDRs(d.effectiveConfig().RateLimits.TrustedProxies)
			ip := ratelimit.ClientIP(r.RemoteAddr, r.Header, trusted)
			if ip != "" {
				r.RemoteAddr = joinHostKeepPort(ip, r.RemoteAddr)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func joinHostKeepPort(host, remoteAddr string) string {
	if _, port, err := net.SplitHostPort(remoteAddr); err == nil && port != "" {
		return net.JoinHostPort(host, port)
	}
	return host
}
