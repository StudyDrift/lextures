package context

import (
	stdctx "context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// FetchOutcome is the result of an SSRF-guarded GET (FR-4–FR-6).
type FetchOutcome struct {
	FinalURL     string
	StatusCode   int
	ContentType  string
	Body         []byte
	ETag         string
	LastModified string
	NotModified  bool
}

// KillSwitchActive reports CONTENT_TOOLS_LINK_INGEST_KILL_SWITCH.
func KillSwitchActive() bool {
	v := strings.TrimSpace(os.Getenv("CONTENT_TOOLS_LINK_INGEST_KILL_SWITCH"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// NewFetchClient builds an HTTP client with pinned dial + redirect SSRF re-check.
func NewFetchClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil, // never honour env proxy for SSRF clarity; egress is documented separately
		DialContext: func(ctx stdctx.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			d.Timeout = FetchTimeout
			// Re-validate before dial (DNS rebinding window).
			host, _, err := net.SplitHostPort(addr)
			if err == nil {
				if err := ValidateFetchURL("https://" + host); err != nil {
					// Validate with synthetic URL; scheme irrelevant for host check.
					if ip := parseLiteralIP(host); ip != nil && isBlockedIP(ip) {
						return nil, ErrSSRFBlocked
					}
					if ip := parseLiteralIP(host); ip == nil {
						ips, lerr := LookupIPs(host)
						if lerr != nil {
							return nil, lerr
						}
						for _, ip := range ips {
							if isBlockedIP(ip) {
								return nil, ErrSSRFBlocked
							}
						}
					}
				}
			}
			return PinDial(network, addr)
		},
		ResponseHeaderTimeout: FetchTimeout,
		DisableKeepAlives:     true,
	}
	return &http.Client{
		Timeout:   FetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("contenttools/context: too many redirects")
			}
			if err := ValidateFetchURL(req.URL.String()); err != nil {
				return err
			}
			// Never forward cookies/auth on redirects.
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
			return nil
		},
	}
}

// FetchURL performs a conditional GET with SSRF guards (FR-4, FR-6, AC-3, AC-12).
func FetchURL(ctx stdctx.Context, rawURL, etag, lastModified string) (*FetchOutcome, error) {
	return FetchURLWithClient(ctx, NewFetchClient(), rawURL, etag, lastModified, true)
}

// FetchURLWithClient is like FetchURL but uses the provided client.
// When validate is false, SSRF host checks are skipped (tests with httptest only).
func FetchURLWithClient(ctx stdctx.Context, client *http.Client, rawURL, etag, lastModified string, validate bool) (*FetchOutcome, error) {
	if KillSwitchActive() {
		return nil, ErrKillSwitch
	}
	if validate {
		if err := ValidateFetchURL(rawURL); err != nil {
			return nil, err
		}
	}
	host := HostOf(rawURL)
	if HostBreakerOpen(host) {
		return nil, ErrHostBreakerOpen
	}
	if client == nil {
		client = NewFetchClient()
	}
	ok, err := RobotsAllowed(ctx, client, rawURL)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrRobotsDisallowed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/pdf,*/*;q=0.8")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, ErrSSRFBlocked) {
			HostBreakerRecordFailure(host)
			return nil, err
		}
		HostBreakerRecordFailure(host)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	out := &FetchOutcome{
		FinalURL:     resp.Request.URL.String(),
		StatusCode:   resp.StatusCode,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}
	if validate {
		if err := ValidateFetchURL(out.FinalURL); err != nil {
			HostBreakerRecordFailure(host)
			return nil, err
		}
	}
	if resp.StatusCode == http.StatusNotModified {
		out.NotModified = true
		HostBreakerRecordSuccess(host)
		return out, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		HostBreakerRecordFailure(host)
		return out, fmt.Errorf("contenttools/context: upstream status %d", resp.StatusCode)
	}
	// Reject oversize via Content-Length when present (AC-12).
	if resp.ContentLength > MaxFetchBytes {
		HostBreakerRecordFailure(host)
		return nil, ErrTooLarge
	}
	limited := io.LimitReader(resp.Body, int64(MaxFetchBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		HostBreakerRecordFailure(host)
		return nil, err
	}
	if len(body) > MaxFetchBytes {
		HostBreakerRecordFailure(host)
		return nil, ErrTooLarge
	}
	out.Body = body
	HostBreakerRecordSuccess(host)
	return out, nil
}

// ReasonForFetchError maps errors to instructor-visible reason codes.
func ReasonForFetchError(err error) (status, reason string) {
	switch {
	case err == nil:
		return StatusReady, ""
	case errors.Is(err, ErrSSRFBlocked):
		return StatusBlocked, "blocked: private network"
	case errors.Is(err, ErrHostNotAllowlisted):
		return StatusBlocked, "blocked: host not allowlisted"
	case errors.Is(err, ErrIngestDisabled), errors.Is(err, ErrKillSwitch):
		return StatusBlocked, "blocked: ingestion disabled"
	case errors.Is(err, ErrRobotsDisallowed):
		return StatusBlocked, "blocked: robots.txt"
	case errors.Is(err, ErrTooLarge):
		return StatusUnsupported, "unsupported: exceeds size cap"
	case errors.Is(err, ErrUnsupportedType):
		return StatusUnsupported, "unsupported: content type"
	case errors.Is(err, ErrHostBreakerOpen):
		return StatusFailed, "unavailable: host circuit breaker open"
	default:
		return StatusFailed, "failed: " + truncate(err.Error(), 200)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Sleep is overridable in tests.
var Sleep = time.Sleep
