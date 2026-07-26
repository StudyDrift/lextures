package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/auth"
)

// Auth posture classes for the route inventory golden (TD.1 FR-4 / AC-6).
//
// Classification is derived by probing, not annotations:
//   - anonymous: unauthenticated request does not receive 401
//   - session:   unauthenticated request receives 401 (any signed-in principal required)
//
// Elevated (admin/org-admin) scope is covered by characterization fixtures for
// high-risk admin endpoints. Full elevated matrix for every mutator would require
// side-effect-safe DB probing and is intentionally out of the fast inventory path.
const (
	authAnonymous = "anonymous"
	authSession   = "session"
)

// routeInventoryEntry is one registered route as recorded in the golden file.
type routeInventoryEntry struct {
	Method  string
	Pattern string
	Auth    string
}

func (e routeInventoryEntry) line() string {
	return e.Method + "\t" + e.Pattern + "\t" + e.Auth
}

func parseRouteInventoryLine(line string) (routeInventoryEntry, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return routeInventoryEntry{}, fmt.Errorf("want METHOD\\tPATTERN\\tAUTH, got %q", line)
	}
	return routeInventoryEntry{Method: parts[0], Pattern: parts[1], Auth: parts[2]}, nil
}

func parseRouteInventory(text string) ([]routeInventoryEntry, error) {
	var out []routeInventoryEntry
	for i, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		e, err := parseRouteInventoryLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func formatRouteInventory(entries []routeInventoryEntry) string {
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.line()
	}
	return strings.Join(lines, "\n") + "\n"
}

// inventoryTestHandler builds the production router with minimal deps for
// route walking and unauthenticated auth-posture probes (no DB, no Redis).
func inventoryTestHandler(t *testing.T) http.Handler {
	t.Helper()
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	return NewHandler(Deps{JWTSigner: signer})
}

func inventoryChiRoutes(t *testing.T, h http.Handler) chi.Routes {
	t.Helper()
	routes, ok := h.(chi.Routes)
	if !ok {
		t.Fatalf("NewHandler must return chi.Routes for inventory walk; got %T", h)
	}
	return routes
}

// collectRoutePatterns walks the chi router and returns sorted (method, pattern) pairs.
func collectRoutePatterns(t *testing.T, routes chi.Routes) []routeInventoryEntry {
	t.Helper()
	var list []routeInventoryEntry
	err := chi.Walk(routes, func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		list = append(list, routeInventoryEntry{Method: method, Pattern: pattern})
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Pattern != list[j].Pattern {
			return list[i].Pattern < list[j].Pattern
		}
		return list[i].Method < list[j].Method
	})
	return list
}

var pathParamRE = regexp.MustCompile(`\{[^/}]+\}`)

// materializePath turns a chi pattern into a concrete path for httptest probes.
// Path params become "x"; trailing wildcards become "x".
func materializePath(pattern string) string {
	p := pathParamRE.ReplaceAllString(pattern, "x")
	p = strings.ReplaceAll(p, "*", "x")
	if p == "" {
		return "/"
	}
	return p
}

// probeIPSeq assigns a unique synthetic client IP to each inventory probe so
// process-wide rate limiters (global middleware + institution-inquiry form) do
// not trip when the suite walks 1k+ routes. Empty RemoteAddr would otherwise
// share one bucket and flake parallel nodb tests with 429.
var probeIPSeq atomic.Uint64

// probeAuthPosture classifies a route by issuing an unauthenticated request.
// 401 => session; anything else => anonymous (auth gate did not block).
func probeAuthPosture(h http.Handler, method, pattern string) string {
	path := materializePath(pattern)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept-Language", "en")
	n := probeIPSeq.Add(1)
	// 10.a.b.c — unique per probe, valid for onboardingRealIP / ratelimit.ClientIP.
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1", (n>>16)&0xff, (n>>8)&0xff, n&0xff)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		return authSession
	}
	return authAnonymous
}

// buildRouteInventory walks the live router and probes auth posture for every route.
func buildRouteInventory(t *testing.T) []routeInventoryEntry {
	t.Helper()
	h := inventoryTestHandler(t)
	list := collectRoutePatterns(t, inventoryChiRoutes(t, h))
	for i := range list {
		list[i].Auth = probeAuthPosture(h, list[i].Method, list[i].Pattern)
	}
	return list
}

// diffRouteInventory returns human-readable added/removed/changed lines (TD.1 FR-9).
func diffRouteInventory(want, got []routeInventoryEntry) (added, removed, changed []string) {
	type key struct{ m, p string }
	wantMap := make(map[key]string, len(want))
	gotMap := make(map[key]string, len(got))
	for _, e := range want {
		wantMap[key{e.Method, e.Pattern}] = e.Auth
	}
	for _, e := range got {
		gotMap[key{e.Method, e.Pattern}] = e.Auth
	}
	for k, ga := range gotMap {
		wa, ok := wantMap[k]
		if !ok {
			added = append(added, fmt.Sprintf("+ %s %s (%s)", k.m, k.p, ga))
			continue
		}
		if wa != ga {
			changed = append(changed, fmt.Sprintf("~ %s %s auth %s -> %s", k.m, k.p, wa, ga))
		}
	}
	for k, wa := range wantMap {
		if _, ok := gotMap[k]; !ok {
			removed = append(removed, fmt.Sprintf("- %s %s (%s)", k.m, k.p, wa))
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	return added, removed, changed
}

func updateGoldenRequested() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("UPDATE_GOLDEN")))
	return v == "1" || v == "true" || v == "yes"
}
