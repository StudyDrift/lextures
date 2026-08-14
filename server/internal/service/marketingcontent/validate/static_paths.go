package validate

import "strings"

// StaticSitePaths are first-party marketing-site routes that always resolve.
// They exist in www/src/lib/route-manifest.tsx as static pages, independent of
// published CMS articles and of the post-deploy known-paths sync.
var StaticSitePaths = []string{
	"/",
	"/about",
	"/press",
	"/authors",
	"/get-started",
	"/parents",
	"/higher-ed",
	"/k12",
	"/homeschool",
	"/pricing",
	"/pricing/calculator",
	"/courses",
	"/request-information",
	"/platform",
	"/platform/adaptive-learning",
	"/platform/assessment",
	"/platform/grading",
	"/platform/analytics",
	"/platform/accessibility",
	"/platform/ai",
	"/resources",
	"/guides",
	"/research",
	"/trust",
	"/compare",
	"/alternatives",
	"/integrations",
	"/glossary",
	"/standards",
	"/templates",
	"/tools",
	"/resources/guides",
	"/resources/research",
	"/resources/research/methodology",
	"/blog",
	"/docs",
	"/privacy",
	"/privacy/history",
	"/terms",
	"/terms/history",
	"/security",
	"/accessibility",
	"/accessibility/vpat",
	"/privacy-rights/california",
}

// WithStaticSitePaths unions catalog paths with always-valid marketing hubs.
func WithStaticSitePaths(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in)+len(StaticSitePaths))
	for path := range in {
		if normalized := normalizeInternalPath(path); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	for _, path := range StaticSitePaths {
		out[path] = struct{}{}
	}
	return out
}

func normalizeInternalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func pathIsKnown(known map[string]struct{}, path string) bool {
	_, ok := known[normalizeInternalPath(path)]
	return ok
}
