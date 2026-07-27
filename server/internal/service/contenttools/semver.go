package contenttools

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer is a parsed semantic version (major.minor.patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// ParseSemVer parses a strict major.minor.patch string (prerelease/build ignored for compare).
func ParseSemVer(v string) (SemVer, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return SemVer{}, fmt.Errorf("empty version")
	}
	// Strip build metadata / prerelease for numeric compare.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("version %q is not major.minor.patch", v)
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || maj < 0 || min < 0 || pat < 0 {
		return SemVer{}, fmt.Errorf("version %q has non-integer components", v)
	}
	return SemVer{Major: maj, Minor: min, Patch: pat}, nil
}

// Compare returns -1 if a<b, 0 if equal, 1 if a>b.
func CompareSemVer(a, b SemVer) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// SameMajor reports whether two versions share a major.
func SameMajor(a, b string) bool {
	sa, errA := ParseSemVer(a)
	sb, errB := ParseSemVer(b)
	if errA != nil || errB != nil {
		return false
	}
	return sa.Major == sb.Major
}

// BumpKind is the minimum semver bump required by a schema diff.
type BumpKind string

const (
	BumpNone  BumpKind = "none"
	BumpPatch BumpKind = "patch"
	BumpMinor BumpKind = "minor"
	BumpMajor BumpKind = "major"
)

// ClassifyVersionBump classifies the declared version change from fromVer → toVer.
func ClassifyVersionBump(fromVer, toVer string) (BumpKind, error) {
	a, err := ParseSemVer(fromVer)
	if err != nil {
		return BumpNone, err
	}
	b, err := ParseSemVer(toVer)
	if err != nil {
		return BumpNone, err
	}
	switch {
	case CompareSemVer(a, b) == 0:
		return BumpNone, nil
	case b.Major > a.Major:
		return BumpMajor, nil
	case b.Major < a.Major:
		return BumpNone, fmt.Errorf("version decreased from %s to %s", fromVer, toVer)
	case b.Minor > a.Minor:
		return BumpMinor, nil
	case b.Minor < a.Minor:
		return BumpNone, fmt.Errorf("version decreased from %s to %s", fromVer, toVer)
	case b.Patch > a.Patch:
		return BumpPatch, nil
	default:
		return BumpNone, fmt.Errorf("version decreased from %s to %s", fromVer, toVer)
	}
}

// RequiredBumpRank returns a rank for comparison (none < patch < minor < major).
func RequiredBumpRank(k BumpKind) int {
	switch k {
	case BumpNone:
		return 0
	case BumpPatch:
		return 1
	case BumpMinor:
		return 2
	case BumpMajor:
		return 3
	default:
		return 0
	}
}
