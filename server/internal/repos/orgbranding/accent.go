// Package orgbranding — UX.1 brand accent OKLCH validation and ramp derivation.
package orgbranding

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Oklch is a parsed OKLCH colour (L 0–1, C ≥ 0, H 0–360).
type Oklch struct {
	L, C, H float64
}

var oklchRe = regexp.MustCompile(`(?i)^oklch\(\s*([0-9.]+%?)\s+([0-9.]+)\s+([0-9.]+)\s*(?:/\s*[0-9.]+%?)?\s*\)$`)

// ValidateOklch parses and re-serialises an OKLCH string; rejects CSS injection.
func ValidateOklch(s string) (string, Oklch, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", Oklch{}, fmt.Errorf("empty oklch")
	}
	m := oklchRe.FindStringSubmatch(t)
	if m == nil {
		return "", Oklch{}, fmt.Errorf("expected oklch(L C H)")
	}
	lStr := m[1]
	l, err := strconv.ParseFloat(strings.TrimSuffix(lStr, "%"), 64)
	if err != nil {
		return "", Oklch{}, fmt.Errorf("invalid L")
	}
	if strings.Contains(lStr, "%") {
		l = l / 100
	}
	if l < 0 || l > 1.001 {
		return "", Oklch{}, fmt.Errorf("L out of range")
	}
	c, err := strconv.ParseFloat(m[2], 64)
	if err != nil || c < 0 || c > 0.5 {
		return "", Oklch{}, fmt.Errorf("C out of range")
	}
	h, err := strconv.ParseFloat(m[3], 64)
	if err != nil || h < 0 || h >= 360 {
		return "", Oklch{}, fmt.Errorf("H out of range")
	}
	l = math.Min(1, math.Max(0, l))
	norm := fmt.Sprintf("oklch(%.4g %.4g %.4g)", l, c, h)
	return norm, Oklch{L: l, C: c, H: h}, nil
}

// AccentStep is a ramp key.
type AccentStep string

var accentSteps = []AccentStep{"50", "100", "200", "300", "400", "500", "600", "700", "800", "900", "950"}

var rampL = map[AccentStep]float64{
	"50": 0.96, "100": 0.93, "200": 0.87, "300": 0.79, "400": 0.67,
	"500": 0.59, "600": 0.51, "700": 0.46, "800": 0.4, "900": 0.36, "950": 0.26,
}

// DeriveAccentRamp builds a full accent scale from a seed OKLCH.
func DeriveAccentRamp(seed Oklch) map[string]string {
	hue := seed.H
	chroma := math.Min(0.22, math.Max(0.08, seed.C))
	out := make(map[string]string, len(accentSteps))
	for _, step := range accentSteps {
		l := rampL[step]
		cScale := 0.85
		if l > 0.9 {
			cScale = 0.15
		} else if l < 0.3 {
			cScale = 0.45
		} else {
			cScale = 0.85 + (0.5-math.Abs(l-0.55))*0.4
		}
		c := chroma * cScale
		out[string(step)] = fmt.Sprintf("oklch(%.4g %.4g %.4g)", l, c, hue)
	}
	return out
}

// FailingPair describes a contrast failure for API errors.
type FailingPair struct {
	FG       string  `json:"fg"`
	BG       string  `json:"bg"`
	Ratio    float64 `json:"ratio"`
	Required float64 `json:"required"`
}

// ValidateAccentRampAA checks onAccent (white) on solid (600) ≥ 4.5:1 using approximate OKLCH→hex.
func ValidateAccentRampAA(seed Oklch) (ramp map[string]string, failing []FailingPair, suggestion string) {
	ramp = DeriveAccentRamp(seed)
	// Convert solid (600) to approximate sRGB and compare white
	solid := Oklch{L: rampL["600"], C: math.Min(0.22, math.Max(0.08, seed.C)) * 0.9, H: seed.H}
	hex := oklchToHexApprox(solid)
	ratio, err := contrastHex(hex, "#FFFFFF")
	if err != nil || ratio < 4.5 {
		if err != nil {
			ratio = 0
		}
		failing = append(failing, FailingPair{
			FG: "fg-onAccent", BG: "accent-600", Ratio: math.Round(ratio*100) / 100, Required: 4.5,
		})
		// Suggest darker L
		for l := 0.45; l >= 0.25; l -= 0.02 {
			try := Oklch{L: l, C: seed.C, H: seed.H}
			th := oklchToHexApprox(try)
			r, e := contrastHex(th, "#FFFFFF")
			if e == nil && r >= 4.5 {
				suggestion = fmt.Sprintf("oklch(%.4g %.4g %.4g)", l, seed.C, seed.H)
				break
			}
		}
	}
	return ramp, failing, suggestion
}

func contrastHex(a, b string) (float64, error) {
	la, err := RelativeLuminanceWCAG(a)
	if err != nil {
		return 0, err
	}
	lb, err := RelativeLuminanceWCAG(b)
	if err != nil {
		return 0, err
	}
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05), nil
}

// Minimal OKLCH → sRGB hex (clamped) for contrast checks.
func oklchToHexApprox(c Oklch) string {
	hr := c.H * math.Pi / 180
	a := c.C * math.Cos(hr)
	b := c.C * math.Sin(hr)
	l_ := c.L + 0.3963377774*a + 0.2158037573*b
	m_ := c.L - 0.1055613458*a - 0.0638541728*b
	s_ := c.L - 0.0894841775*a - 1.291485548*b
	l3, m3, s3 := l_*l_*l_, m_*m_*m_, s_*s_*s_
	lr := +4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3
	lg := -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3
	lb := -0.0041960863*l3 - 0.7034186147*m3 + 1.707614701*s3
	toByte := func(v float64) int {
		x := math.Min(1, math.Max(0, v))
		var g float64
		if x <= 0.0031308 {
			g = 12.92 * x
		} else {
			g = 1.055*math.Pow(x, 1/2.4) - 0.055
		}
		return int(math.Round(g * 255))
	}
	return fmt.Sprintf("#%02X%02X%02X", toByte(lr), toByte(lg), toByte(lb))
}
