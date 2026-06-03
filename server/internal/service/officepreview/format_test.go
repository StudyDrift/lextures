package officepreview

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		mime     string
		want     Format
		ok       bool
	}{
		{"docx ext", "a.docx", "", FormatDOCX, true},
		{"xlsx ext", "sheet.XLSX", "", FormatXLSX, true},
		{"pptx mime", "slides.bin", "application/vnd.openxmlformats-officedocument.presentationml.presentation", FormatPPTX, true},
		{"legacy doc", "old.doc", "application/msword", "", false},
		{"pdf", "file.pdf", "application/pdf", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DetectFormat(tc.filename, tc.mime)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("DetectFormat(%q, %q) = (%q, %v), want (%q, %v)", tc.filename, tc.mime, got, ok, tc.want, tc.ok)
			}
		})
	}
}
