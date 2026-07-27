package media_checkpoints

import (
	"strconv"
)

const (
	// DefaultGranularitySec is the coarse watch-segment quantum (FR-7 / FR-8).
	DefaultGranularitySec = 5.0
	// MaxWatchedSegments caps stored ranges after merge.
	MaxWatchedSegments = 200
)

// NormalizeSegments coarsens, merges overlapping/adjacent ranges, and caps count.
// Granularity floors start and ceils end to the given second quantum (min 1).
func NormalizeSegments(segments [][2]float64, granularitySec float64) [][2]float64 {
	if granularitySec < 1 {
		granularitySec = DefaultGranularitySec
	}
	cleaned := make([][2]float64, 0, len(segments))
	for _, seg := range segments {
		start, end := seg[0], seg[1]
		if end < start {
			start, end = end, start
		}
		if end <= 0 || end-start < 0.01 {
			continue
		}
		if start < 0 {
			start = 0
		}
		start = floorToQuantum(start, granularitySec)
		end = ceilToQuantum(end, granularitySec)
		if end <= start {
			end = start + granularitySec
		}
		cleaned = append(cleaned, [2]float64{start, end})
	}
	if len(cleaned) == 0 {
		return [][2]float64{}
	}
	// Insertion sort keeps the dependency surface small for this hot helper.
	for i := 1; i < len(cleaned); i++ {
		j := i
		for j > 0 && (cleaned[j][0] < cleaned[j-1][0] ||
			(cleaned[j][0] == cleaned[j-1][0] && cleaned[j][1] < cleaned[j-1][1])) {
			cleaned[j], cleaned[j-1] = cleaned[j-1], cleaned[j]
			j--
		}
	}
	merged := [][2]float64{cleaned[0]}
	for i := 1; i < len(cleaned); i++ {
		cur := cleaned[i]
		last := &merged[len(merged)-1]
		// Merge overlapping or adjacent (touching) ranges.
		if cur[0] <= last[1]+granularitySec*0.01 {
			if cur[1] > last[1] {
				last[1] = cur[1]
			}
			continue
		}
		merged = append(merged, cur)
	}
	if len(merged) > MaxWatchedSegments {
		merged = merged[len(merged)-MaxWatchedSegments:]
	}
	return merged
}

// WatchedBins returns coarse time-bin labels covered by segments (e.g. "0-5").
func WatchedBins(segments [][2]float64, granularitySec float64) []string {
	if granularitySec < 1 {
		granularitySec = DefaultGranularitySec
	}
	norm := NormalizeSegments(segments, granularitySec)
	if len(norm) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, seg := range norm {
		for t := seg[0]; t < seg[1]-1e-9; t += granularitySec {
			start := floorToQuantum(t, granularitySec)
			end := start + granularitySec
			label := strconv.FormatFloat(start, 'f', -1, 64) + "-" + strconv.FormatFloat(end, 'f', -1, 64)
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	// Insertion sort for stable deterministic output.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j] < out[j-1] {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}

func floorToQuantum(v, q float64) float64 {
	if v <= 0 {
		return 0
	}
	n := int(v / q)
	return float64(n) * q
}

func ceilToQuantum(v, q float64) float64 {
	if v <= 0 {
		return 0
	}
	n := int(v / q)
	f := float64(n) * q
	if v > f+1e-9 {
		return f + q
	}
	return f
}
