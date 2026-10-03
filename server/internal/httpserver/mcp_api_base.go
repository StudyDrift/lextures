package httpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// apiBaseURL is the origin an out-of-process MCP client should call.
// LTI_API_BASE_URL defaults to http://localhost:8080 when unset, which is the
// developer API, not the deployment a browser just reached. Prefer an explicit
// public LTI base, then the request's public host (nginx forwards Host and
// X-Forwarded-Proto), then a non-loopback PUBLIC_WEB_ORIGIN.
func (d Deps) apiBaseURL(r *http.Request) string {
	cfg := d.effectiveConfig()
	if base := canonicalPublicOrigin(cfg.LTIAPIBaseURL); base != "" {
		return base
	}
	if base := requestPublicOrigin(r); base != "" {
		return base
	}
	if base := canonicalPublicOrigin(cfg.PublicWebOrigin); base != "" {
		return base
	}
	if base := strings.TrimRight(strings.TrimSpace(cfg.LTIAPIBaseURL), "/"); base != "" {
		return base
	}
	return "http://localhost:8080"
}

func canonicalPublicOrigin(raw string) string {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	if hostIsLoopback(u.Host) {
		return ""
	}
	return s
}

func requestPublicOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := strings.TrimSpace(r.Host)
	if !plausibleRequestHost(host) || hostIsLoopback(host) {
		return ""
	}
	return requestScheme(r) + "://" + host
}

func requestScheme(r *http.Request) string {
	switch forwardedProto(r) {
	case "https":
		return "https"
	case "http":
		if cloudflareVisitorScheme(r) == "https" {
			return "https"
		}
		return "http"
	}
	if cloudflareVisitorScheme(r) == "https" || r.TLS != nil {
		return "https"
	}
	return "http"
}

func forwardedProto(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if raw == "" {
		return ""
	}
	proto := strings.ToLower(strings.TrimSpace(strings.Split(raw, ",")[0]))
	if proto == "https" || proto == "http" {
		return proto
	}
	return ""
}

func cloudflareVisitorScheme(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("CF-Visitor"))
	if raw == "" {
		return ""
	}
	var visitor struct {
		Scheme string `json:"scheme"`
	}
	if err := json.Unmarshal([]byte(raw), &visitor); err != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(visitor.Scheme)) {
	case "https", "http":
		return strings.ToLower(strings.TrimSpace(visitor.Scheme))
	default:
		return ""
	}
}

func plausibleRequestHost(hostport string) bool {
	if hostport == "" || strings.ContainsAny(hostport, " \t\r\n/\\@") {
		return false
	}
	u, err := url.Parse("http://" + hostport)
	if err != nil || u.Hostname() == "" {
		return false
	}
	return strings.EqualFold(u.Host, hostport)
}

func hostIsLoopback(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	switch host {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}
