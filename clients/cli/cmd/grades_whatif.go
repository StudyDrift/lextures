package cmd

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ungroupedGroupID = "__ungrouped__"

type assignmentGroupWeight struct {
	ID                     string  `json:"id"`
	WeightPercent          float64 `json:"weightPercent"`
	DropLowest             int     `json:"dropLowest"`
	DropHighest            int     `json:"dropHighest"`
	ReplaceLowestWithFinal bool    `json:"replaceLowestWithFinal"`
}

type groupDropPolicy struct {
	dropLowest             int
	dropHighest            int
	replaceLowestWithFinal bool
}

type scoredLine struct {
	id      string
	max     float64
	earned  float64
	pct     float64
	canDrop bool
	isFinal bool
}

func isFiniteFloat(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func parseEarnedPoints(raw string) float64 {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(t, ",", ""), 64)
	if err != nil || !isFiniteFloat(n) {
		return 0
	}
	return n
}

func mergeGradesForWhatIf(
	actual map[string]string,
	overrides map[string]string,
	held map[string]bool,
) map[string]string {
	merged := make(map[string]string, len(actual)+len(overrides))
	for k, v := range actual {
		merged[k] = v
	}
	for id := range held {
		delete(merged, id)
	}
	for id, val := range overrides {
		t := strings.TrimSpace(val)
		if t == "" {
			delete(merged, id)
		} else {
			merged[id] = t
		}
	}
	return merged
}

func groupEffectiveEarnedAndMax(policy groupDropPolicy, lines []struct {
	itemID           string
	max              float64
	earned           float64
	neverDrop        bool
	replaceWithFinal bool
}) (effectiveEarned, effectiveMax float64, dropped map[string]bool) {
	dropped = make(map[string]bool)
	if len(lines) == 0 {
		return 0, 0, dropped
	}
	rows := make([]scoredLine, 0, len(lines))
	for _, l := range lines {
		max := 0.0
		if l.max > 0 && isFiniteFloat(l.max) {
			max = l.max
		}
		earned := math.Max(0, l.earned)
		pct := 0.0
		if max > 0 {
			pct = earned / max
		}
		if !isFiniteFloat(pct) {
			pct = 0
		}
		isFinal := l.replaceWithFinal
		canDrop := !l.neverDrop && !isFinal
		if max > 0 {
			rows = append(rows, scoredLine{
				id: l.itemID, max: max, earned: earned, pct: pct, canDrop: canDrop, isFinal: isFinal,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].pct != rows[j].pct {
			return rows[i].pct < rows[j].pct
		}
		return rows[i].id < rows[j].id
	})
	work := make([]scoredLine, 0, len(rows))
	for _, r := range rows {
		if r.canDrop {
			work = append(work, r)
		}
	}
	for i := 0; i < policy.dropLowest; i++ {
		if len(work) == 0 {
			break
		}
		dropped[work[0].id] = true
		work = work[1:]
	}
	for i := 0; i < policy.dropHighest; i++ {
		if len(work) == 0 {
			break
		}
		dropped[work[len(work)-1].id] = true
		work = work[:len(work)-1]
	}
	for _, r := range rows {
		if dropped[r.id] {
			continue
		}
		effectiveMax += r.max
		effectiveEarned += r.earned
	}
	if policy.replaceLowestWithFinal {
		var finalRow *scoredLine
		for i := range rows {
			r := rows[i]
			if r.isFinal && !dropped[r.id] && r.pct > 0 {
				finalRow = &rows[i]
				break
			}
		}
		if finalRow != nil {
			var lowest *scoredLine
			for i := range rows {
				r := rows[i]
				if r.isFinal || dropped[r.id] {
					continue
				}
				if lowest == nil || r.pct < lowest.pct || (r.pct == lowest.pct && r.id < lowest.id) {
					lowest = &rows[i]
				}
			}
			if lowest != nil && finalRow.pct > lowest.pct+1e-12 {
				effectiveEarned -= lowest.earned
				effectiveEarned += lowest.max * finalRow.pct
			}
		}
	}
	return effectiveEarned, effectiveMax, dropped
}

type whatIfComputeOptions struct {
	mode            string
	whatIfOverrides map[string]string
	heldItemIDs     map[string]bool
	excused         map[string]bool
	now             time.Time
}

func computeCourseFinalPercent(
	columns []gradeColumn,
	gradesByItemID map[string]string,
	assignmentGroups []assignmentGroupWeight,
	opts whatIfComputeOptions,
) *float64 {
	mode := opts.mode
	if mode == "" {
		mode = "actual"
	}
	merged := gradesByItemID
	if mode == "whatIf" {
		merged = mergeGradesForWhatIf(gradesByItemID, opts.whatIfOverrides, opts.heldItemIDs)
	}

	settingsIDs := make(map[string]struct{}, len(assignmentGroups))
	polByG := make(map[string]groupDropPolicy, len(assignmentGroups))
	for _, g := range assignmentGroups {
		settingsIDs[g.ID] = struct{}{}
		dropLow := g.DropLowest
		if dropLow < 0 {
			dropLow = 0
		}
		dropHigh := g.DropHighest
		if dropHigh < 0 {
			dropHigh = 0
		}
		polByG[g.ID] = groupDropPolicy{
			dropLowest:             dropLow,
			dropHighest:            dropHigh,
			replaceLowestWithFinal: g.ReplaceLowestWithFinal,
		}
	}

	maxByBucket := make(map[string]float64)
	earnedByBucket := make(map[string]float64)
	byGroup := make(map[string][]struct {
		itemID           string
		max              float64
		earned           float64
		neverDrop        bool
		replaceWithFinal bool
	})

	nowMs := opts.now.UnixMilli()

	for _, col := range columns {
		if col.MaxPoints == nil || *col.MaxPoints <= 0 {
			continue
		}
		if opts.excused != nil && opts.excused[col.ID] {
			continue
		}
		hasOverride := mode == "whatIf" && strings.TrimSpace(opts.whatIfOverrides[col.ID]) != ""
		gradeStr := merged[col.ID]
		if !shouldIncludeColumn(col, gradeStr, hasOverride, mode, nowMs) {
			continue
		}
		earned := parseEarnedPoints(gradeStr)
		max := float64(*col.MaxPoints)
		gid := ""
		if col.AssignmentGroupID != nil {
			gid = strings.TrimSpace(*col.AssignmentGroupID)
		}
		bucket := ungroupedGroupID
		if gid != "" {
			if _, ok := settingsIDs[gid]; ok {
				bucket = gid
			}
		}
		line := struct {
			itemID           string
			max              float64
			earned           float64
			neverDrop        bool
			replaceWithFinal bool
		}{
			itemID: col.ID, max: max, earned: earned,
			neverDrop: col.NeverDrop, replaceWithFinal: col.ReplaceWithFinal,
		}
		if bucket == ungroupedGroupID {
			maxByBucket[bucket] += max
			earnedByBucket[bucket] += earned
		} else {
			byGroup[bucket] = append(byGroup[bucket], line)
		}
	}

	for gid, lines := range byGroup {
		p := polByG[gid]
		effectiveEarned, effectiveMax, _ := groupEffectiveEarnedAndMax(p, lines)
		maxByBucket[gid] += effectiveMax
		earnedByBucket[gid] += effectiveEarned
	}

	var totalMaxPoints float64
	bucketsWithColumns := make(map[string]struct{})
	for bucket, mx := range maxByBucket {
		totalMaxPoints += mx
		if mx > 0 {
			bucketsWithColumns[bucket] = struct{}{}
		}
	}
	if totalMaxPoints <= 0 || len(bucketsWithColumns) == 0 {
		return nil
	}

	var configuredSum float64
	for _, g := range assignmentGroups {
		if isFiniteFloat(g.WeightPercent) && g.WeightPercent > 0 {
			configuredSum += g.WeightPercent
		}
	}
	remainder := math.Max(0, 100-configuredSum)

	var lostConfiguredWeight float64
	for _, g := range assignmentGroups {
		if !isFiniteFloat(g.WeightPercent) || g.WeightPercent <= 0 {
			continue
		}
		if _, ok := bucketsWithColumns[g.ID]; !ok {
			lostConfiguredWeight += g.WeightPercent
		}
	}

	maxUngrouped := maxByBucket[ungroupedGroupID]
	rawWeight := make(map[string]float64)
	for _, g := range assignmentGroups {
		if _, ok := bucketsWithColumns[g.ID]; !ok {
			continue
		}
		if isFiniteFloat(g.WeightPercent) && g.WeightPercent > 0 {
			rawWeight[g.ID] = g.WeightPercent
		}
	}
	if _, ok := bucketsWithColumns[ungroupedGroupID]; ok {
		wU := remainder + lostConfiguredWeight
		if wU <= 0 && maxUngrouped > 0 && totalMaxPoints > 0 {
			wU = (maxUngrouped / totalMaxPoints) * 100
		}
		rawWeight[ungroupedGroupID] += wU
	}

	var weightSum float64
	for _, w := range rawWeight {
		weightSum += w
	}
	if weightSum <= 0 {
		var earnedTotal float64
		for _, e := range earnedByBucket {
			earnedTotal += e
		}
		pct := (earnedTotal / totalMaxPoints) * 100
		return &pct
	}

	var acc float64
	for bucket, rw := range rawWeight {
		if rw <= 0 {
			continue
		}
		maxB := maxByBucket[bucket]
		earnedB := earnedByBucket[bucket]
		ratio := 0.0
		if maxB > 0 {
			ratio = earnedB / maxB
		}
		acc += ratio * (rw / weightSum)
	}
	pct := acc * 100
	return &pct
}

func shouldIncludeColumn(col gradeColumn, gradeStr string, hasOverride bool, mode string, nowMs int64) bool {
	if mode == "whatIf" && hasOverride {
		return true
	}
	hasGrade := strings.TrimSpace(gradeStr) != ""
	if hasGrade {
		return true
	}
	if col.DueAt != nil && strings.TrimSpace(*col.DueAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*col.DueAt)); err == nil {
			return t.UnixMilli() < nowMs
		}
	}
	return false
}

func computeWhatIfFinalPercent(
	columns []gradeColumn,
	actualGrades map[string]string,
	assignmentGroups []assignmentGroupWeight,
	excused map[string]bool,
	overrides map[string]string,
	held map[string]bool,
	now time.Time,
) *float64 {
	return computeCourseFinalPercent(columns, actualGrades, assignmentGroups, whatIfComputeOptions{
		mode:            "whatIf",
		whatIfOverrides: overrides,
		heldItemIDs:     held,
		excused:         excused,
		now:             now,
	})
}