package gradingdrops

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FinalCol is one gradable gradebook column for the course-final calculation.
type FinalCol struct {
	ID               uuid.UUID
	MaxPoints        float64 // <= 0 means the column is not gradable and is skipped
	GroupID          *uuid.UUID
	NeverDrop        bool
	ReplaceWithFinal bool
	DueAt            *time.Time
}

// FinalGroup is one assignment group's weight and drop policy.
type FinalGroup struct {
	ID            uuid.UUID
	WeightPercent float64
	Policy        GroupDropPolicy
}

type finalLine struct {
	id        uuid.UUID
	max       float64
	earned    float64
	neverDrop bool
	isFinal   bool
}

// parseEarned parses a gradebook cell ("", "7", "1,234.5") the same way the web gradebook does.
func parseEarned(raw string) float64 {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(t, ",", ""), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

// groupEffective ports the web `groupEffectiveEarnedAndMax` (plan 3.9): effective earned/max for one
// assignment group after drop-lowest / drop-highest / replace-lowest-with-final.
func groupEffective(p GroupDropPolicy, lines []finalLine) (effEarned, effMax float64) {
	type row struct {
		id      uuid.UUID
		max     float64
		earned  float64
		pct     float64
		canDrop bool
		isFinal bool
	}
	rows := make([]row, 0, len(lines))
	for _, l := range lines {
		max := 0.0
		if l.max > 0 && !math.IsInf(l.max, 0) && !math.IsNaN(l.max) {
			max = l.max
		}
		if max <= 0 {
			continue
		}
		earned := math.Max(0, l.earned)
		pct := earned / max
		if math.IsNaN(pct) || math.IsInf(pct, 0) {
			pct = 0
		}
		rows = append(rows, row{id: l.id, max: max, earned: earned, pct: pct, canDrop: !l.neverDrop && !l.isFinal, isFinal: l.isFinal})
	}
	if len(rows) == 0 {
		return 0, 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].pct != rows[j].pct {
			return rows[i].pct < rows[j].pct
		}
		return rows[i].id.String() < rows[j].id.String()
	})
	var work []row
	for _, r := range rows {
		if r.canDrop {
			work = append(work, r)
		}
	}
	dropped := make(map[uuid.UUID]bool)
	for i := 0; i < p.DropLowest && len(work) > 0; i++ {
		dropped[work[0].id] = true
		work = work[1:]
	}
	for i := 0; i < p.DropHighest && len(work) > 0; i++ {
		dropped[work[len(work)-1].id] = true
		work = work[:len(work)-1]
	}
	for _, r := range rows {
		if dropped[r.id] {
			continue
		}
		effMax += r.max
		effEarned += r.earned
	}
	if p.ReplaceLowestWithFinal {
		var fin *row
		for i := range rows {
			if rows[i].isFinal && !dropped[rows[i].id] {
				fin = &rows[i]
				break
			}
		}
		if fin != nil && fin.pct > 0 {
			var target *row
			for i := range rows {
				r := &rows[i]
				if r.isFinal || dropped[r.id] {
					continue
				}
				if target == nil || r.pct < target.pct || (r.pct == target.pct && r.id.String() < target.id.String()) {
					target = r
				}
			}
			if target != nil && fin.pct > target.pct+1e-12 {
				effEarned -= target.earned
				effEarned += target.max * fin.pct
			}
		}
	}
	return effEarned, effMax
}

const ungroupedBucket = "__ungrouped__"

// CourseFinalPercent is the Go port of the web `computeCourseFinalPercent` that backs the gradebook
// FINAL column and the My grades course total. It returns a 0–100 percentage, or nil when nothing
// counts yet.
//
// gradesByItem holds raw gradebook cell strings; an item counts when it has a grade entered or is past
// its due date (missing work counts as zero). Excused items never count. Items in an assignment
// group use that group's drop policy and weight; ungrouped items share the unallocated remainder.
func CourseFinalPercent(
	cols []FinalCol,
	gradesByItem map[uuid.UUID]string,
	groups []FinalGroup,
	excused map[uuid.UUID]bool,
	now time.Time,
) *float64 {
	settings := make(map[uuid.UUID]FinalGroup, len(groups))
	for _, g := range groups {
		settings[g.ID] = g
	}
	maxByBucket := map[string]float64{}
	earnedByBucket := map[string]float64{}
	byGroup := map[uuid.UUID][]finalLine{}

	for _, c := range cols {
		if c.MaxPoints <= 0 || math.IsNaN(c.MaxPoints) || math.IsInf(c.MaxPoints, 0) {
			continue
		}
		if excused[c.ID] {
			continue
		}
		raw, hasEntry := gradesByItem[c.ID]
		hasGrade := hasEntry && strings.TrimSpace(raw) != ""
		pastDue := c.DueAt != nil && c.DueAt.Before(now)
		if !hasGrade && !pastDue {
			continue
		}
		earned := parseEarned(raw)
		if c.GroupID != nil {
			if _, ok := settings[*c.GroupID]; ok {
				byGroup[*c.GroupID] = append(byGroup[*c.GroupID], finalLine{
					id: c.ID, max: c.MaxPoints, earned: earned, neverDrop: c.NeverDrop, isFinal: c.ReplaceWithFinal,
				})
				continue
			}
		}
		maxByBucket[ungroupedBucket] += c.MaxPoints
		earnedByBucket[ungroupedBucket] += earned
	}

	for gid, lines := range byGroup {
		e, m := groupEffective(settings[gid].Policy, lines)
		maxByBucket[gid.String()] += m
		earnedByBucket[gid.String()] += e
	}

	var totalMax float64
	for _, v := range maxByBucket {
		totalMax += v
	}
	if totalMax <= 0 {
		return nil
	}
	has := map[string]bool{}
	for b, mx := range maxByBucket {
		if mx > 0 {
			has[b] = true
		}
	}
	if len(has) == 0 {
		return nil
	}

	weightOf := func(g FinalGroup) float64 {
		if g.WeightPercent > 0 && !math.IsInf(g.WeightPercent, 0) && !math.IsNaN(g.WeightPercent) {
			return g.WeightPercent
		}
		return 0
	}
	var configuredSum, lost float64
	for _, g := range groups {
		w := weightOf(g)
		configuredSum += w
		if w > 0 && !has[g.ID.String()] {
			lost += w
		}
	}
	remainder := math.Max(0, 100-configuredSum)
	maxUngrouped := maxByBucket[ungroupedBucket]

	rawWeight := map[string]float64{}
	for _, g := range groups {
		if !has[g.ID.String()] {
			continue
		}
		if w := weightOf(g); w > 0 {
			rawWeight[g.ID.String()] = w
		}
	}
	if has[ungroupedBucket] {
		wU := remainder + lost
		if wU <= 0 && maxUngrouped > 0 && totalMax > 0 {
			wU = maxUngrouped / totalMax * 100
		}
		rawWeight[ungroupedBucket] += wU
	}
	var weightSum float64
	for _, w := range rawWeight {
		weightSum += w
	}
	if weightSum <= 0 {
		var earnedTotal float64
		for _, v := range earnedByBucket {
			earnedTotal += v
		}
		p := earnedTotal / totalMax * 100
		return &p
	}
	var acc float64
	for b, rw := range rawWeight {
		if rw <= 0 {
			continue
		}
		mx := maxByBucket[b]
		ratio := 0.0
		if mx > 0 {
			ratio = earnedByBucket[b] / mx
		}
		acc += ratio * (rw / weightSum)
	}
	p := acc * 100
	return &p
}
