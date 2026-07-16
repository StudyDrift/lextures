package board

import "math"

const (
	// sortGapFloor is the minimum gap before renormalization is recommended.
	sortGapFloor = 1e-9
	// defaultSortStep is used when appending after the last item.
	defaultSortStep = 1.0
)

// MidpointSortIndex returns a sort_index between a and b (fractional indexing).
// When a >= b the result is a + defaultSortStep (append-after semantics).
func MidpointSortIndex(a, b float64) float64 {
	if math.IsNaN(a) || math.IsInf(a, 0) {
		a = 0
	}
	if math.IsNaN(b) || math.IsInf(b, 0) {
		return a + defaultSortStep
	}
	if b <= a {
		return a + defaultSortStep
	}
	mid := (a + b) / 2
	if mid-a < sortGapFloor || b-mid < sortGapFloor {
		// Caller should renormalize; still return a usable midpoint.
		return mid
	}
	return mid
}

// AppendSortIndex returns an index after the current max (or 0 when empty).
func AppendSortIndex(maxExisting *float64) float64 {
	if maxExisting == nil {
		return 0
	}
	return *maxExisting + defaultSortStep
}

// PrependSortIndex returns an index before the current min (or 0 when empty).
func PrependSortIndex(minExisting *float64) float64 {
	if minExisting == nil {
		return 0
	}
	return *minExisting - defaultSortStep
}

// NeedsRenormalize reports whether the gap between neighbors is too small.
func NeedsRenormalize(a, b float64) bool {
	if b <= a {
		return true
	}
	return b-a < sortGapFloor*2
}

// RenormalizeSortIndexes returns evenly spaced indexes 0, 1, 2, … for n items.
func RenormalizeSortIndexes(n int) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = float64(i)
	}
	return out
}
