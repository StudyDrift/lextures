package catalogsearch

import (
	"fmt"

	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

// ProviderName is the Schema.org provider organisation name for course markup.
const ProviderName = "Lextures"

// BuildCourseJSONLD produces a Schema.org Course object (with a nested
// CourseInstance and Offer) for a public course landing page (plan 15.1, FR-5).
// baseURL is the public site origin (e.g. "https://lextures.com"), used to build
// the canonical course URL; it may be empty.
func BuildCourseJSONLD(c repoCourse.PublicCatalogCourse, baseURL string) map[string]any {
	ld := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "Course",
		"name":        c.Title,
		"description": c.Description,
		"provider": map[string]any{
			"@type": "Organization",
			"name":  ProviderName,
		},
	}
	if baseURL != "" && c.Slug != "" {
		ld["url"] = baseURL + "/explore/" + c.Slug
	}
	if c.Category != nil && *c.Category != "" {
		ld["about"] = *c.Category
	}
	if c.Language != "" {
		ld["inLanguage"] = c.Language
	}
	if c.AverageRating != nil {
		ld["aggregateRating"] = map[string]any{
			"@type":       "AggregateRating",
			"ratingValue": *c.AverageRating,
			"bestRating":  5,
			"ratingCount": c.EnrollmentCount,
		}
	}

	instance := map[string]any{
		"@type":          "CourseInstance",
		"courseMode":     "online",
		"courseWorkload": "Self-paced",
	}
	if c.InstructorName != nil && *c.InstructorName != "" {
		instance["instructor"] = map[string]any{
			"@type": "Person",
			"name":  *c.InstructorName,
		}
	}
	ld["hasCourseInstance"] = instance

	offer := map[string]any{
		"@type":         "Offer",
		"price":         priceString(c.PriceCents),
		"priceCurrency": "USD",
		"category":      offerCategory(c.PriceCents),
		"availability":  "https://schema.org/InStock",
	}
	ld["offers"] = offer

	return ld
}

func offerCategory(priceCents int) string {
	if priceCents <= 0 {
		return "Free"
	}
	return "Paid"
}

func priceString(priceCents int) string {
	if priceCents < 0 {
		priceCents = 0
	}
	return fmt.Sprintf("%d.%02d", priceCents/100, priceCents%100)
}
