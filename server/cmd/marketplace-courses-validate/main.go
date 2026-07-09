// Command marketplace-courses-validate lints embedded official marketplace course fixtures (MC0).
package main

import (
	"fmt"
	"os"

	mcservice "github.com/lextures/lextures/server/internal/service/marketplacecourses"
)

func main() {
	if err := mcservice.ValidateAllCourses(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	slugs, err := mcservice.ListCourseSlugs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list courses: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("marketplace courses content OK: %d course(s)\n", len(slugs))
	for _, dir := range slugs {
		spec, err := mcservice.LoadCourseSpec(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("  %s code=%s catalog_slug=%s modules=%d content_version=%d\n",
			dir, spec.Manifest.Code, spec.Manifest.CatalogSlug, len(spec.Modules), spec.Manifest.ContentVersion)
	}
	urls, err := mcservice.ExtractExternalURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract urls: %v\n", err)
		os.Exit(1)
	}
	if len(urls) > 0 {
		fmt.Printf("external URLs to link-check (%d):\n", len(urls))
		for _, u := range urls {
			fmt.Printf("  %s\n", u)
		}
	}
}
