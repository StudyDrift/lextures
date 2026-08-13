package objectcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

const prefix = "cache:"

// CourseStructureKey caches ListForCourseWithEnrichment output (staff or student base list).
// Assign-to filtering for individual students is applied after a cache hit.
func CourseStructureKey(courseID string, staffView bool) string {
	variant := "student"
	if staffView {
		variant = "staff"
	}
	return prefix + "course:" + courseID + ":structure:" + variant
}

// CourseEnrollmentsKey caches the enrollment roster for a course.
func CourseEnrollmentsKey(courseID string) string {
	return prefix + "course:" + courseID + ":enrollments"
}

// CatalogPageKey caches a public catalog search result page.
func CatalogPageKey(f repoCourse.PublicCatalogFilter) string {
	raw := fmt.Sprintf("q=%s|cat=%s|lang=%s|lvl=%s|sort=%s|lim=%d|off=%d",
		f.Q, f.Category, f.Language, f.Level, f.Sort, f.Limit, f.Offset)
	if f.PriceMax != nil {
		raw += fmt.Sprintf("|pm=%d", *f.PriceMax)
	}
	sum := sha256.Sum256([]byte(raw))
	return prefix + "catalog:page:" + hex.EncodeToString(sum[:8])
}

// MarketplacePageKey caches a marketplace storefront listing page (without ownership).
func MarketplacePageKey(f repoCourse.MarketplaceFilter) string {
	raw := fmt.Sprintf("q=%s|cat=%s|lang=%s|lvl=%s|sort=%s|lim=%d|off=%d|free=%t",
		f.Q, f.Category, f.Language, f.Level, f.Sort, f.Limit, f.Offset, f.FreeOnly)
	if f.PriceMax != nil {
		raw += fmt.Sprintf("|pm=%d", *f.PriceMax)
	}
	sum := sha256.Sum256([]byte(raw))
	return prefix + "marketplace:page:" + hex.EncodeToString(sum[:8])
}

// MarketingContentKey keys immutable public-content snapshots by route, query,
// and the repository content version. Publishing creates a new namespace, so
// stale manifests can never mask newly published content.
func MarketingContentKey(route, query, version string) string {
	sum := sha256.Sum256([]byte(route + "?" + query + "@" + version))
	return prefix + "marketing-content:" + hex.EncodeToString(sum[:8])
}

// HelpContextualKey caches a tiered contextual-help resolution by route and the
// viewer's role set (role-filtered results differ per viewer, so roles are part
// of the key rather than filtered post-cache).
func HelpContextualKey(route string, roles []string, locale string) string {
	sorted := append([]string(nil), roles...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(route + "@" + strings.Join(sorted, ",") + "@" + locale))
	return prefix + "help-contextual:" + hex.EncodeToString(sum[:8])
}

// UserCalendarKey caches a generated iCal body for a user (all courses or scoped).
func UserCalendarKey(userID string, courseID *string) string {
	if courseID != nil && strings.TrimSpace(*courseID) != "" {
		return prefix + "user:" + userID + ":calendar:course:" + strings.TrimSpace(*courseID)
	}
	return prefix + "user:" + userID + ":calendar"
}
