// Package contentdoc is the shared authored-content model for checklist accessibility rules (CC.6 FR-22).
package contentdoc

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/imagealt"
	"github.com/lextures/lextures/server/internal/service/readinglevel"
)

// MaxAuthoredBytesPerCourse caps total markdown scanned into a Doc (FR-22 / NFR).
const MaxAuthoredBytesPerCourse = 4 * 1024 * 1024

// Doc is the shared authored-content model for accessibility rules (FR-22).
type Doc struct {
	Pages      []Page
	ParseCount int
}

// Page is one authored surface (content page, assignment/quiz body, syllabus section).
type Page struct {
	ItemID      uuid.UUID
	Kind        string
	Title       string
	ModuleTitle string
	Route       string
	Images      []Image
	Headings    []Heading
	Links       []Link
	Tables      []Table
	Media       []Media
	HasBlinking bool
	AllCaps     []string
	PlainText   string
	Modalities  map[string]bool
}

// Image is one image occurrence.
type Image struct {
	Src         string
	Alt         string
	Decorative  bool
	HasValidAlt bool
	Line        int
}

// Heading is one ATX/HTML heading.
type Heading struct {
	Level         int
	Text          string
	Line          int
	BoldAsHeading bool
}

// Link is one hyperlink.
type Link struct {
	Text string
	Href string
	Line int
}

// Table summarizes a markdown/HTML table.
type Table struct {
	HasHeader  bool
	Rows       int
	Cols       int
	LayoutOnly bool
	Line       int
}

// Media is embedded time-based media.
type Media struct {
	Kind                    string
	Src                     string
	HasCaptionsOrTranscript bool
	Line                    int
}

// Source is one markdown body to parse into the shared Doc.
type Source struct {
	ItemID      uuid.UUID
	Kind        string
	Title       string
	ModuleTitle string
	Route       string
	Markdown    string
}

var (
	atxHeadingRE  = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	mdLinkRE      = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	bareURLRE     = regexp.MustCompile(`(?i)\bhttps?://[^\s<>\[\]()]{8,}`)
	htmlHeadingRE = regexp.MustCompile(`(?i)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	htmlTableRE   = regexp.MustCompile(`(?is)<table\b[^>]*>(.*?)</table>`)
	htmlThRE      = regexp.MustCompile(`(?i)<th\b`)
	htmlTrRE      = regexp.MustCompile(`(?i)<tr\b`)
	htmlTdRE      = regexp.MustCompile(`(?i)<td\b`)
	videoEmbedRE  = regexp.MustCompile(`(?i)(youtube\.com|youtu\.be|vimeo\.com|<video\b|<audio\b|\.mp4\b|\.webm\b|\.mp3\b|\.m4a\b)`)
	captionHintRE = regexp.MustCompile(`(?i)(caption|transcript|closed captions?|cc:)`)
	blinkRE       = regexp.MustCompile(`(?i)<(blink|marquee)\b`)
	fontSizeRE    = regexp.MustCompile(`(?i)(font-size\s*:\s*\d+(\.\d+)?(px|pt)|style=["\'][^"\']*font-size)`)
	boldLineRE    = regexp.MustCompile(`(?m)^\*\*([^*].{2,80}[^*])\*\*\s*$`)
	mdTableSepRE  = regexp.MustCompile(`(?m)^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?\s*$`)
)

// Parse builds a Doc from sources with a course-wide byte budget.
func Parse(sources []Source) *Doc {
	doc := &Doc{ParseCount: 1}
	budget := MaxAuthoredBytesPerCourse
	for _, src := range sources {
		md := src.Markdown
		if budget <= 0 {
			break
		}
		if len(md) > budget {
			md = md[:budget]
		}
		budget -= len(md)
		if strings.TrimSpace(md) == "" {
			continue
		}
		page := parsePage(Page{
			ItemID: src.ItemID, Kind: src.Kind, Title: src.Title,
			ModuleTitle: src.ModuleTitle, Route: src.Route,
		}, md)
		doc.Pages = append(doc.Pages, page)
	}
	return doc
}

// PageRoute returns the checklist nav route template for a structure kind.
func PageRoute(kind string) string {
	switch kind {
	case "content_page":
		return "/courses/{courseCode}/modules/content/{itemId}"
	case "assignment":
		return "/courses/{courseCode}/modules/assignment/{itemId}"
	case "quiz":
		return "/courses/{courseCode}/modules/quiz/{itemId}"
	default:
		return "/courses/{courseCode}/modules"
	}
}

func parsePage(base Page, md string) Page {
	base.Modalities = map[string]bool{"text": strings.TrimSpace(md) != ""}
	for _, img := range imagealt.ScanMarkdown(md) {
		base.Images = append(base.Images, Image{
			Src: img.Src, Alt: img.Alt, Decorative: img.Decorative,
			HasValidAlt: img.HasValidAlt, Line: img.Line,
		})
	}
	base.Headings = extractHeadings(md)
	base.Links = extractLinks(md)
	base.Tables = extractTables(md)
	base.Media = extractMedia(md)
	if len(base.Media) > 0 {
		base.Modalities["media"] = true
	}
	base.HasBlinking = blinkRE.MatchString(md)
	base.AllCaps = findAllCapsBlocks(md)
	_ = fontSizeRE // reserved for future platform-min font-size checks (FR-8)
	base.PlainText = readinglevel.PlainTextFromMarkdown(md)
	// Interactive markers: H5P/SCORM/tool mentions are handled at module level elsewhere.
	if strings.Contains(strings.ToLower(md), "h5p") || strings.Contains(md, "content-tool") {
		base.Modalities["interactive"] = true
	}
	return base
}

func extractHeadings(md string) []Heading {
	var out []Heading
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		if m := atxHeadingRE.FindStringSubmatch(line); len(m) == 3 {
			out = append(out, Heading{Level: len(m[1]), Text: strings.TrimSpace(m[2]), Line: i + 1})
			continue
		}
		if m := htmlHeadingRE.FindStringSubmatch(line); len(m) == 3 {
			lvl := int(m[1][0] - '0')
			out = append(out, Heading{Level: lvl, Text: stripTags(m[2]), Line: i + 1})
			continue
		}
		if m := boldLineRE.FindStringSubmatch(line); len(m) == 2 {
			text := strings.TrimSpace(m[1])
			if looksLikeHeadingText(text) {
				out = append(out, Heading{Level: 0, Text: text, Line: i + 1, BoldAsHeading: true})
			}
		}
	}
	return out
}

func looksLikeHeadingText(s string) bool {
	if utf8.RuneCountInString(s) < 3 || utf8.RuneCountInString(s) > 80 {
		return false
	}
	// Avoid flagging bold emphasis mid-sentence endings.
	if strings.HasSuffix(s, ".") && utf8.RuneCountInString(s) > 40 {
		return false
	}
	return true
}

func extractLinks(md string) []Link {
	var out []Link
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		for _, m := range mdLinkRE.FindAllStringSubmatch(line, -1) {
			if len(m) < 3 {
				continue
			}
			// Skip images (handled separately); mdLinkRE also matches image alt form after !.
			if strings.Contains(line, "![") && strings.Contains(line, m[0]) {
				idx := strings.Index(line, m[0])
				if idx > 0 && line[idx-1] == '!' {
					continue
				}
			}
			out = append(out, Link{Text: m[1], Href: m[2], Line: i + 1})
		}
		for _, m := range bareURLRE.FindAllString(line, -1) {
			out = append(out, Link{Text: m, Href: m, Line: i + 1})
		}
	}
	return out
}

func extractTables(md string) []Table {
	var out []Table
	// Markdown pipe tables
	lines := strings.Split(md, "\n")
	for i := 0; i < len(lines); i++ {
		if !strings.Contains(lines[i], "|") {
			continue
		}
		if i+1 >= len(lines) || !mdTableSepRE.MatchString(lines[i+1]) {
			continue
		}
		cols := strings.Count(lines[i+1], "|")
		if strings.HasPrefix(strings.TrimSpace(lines[i+1]), "|") {
			cols--
		}
		if strings.HasSuffix(strings.TrimSpace(lines[i+1]), "|") {
			cols--
		}
		if cols < 1 {
			cols = strings.Count(lines[i], "|")
		}
		rows := 1 // header
		for j := i + 2; j < len(lines); j++ {
			if !strings.Contains(lines[j], "|") || strings.TrimSpace(lines[j]) == "" {
				break
			}
			rows++
		}
		layout := rows <= 1 || cols <= 1
		out = append(out, Table{
			HasHeader: true, Rows: rows, Cols: cols, LayoutOnly: layout, Line: i + 1,
		})
		i += rows
	}
	// HTML tables
	for _, m := range htmlTableRE.FindAllStringSubmatch(md, -1) {
		body := m[1]
		hasHeader := htmlThRE.MatchString(body)
		rows := len(htmlTrRE.FindAllStringIndex(body, -1))
		cols := 0
		if td := htmlTdRE.FindAllStringIndex(body, -1); len(td) > 0 && rows > 0 {
			cols = len(td) / rows
		}
		layout := !hasHeader && (rows <= 1 || cols <= 1)
		out = append(out, Table{
			HasHeader: hasHeader, Rows: rows, Cols: cols, LayoutOnly: layout, Line: 0,
		})
	}
	return out
}

func extractMedia(md string) []Media {
	if !videoEmbedRE.MatchString(md) {
		return nil
	}
	hasAlt := captionHintRE.MatchString(md)
	var out []Media
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		if videoEmbedRE.MatchString(line) {
			kind := "video"
			if strings.Contains(strings.ToLower(line), "audio") || strings.Contains(strings.ToLower(line), ".mp3") {
				kind = "audio"
			}
			out = append(out, Media{
				Kind: kind, Src: strings.TrimSpace(line),
				HasCaptionsOrTranscript: hasAlt || captionHintRE.MatchString(line),
				Line: i + 1,
			})
		}
	}
	return out
}

func findAllCapsBlocks(md string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if utf8.RuneCountInString(s) > 80 && isMostlyUpper(s) {
			out = append(out, s)
		}
	}
	for _, r := range md {
		if r == '\n' {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

func isMostlyUpper(s string) bool {
	letters, upper := 0, 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	return letters >= 20 && upper*100/letters >= 80
}

func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
