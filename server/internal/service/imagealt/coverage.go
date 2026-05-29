// Package imagealt scans course markdown for image alt-text coverage (plan 12.5).
package imagealt

import (
	"regexp"
	"strings"
)

const decorativeTitleMarker = "lex-decorative"

var markdownImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)`)

// ImageRef is one image occurrence in markdown content.
type ImageRef struct {
	Alt        string
	Src        string
	Title      string
	Decorative bool
	HasValidAlt bool
	Line       int
}

// ScanMarkdown finds markdown images and classifies alt-text status.
func ScanMarkdown(markdown string) []ImageRef {
	lines := strings.Split(markdown, "\n")
	var out []ImageRef
	for lineNo, line := range lines {
		matches := markdownImageRE.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			alt := m[1]
			src := m[2]
			title := ""
			if len(m) >= 4 {
				title = m[3]
			}
			decorative := title == decorativeTitleMarker
			valid := decorative || strings.TrimSpace(alt) != ""
			out = append(out, ImageRef{
				Alt:         alt,
				Src:         src,
				Title:       title,
				Decorative:  decorative,
				HasValidAlt: valid,
				Line:        lineNo + 1,
			})
		}
	}
	return out
}

// Coverage summarizes alt-text status for a slice of images.
type Coverage struct {
	WithAlt int
	Total   int
}

// Summarize counts images with valid alt (including decorative).
func Summarize(images []ImageRef) Coverage {
	total := len(images)
	with := 0
	for _, img := range images {
		if img.HasValidAlt {
			with++
		}
	}
	return Coverage{WithAlt: with, Total: total}
}

// DecorativeTitleMarker is stored in markdown title for decorative images.
func DecorativeTitleMarker() string { return decorativeTitleMarker }
