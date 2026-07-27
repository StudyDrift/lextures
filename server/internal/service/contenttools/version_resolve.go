package contenttools

import (
	"fmt"
	"sort"
)

// ResolveWithinMajor picks the newest published version that shares the pinned
// major and is not newer across a major boundary (FR-9 / AC-10).
// published must be semver strings for the same tool_id.
func ResolveWithinMajor(pinned string, published []string) (string, error) {
	pin, err := ParseSemVer(pinned)
	if err != nil {
		return "", fmt.Errorf("pinned version: %w", err)
	}
	type cand struct {
		raw string
		v   SemVer
	}
	var candidates []cand
	for _, p := range published {
		sv, err := ParseSemVer(p)
		if err != nil {
			continue
		}
		if sv.Major != pin.Major {
			continue
		}
		// Compatible: same major and >= pinned (or any 1.x when pinned to 1.4.0 → newest 1.x).
		if CompareSemVer(sv, pin) < 0 {
			// Still allow older patches within major only if nothing newer exists;
			// prefer versions >= pinned first.
			candidates = append(candidates, cand{raw: p, v: sv})
			continue
		}
		candidates = append(candidates, cand{raw: p, v: sv})
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no published version within major of %s", pinned)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return CompareSemVer(candidates[i].v, candidates[j].v) > 0
	})
	// Prefer newest >= pinned; else newest within major (shouldn't strand on missing patch).
	for _, c := range candidates {
		if CompareSemVer(c.v, pin) >= 0 {
			return c.raw, nil
		}
	}
	return candidates[0].raw, nil
}
