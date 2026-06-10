package officepreview

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestLayoutShowsMasterShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		xml  string
		want bool
	}{
		{"default when missing", `<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`, true},
		{"explicit show", `<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" showMasterSp="1"/>`, true},
		{"hide master shapes", `<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" showMasterSp="0"/>`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, err := parsePptxXML([]byte(tc.xml))
			if err != nil {
				t.Fatal(err)
			}
			if got := layoutShowsMasterShapes(root); got != tc.want {
				t.Fatalf("layoutShowsMasterShapes() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPptxSlide1HidesMasterShapesWhenLayoutOptsOut(t *testing.T) {
	path := "../../../../data/course-files/managed-files/C-5CV2AV/a94acbed-1958-4f0c-824b-4c4486ed1e3c.pptx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample pptx not available:", err)
	}
	html, err := ConvertToHTML(data, "week02.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	if err != nil {
		t.Fatal(err)
	}
	slide1 := html
	if idx := strings.Index(html, "Slide 2"); idx > 0 {
		slide1 = html[:idx]
	}

	// Master rounded rectangle is ~576px tall; layout content box is ~280px.
	masterRoundRectH := regexp.MustCompile(`height:5[67][0-9]\.\d+px`)
	layoutContentH := regexp.MustCompile(`height:28[0-9]\.\d+px`)
	if masterRoundRectH.FindString(slide1) != "" {
		t.Fatalf("slide 1 still renders master rounded rectangle height: %s", masterRoundRectH.FindString(slide1))
	}
	if layoutContentH.FindString(slide1) == "" {
		t.Fatalf("slide 1 missing layout content box (~280px tall), got heights: %v", layerHeights(slide1))
	}
	if !strings.Contains(slide1, "Week 02:") || !strings.Contains(slide1, "Schedule") {
		t.Fatalf("slide 1 missing expected text content")
	}
}

func TestPptxSlide4OmitsLayoutPlaceholderPromptText(t *testing.T) {
	path := "../../../../data/course-files/managed-files/C-5CV2AV/a94acbed-1958-4f0c-824b-4c4486ed1e3c.pptx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample pptx not available:", err)
	}
	html, err := ConvertToHTML(data, "week02.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(html, "Slide 4")
	if start < 0 {
		t.Fatal("slide 4 not found")
	}
	slide4 := html[start:]
	if end := strings.Index(slide4, "Slide 5"); end > 0 {
		slide4 = slide4[:end]
	}
	for _, prompt := range []string{
		"Click to edit Master text styles",
		"Second level",
		"Third level",
	} {
		if strings.Contains(slide4, prompt) {
			t.Fatalf("slide 4 still contains placeholder prompt %q", prompt)
		}
	}
	if !strings.Contains(slide4, "Role Prompting - Categories") {
		t.Fatalf("slide 4 missing title text")
	}
}

func layerHeights(fragment string) []string {
	re := regexp.MustCompile(`height:(\d+\.?\d*)px`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(fragment, -1) {
		out = append(out, m[1])
	}
	return out
}
