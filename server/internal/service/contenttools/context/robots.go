package context

import (
	"bufio"
	stdctx "context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type robotsCacheEntry struct {
	disallow []string
	fetched  time.Time
}

var (
	robotsMu    sync.Mutex
	robotsCache = map[string]robotsCacheEntry{}
	robotsTTL   = 1 * time.Hour
)

// RobotsAllowed reports whether UserAgent may fetch path on host (FR-5 / RFC 9309).
// Fail-open on network errors so a broken robots.txt cannot brick ingestion.
func RobotsAllowed(ctx stdctx.Context, client *http.Client, rawURL string) (bool, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}
	origin := u.Scheme + "://" + u.Host
	path := u.Path
	if path == "" {
		path = "/"
	}

	robotsMu.Lock()
	if ent, ok := robotsCache[origin]; ok && time.Since(ent.fetched) < robotsTTL {
		dis := ent.disallow
		robotsMu.Unlock()
		return !pathMatched(dis, path), nil
	}
	robotsMu.Unlock()

	robotsURL := origin + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return true, nil
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return true, nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return true, nil
	}
	disallow := parseRobots(io.LimitReader(resp.Body, 256*1024), UserAgent)
	robotsMu.Lock()
	robotsCache[origin] = robotsCacheEntry{disallow: disallow, fetched: time.Now()}
	robotsMu.Unlock()
	return !pathMatched(disallow, path), nil
}

func parseRobots(r io.Reader, ua string) []string {
	ua = strings.ToLower(ua)
	var (
		active   bool
		global   bool
		disallow []string
		starDis  []string
	)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "user-agent:"):
			val := strings.TrimSpace(line[len("User-agent:"):])
			valLower := strings.ToLower(val)
			active = valLower == "*" || strings.Contains(ua, valLower) || strings.Contains(valLower, "lextures")
			global = valLower == "*"
		case active && strings.HasPrefix(lower, "disallow:"):
			val := strings.TrimSpace(line[len("Disallow:"):])
			if val == "" {
				continue
			}
			if global {
				starDis = append(starDis, val)
			} else {
				disallow = append(disallow, val)
			}
		}
	}
	if len(disallow) == 0 {
		return starDis
	}
	return disallow
}

func pathMatched(prefixes []string, path string) bool {
	for _, p := range prefixes {
		if p == "/" {
			return true
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
