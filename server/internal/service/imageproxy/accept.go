package imageproxy

import (
	"strconv"
	"strings"
)

// FormatFromAccept prefers WebP when the client accepts it, otherwise JPEG.
// A missing Accept header, */*, and image/webp;q=0 stay on JPEG.
func FormatFromAccept(header string) string {
	q, ok := acceptQuality(header, "image/webp")
	if ok && q > 0 {
		return FormatWebP
	}
	return FormatJPEG
}

func acceptQuality(header, mime string) (float64, bool) {
	found := false
	best := 0.0
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := part
		q := 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			name = strings.TrimSpace(part[:i])
			for _, param := range strings.Split(part[i+1:], ";") {
				param = strings.TrimSpace(param)
				if !strings.HasPrefix(strings.ToLower(param), "q=") {
					continue
				}
				if v, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64); err == nil {
					q = v
				}
			}
		}
		if strings.EqualFold(name, mime) {
			found = true
			if q > best {
				best = q
			}
		}
	}
	return best, found
}
