package h5p

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Manifest is the parsed h5p.json from an H5P package.
type Manifest struct {
	Title              string          `json:"title"`
	Language           string          `json:"language"`
	MainLibrary        string          `json:"mainLibrary"`
	EmbedTypes         []string        `json:"embedTypes"`
	License            string          `json:"license"`
	PreloadedDeps      json.RawMessage `json:"preloadedDependencies"`
	ExtraLibraries     []string        `json:"-"`
}

// SupportedMainLibraries lists H5P content types allowed on upload (plan 8.12 FR-8).
var SupportedMainLibraries = map[string]bool{
	"H5P.TrueFalse":           true,
	"H5P.MultiChoice":         true,
	"H5P.DragQuestion":        true,
	"H5P.DragText":            true,
	"H5P.FillInTheBlanks":     true,
	"H5P.InteractiveVideo":    true,
	"H5P.CoursePresentation":  true,
	"H5P.BranchingScenario":   true,
	"H5P.Flashcards":          true,
	"H5P.SingleChoiceSet":     true,
	"H5P.QuestionSet":         true,
	"H5P.ImageHotspots":       true,
	"H5P.Accordion":           true,
	"H5P.Timeline":            true,
	"H5P.Summary":             true,
}

const maxH5PUploadBytes = 50 << 20 // 50 MB

// ParseAndValidateZip reads an .h5p zip from r, parses h5p.json, and validates the main library.
func ParseAndValidateZip(r io.ReaderAt, size int64) (Manifest, json.RawMessage, error) {
	if size <= 0 {
		return Manifest{}, nil, fmt.Errorf("h5p: empty package")
	}
	if size > maxH5PUploadBytes {
		return Manifest{}, nil, fmt.Errorf("h5p: package exceeds maximum size (%d MB)", maxH5PUploadBytes>>20)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("h5p: invalid zip archive")
	}
	var manifestRaw []byte
	found := false
	for _, f := range zr.File {
		if f.Name == "h5p.json" || strings.HasSuffix(f.Name, "/h5p.json") {
			rc, openErr := f.Open()
			if openErr != nil {
				return Manifest{}, nil, fmt.Errorf("h5p: cannot read h5p.json")
			}
			manifestRaw, err = io.ReadAll(io.LimitReader(rc, 1<<20))
			_ = rc.Close()
			if err != nil {
				return Manifest{}, nil, fmt.Errorf("h5p: cannot read h5p.json")
			}
			found = true
			break
		}
	}
	if !found {
		return Manifest{}, nil, fmt.Errorf("h5p: package missing h5p.json manifest")
	}
	var m Manifest
	if err := json.Unmarshal(manifestRaw, &m); err != nil {
		return Manifest{}, nil, fmt.Errorf("h5p: invalid h5p.json")
	}
	lib := strings.TrimSpace(m.MainLibrary)
	if lib == "" {
		return Manifest{}, nil, fmt.Errorf("h5p: h5p.json missing mainLibrary")
	}
	if !SupportedMainLibraries[lib] {
		return Manifest{}, nil, fmt.Errorf("h5p: unsupported content type %q", lib)
	}
	return m, append(json.RawMessage(nil), manifestRaw...), nil
}

// ReadZipBytes loads the full zip into memory for small packages (upload path).
func ReadZipBytes(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("h5p: package exceeds maximum size")
	}
	return data, nil
}

// ZipReaderAt wraps bytes for zip.NewReader.
func ZipReaderAt(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}
