package marketingcontentimport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

type sourceArticle struct {
	File, Kind, Slug, Category string
	Meta                       frontmatter
}

func loadArticles(root, only string) ([]sourceArticle, error) {
	var patterns []struct{ kind, glob string }
	if only == "" || only == "blog" {
		patterns = append(patterns, struct{ kind, glob string }{"blog", filepath.Join(root, "blog", "*.md")})
	}
	if only == "" || only == "docs" {
		patterns = append(patterns, struct{ kind, glob string }{"doc", filepath.Join(root, "docs", "*", "*.md")})
	}
	var out []sourceArticle
	for _, p := range patterns {
		files, err := filepath.Glob(p.glob)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			raw, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			fm, err := parseFrontmatter(file, string(raw))
			if err != nil {
				return nil, err
			}
			slug := strings.TrimSuffix(filepath.Base(file), ".md")
			category := fm.Values["category"]
			if p.kind == "doc" && category == "" {
				category = filepath.Base(filepath.Dir(file))
			}
			out = append(out, sourceArticle{File: file, Kind: p.kind, Slug: slug, Category: category, Meta: fm})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func loadCategories(root string) ([]repo.Category, error) {
	b, err := os.ReadFile(filepath.Join(root, "docs", "_categories.ts"))
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\['([^']+)',\s*'([^']*)',\s*'([^']*)',\s*([0-9]+),\s*'([^']*)'\]`)
	var out []repo.Category
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		n, _ := strconv.Atoi(m[4])
		out = append(out, repo.Category{Slug: m[1], Locale: "en", Title: m[2], Description: m[3], SortOrder: n, PlatformPath: m[5]})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no categories parsed from %s", filepath.Join(root, "docs", "_categories.ts"))
	}
	return out, nil
}

func loadAuthors(root string) ([]repo.Author, error) {
	b, err := os.ReadFile(filepath.Join(root, "lib", "authors.ts"))
	if err != nil {
		return nil, err
	}
	s := string(b)
	start := strings.Index(s, "export const AUTHORS")
	if start < 0 {
		return nil, fmt.Errorf("AUTHORS registry not found")
	}
	s = s[start:]
	objRE := regexp.MustCompile(`(?s)\{\s*slug:\s*'([^']+)'(.*?)\n\s*\}`)
	field := func(body, name string) string {
		re := regexp.MustCompile(`(?s)\b` + regexp.QuoteMeta(name) + `:\s*'([^']*)'`)
		m := re.FindStringSubmatch(body)
		if len(m) > 1 {
			return m[1]
		}
		return ""
	}
	array := func(body, name string) []string {
		re := regexp.MustCompile(`(?s)\b` + regexp.QuoteMeta(name) + `:\s*\[(.*?)\]`)
		m := re.FindStringSubmatch(body)
		if len(m) < 2 {
			return []string{}
		}
		q := regexp.MustCompile(`'([^']*)'`)
		var v []string
		for _, x := range q.FindAllStringSubmatch(m[1], -1) {
			v = append(v, x[1])
		}
		return v
	}
	var out []repo.Author
	for _, m := range objRE.FindAllStringSubmatch(s, -1) {
		body := m[2]
		links, _ := json.Marshal(map[string]any{"sameAs": array(body, "sameAs"), "alumniOf": array(body, "alumniOf"), "credentials": array(body, "credentials"), "consentRecordedAt": field(body, "consentRecordedAt")})
		out = append(out, repo.Author{Slug: m[1], Name: field(body, "name"), JobTitle: field(body, "jobTitle"), Bio: field(body, "bio"), Status: field(body, "status"), KnowsAbout: array(body, "knowsAbout"), Links: links})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no authors parsed from registry")
	}
	return out, nil
}
