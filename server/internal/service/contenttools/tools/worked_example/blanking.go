package worked_example

// ResolveBlanked returns which step ids are blanked under the configured policy.
// Progressive blanking is deterministic per enrollment seed: later steps are
// blanked at an increasing rate (roughly first third shown, middle half blanked,
// last two-thirds blanked — exact curve below).
func ResolveBlanked(cfg Config, enrollmentSeed uint64) map[string]bool {
	out := map[string]bool{}
	n := len(cfg.Steps)
	if n == 0 {
		return out
	}
	switch cfg.BlankPolicy {
	case BlankAll:
		for _, s := range cfg.Steps {
			if s.Blank != nil {
				out[s.ID] = true
			}
		}
		return out
	case BlankProgressive:
		for i, s := range cfg.Steps {
			if s.Blank == nil {
				continue
			}
			// Fraction blanked grows with position: 0 at start → 1 at end.
			// Deterministic threshold jitter per enrollment keeps fairness.
			frac := float64(i) / float64(maxInt(n-1, 1))
			threshold := 0.35 + 0.55*frac // earlier steps likelier shown
			jitter := float64((enrollmentSeed+uint64(i)*2654435761)%1000) / 1000.0
			// Blank when frac exceeds threshold adjusted by small jitter.
			if frac+jitter*0.1 >= threshold-0.15 {
				out[s.ID] = true
			}
		}
		// Ensure at least one blanked step when any blanks exist.
		if len(out) == 0 {
			for i := n - 1; i >= 0; i-- {
				if cfg.Steps[i].Blank != nil {
					out[cfg.Steps[i].ID] = true
					break
				}
			}
		}
		return out
	default: // author
		for _, s := range cfg.Steps {
			if s.Blank != nil {
				out[s.ID] = true
			}
		}
		return out
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
