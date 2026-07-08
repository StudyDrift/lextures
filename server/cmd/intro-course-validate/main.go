// Command intro-course-validate lints embedded intro course curriculum fixtures (IC03/IC08).
package main

import (
	"fmt"
	"os"

	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

func main() {
	if err := introcourseservice.ValidateAllLocales(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	locales, err := introcourseservice.ListContentLocales()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list locales: %v\n", err)
		os.Exit(1)
	}
	en, err := introcourseservice.LoadCurriculum("en")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load en: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("intro course content OK: %d modules, content_version=%d, locales=%v\n",
		len(en.Modules), introcourseservice.ContentVersion, locales)
	for _, loc := range locales {
		if loc == "en" {
			continue
		}
		cov, err := introcourseservice.LocaleCoverage(loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "coverage %s: %v\n", loc, err)
			os.Exit(1)
		}
		fmt.Printf("  locale %s coverage: %.0f%%\n", loc, cov*100)
	}
}