package objectcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// UserCalendarKey caches a generated iCal body for a user (all courses or scoped).
func UserCalendarKey(userID string, courseID *string) string {
	if courseID != nil && strings.TrimSpace(*courseID) != "" {
		return prefix + "user:" + userID + ":calendar:course:" + strings.TrimSpace(*courseID)
	}
	return prefix + "user:" + userID + ":calendar"
}
