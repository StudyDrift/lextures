package analytics

// ToolDisengagePct computes the 0–100 percent of learner tool instances with no engagement.
// Used as an optional formative signal for atriskscoring (CT.7 FR-12; weight defaults to 0).
func ToolDisengagePct(rows []SummaryRow) float32 {
	learners := 0
	disengaged := 0
	for _, r := range rows {
		if !IsLearnerRole(r.Role) {
			continue
		}
		learners++
		if !r.Engaged {
			disengaged++
		}
	}
	if learners == 0 {
		return 0
	}
	return float32(disengaged) / float32(learners) * 100
}

// ApplyToolDisengageSignal fills atrisk SignalInputs.ToolDisengagePct from summaries.
// Kept as a named entry point so the formative plumbing stays wired (open Q3: weight 0).
func ApplyToolDisengageSignal(rows []SummaryRow, set func(pct float32)) {
	if set == nil {
		return
	}
	set(ToolDisengagePct(rows))
}
