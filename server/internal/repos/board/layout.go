package board

import (
	"fmt"
	"strings"
)

// Layout mode identifiers (FR-1).
const (
	LayoutWall     = "wall"
	LayoutStream   = "stream"
	LayoutGrid     = "grid"
	LayoutColumns  = "columns"
	LayoutCanvas   = "canvas"
	LayoutTimeline = "timeline"
	LayoutMap      = "map"
)

// UnsortedSectionTitle is the default section used when switching to columns
// or when a section is deleted (FR-3 / AC-1).
const UnsortedSectionTitle = "Unsorted"

var validLayouts = map[string]struct{}{
	LayoutWall:     {},
	LayoutStream:   {},
	LayoutGrid:     {},
	LayoutColumns:  {},
	LayoutCanvas:   {},
	LayoutTimeline: {},
	LayoutMap:      {},
}

// NormalizeLayout returns a canonical layout string or an error.
func NormalizeLayout(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return LayoutWall, nil
	}
	if _, ok := validLayouts[s]; !ok {
		return "", fmt.Errorf("board: layout must be one of wall, stream, grid, columns, canvas, timeline, map")
	}
	return s, nil
}

// IsValidLayout reports whether s is a known layout mode.
func IsValidLayout(s string) bool {
	_, ok := validLayouts[strings.TrimSpace(strings.ToLower(s))]
	return ok
}
