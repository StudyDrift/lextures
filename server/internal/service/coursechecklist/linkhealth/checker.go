package linkhealth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// MaxURLsPerCourse is the hard cap on distinct URLs checked per run (FR-16 / AC-8).
	MaxURLsPerCourse = 200
	// TotalBudget is the wall-clock budget for one course check.
	TotalBudget = 5 * time.Second
	// PerRequestTimeout is the per-URL timeout.
	PerRequestTimeout = 2 * time.Second
	// Concurrency is the max in-flight fetches.
	Concurrency = 8
	// MaxResponseBytes caps body reads (HEAD preferred).
	MaxResponseBytes = 64 * 1024
	// CacheTTL is how long cached results are considered fresh.
	CacheTTL = 24 * time.Hour
	// DefaultUserAgent identifies the crawler (NFR).
	DefaultUserAgent = "LexturesLinkHealth/1.0 (+https://docs.lextures.com/dev/checklist-linkhealth)"
)

// ResultCode is the persisted link-health outcome.
type ResultCode string

const (
	ResultOK      ResultCode = "ok"
	ResultDead    ResultCode = "dead"
	ResultError   ResultCode = "error"
	ResultSkipped ResultCode = "skipped"
)

// CheckResult is one URL outcome.
type CheckResult struct {
	URL        string
	URLHash    []byte
	StatusCode int
	Result     ResultCode
	Reason     string
	CheckedAt  time.Time
}

// Checker performs bounded outbound link checks with SSRF defences.
type Checker struct {
	Client    *http.Client
	UserAgent string
	Now       func() time.Time
	// LookupRobots optionally returns whether the URL path is disallowed.
	// When nil, robots.txt is not consulted (tests may inject a stub).
	LookupRobots func(ctx context.Context, u string) (disallowed bool, err error)
	// HostLimiter optionally rate-limits by host. Returns true when allowed.
	HostLimiter func(host string) bool
}

func (c *Checker) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

func (c *Checker) ua() string {
	if c != nil && strings.TrimSpace(c.UserAgent) != "" {
		return c.UserAgent
	}
	return DefaultUserAgent
}

func (c *Checker) client() *http.Client {
	if c != nil && c.Client != nil {
		return c.Client
	}
	return &http.Client{
		Timeout: PerRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if reason, blocked := IsBlockedHost(req.URL.Hostname()); blocked {
				return fmt.Errorf("redirect blocked: %s", reason)
			}
			// Never forward cookies/auth on redirects.
			req.Header.Del("Cookie")
			req.Header.Del("Authorization")
			return nil
		},
	}
}

// HashURL returns sha256 of the normalized URL string.
func HashURL(raw string) []byte {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return sum[:]
}

// NormalizeURL trims and lowercases scheme/host for dedupe.
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil || u.URL == nil {
		return raw
	}
	u.URL.Fragment = ""
	host := strings.ToLower(u.URL.Hostname())
	if host != "" {
		if u.URL.Port() != "" && u.URL.Port() != "80" && u.URL.Port() != "443" {
			u.URL.Host = host + ":" + u.URL.Port()
		} else {
			u.URL.Host = host
		}
	}
	u.URL.Scheme = strings.ToLower(u.URL.Scheme)
	return u.URL.String()
}

// CheckURLs checks up to MaxURLsPerCourse distinct URLs within TotalBudget.
// Excess URLs are recorded as skipped with reason "cap".
func (c *Checker) CheckURLs(ctx context.Context, urls []string) []CheckResult {
	start := c.now()
	ctx, cancel := context.WithTimeout(ctx, TotalBudget)
	defer cancel()

	seen := make(map[string]struct{}, len(urls))
	var ordered []string
	for _, raw := range urls {
		n := NormalizeURL(raw)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		ordered = append(ordered, n)
	}

	out := make([]CheckResult, 0, len(ordered))
	var overflow []string
	if len(ordered) > MaxURLsPerCourse {
		overflow = ordered[MaxURLsPerCourse:]
		ordered = ordered[:MaxURLsPerCourse]
	}

	type job struct{ url string }
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			res := c.checkOne(ctx, j.url)
			mu.Lock()
			out = append(out, res)
			mu.Unlock()
		}
	}
	nWorkers := Concurrency
	if len(ordered) < nWorkers {
		nWorkers = len(ordered)
	}
	if nWorkers < 1 {
		nWorkers = 1
	}
	wg.Add(nWorkers)
	for i := 0; i < nWorkers; i++ {
		go worker()
	}
	for _, u := range ordered {
		select {
		case <-ctx.Done():
			// Remaining URLs become error/unknown-equivalent skipped for budget.
			mu.Lock()
			out = append(out, CheckResult{
				URL: u, URLHash: HashURL(u), Result: ResultError, Reason: "budget", CheckedAt: c.now(),
			})
			mu.Unlock()
		case jobs <- job{url: u}:
		}
	}
	close(jobs)
	wg.Wait()

	for _, u := range overflow {
		out = append(out, CheckResult{
			URL:       u,
			URLHash:   HashURL(u),
			Result:    ResultSkipped,
			Reason:    "cap",
			CheckedAt: c.now(),
		})
	}
	_ = start
	return out
}

func (c *Checker) checkOne(ctx context.Context, raw string) CheckResult {
	now := c.now()
	res := CheckResult{URL: raw, URLHash: HashURL(raw), CheckedAt: now}

	u, reason, err := ValidateURL(raw)
	if err != nil {
		res.Result = ResultSkipped
		res.Reason = string(reason)
		incBlocked(reason)
		return res
	}
	if c.HostLimiter != nil && !c.HostLimiter(u.Hostname()) {
		res.Result = ResultSkipped
		res.Reason = string(BlockedRateLimit)
		incBlocked(BlockedRateLimit)
		return res
	}
	if c.LookupRobots != nil {
		disallowed, rErr := c.LookupRobots(ctx, raw)
		if rErr == nil && disallowed {
			res.Result = ResultSkipped
			res.Reason = string(BlockedRobots)
			incBlocked(BlockedRobots)
			return res
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, PerRequestTimeout)
	defer cancel()

	status, ferr := c.doRequest(reqCtx, http.MethodHead, raw)
	if ferr != nil || status == http.StatusMethodNotAllowed || status == http.StatusForbidden {
		status, ferr = c.doRequest(reqCtx, http.MethodGet, raw)
	}
	if ferr != nil {
		// Redirect-to-private etc.
		if strings.Contains(ferr.Error(), "blocked") {
			res.Result = ResultSkipped
			res.Reason = string(BlockedPrivateRange)
			incBlocked(BlockedPrivateRange)
			return res
		}
		res.Result = ResultError
		res.Reason = "fetch"
		incURLResult(ResultError)
		return res
	}
	res.StatusCode = status
	if status >= 400 {
		res.Result = ResultDead
		incURLResult(ResultDead)
		return res
	}
	res.Result = ResultOK
	incURLResult(ResultOK)
	return res
}

func (c *Checker) doRequest(ctx context.Context, method, raw string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.ua())
	// Never send cookies or auth.
	req.Header.Del("Cookie")
	req.Header.Del("Authorization")

	resp, err := c.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if method == http.MethodGet {
		_, _ = io.CopyN(io.Discard, resp.Body, MaxResponseBytes)
	} else {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	}
	return resp.StatusCode, nil
}
