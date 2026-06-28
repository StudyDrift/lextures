// Package scheduler implements the cron-like scheduled-job layer (plan 17.4). It
// evaluates a configuration-driven list of cron schedules on a tick loop and,
// guarded by a Postgres distributed lock, enqueues a jobs.queue row when a
// schedule is due so execution reuses the 17.3 retry/dead-letter machinery.
package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule is a parsed standard 5-field cron expression
// (minute hour day-of-month month day-of-week) evaluated in UTC. It supports
// '*', explicit values, comma lists, 'a-b' ranges and '*/n' / 'a-b/n' steps —
// enough for every built-in schedule (plan 17.4 NFR: in-process evaluation).
type Schedule struct {
	expr    string
	minute  uint64 // bitset 0..59
	hour    uint64 // bitset 0..23
	dom     uint64 // bitset 1..31
	month   uint64 // bitset 1..12
	dow     uint64 // bitset 0..6 (Sunday = 0)
	domStar bool   // day-of-month was '*'
	dowStar bool   // day-of-week was '*'
}

// Expr returns the original cron expression.
func (s Schedule) Expr() string { return s.expr }

type fieldSpec struct {
	min, max int
}

var (
	minuteField = fieldSpec{0, 59}
	hourField   = fieldSpec{0, 23}
	domField    = fieldSpec{1, 31}
	monthField  = fieldSpec{1, 12}
	dowField    = fieldSpec{0, 6}
)

// Parse compiles a 5-field cron expression. It returns an error for any field it
// cannot represent so a typo in config.go fails loudly at startup rather than
// silently never firing.
func Parse(expr string) (Schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return Schedule{}, fmt.Errorf("scheduler: cron %q must have 5 fields, got %d", expr, len(fields))
	}
	minute, _, err := parseField(fields[0], minuteField)
	if err != nil {
		return Schedule{}, fmt.Errorf("scheduler: cron %q minute: %w", expr, err)
	}
	hour, _, err := parseField(fields[1], hourField)
	if err != nil {
		return Schedule{}, fmt.Errorf("scheduler: cron %q hour: %w", expr, err)
	}
	dom, domStar, err := parseField(fields[2], domField)
	if err != nil {
		return Schedule{}, fmt.Errorf("scheduler: cron %q day-of-month: %w", expr, err)
	}
	month, _, err := parseField(fields[3], monthField)
	if err != nil {
		return Schedule{}, fmt.Errorf("scheduler: cron %q month: %w", expr, err)
	}
	dow, dowStar, err := parseField(fields[4], dowField)
	if err != nil {
		return Schedule{}, fmt.Errorf("scheduler: cron %q day-of-week: %w", expr, err)
	}
	return Schedule{
		expr:    expr,
		minute:  minute,
		hour:    hour,
		dom:     dom,
		month:   month,
		dow:     dow,
		domStar: domStar,
		dowStar: dowStar,
	}, nil
}

// MustParse is Parse for package-level schedule constants; it panics on error.
func MustParse(expr string) Schedule {
	s, err := Parse(expr)
	if err != nil {
		panic(err)
	}
	return s
}

// parseField turns one cron field into a bitset. The second return reports
// whether the field was a bare '*', which the day-of-month/day-of-week matching
// rule needs (a restricted dom OR dow both match, per cron convention).
func parseField(field string, spec fieldSpec) (uint64, bool, error) {
	if field == "*" {
		var bits uint64
		for v := spec.min; v <= spec.max; v++ {
			bits |= 1 << uint(v)
		}
		return bits, true, nil
	}
	var bits uint64
	for _, part := range strings.Split(field, ",") {
		b, err := parsePart(part, spec)
		if err != nil {
			return 0, false, err
		}
		bits |= b
	}
	return bits, false, nil
}

func parsePart(part string, spec fieldSpec) (uint64, error) {
	rangePart := part
	step := 1
	if slash := strings.IndexByte(part, '/'); slash >= 0 {
		rangePart = part[:slash]
		s, err := strconv.Atoi(part[slash+1:])
		if err != nil || s <= 0 {
			return 0, fmt.Errorf("invalid step %q", part)
		}
		step = s
	}

	lo, hi := spec.min, spec.max
	switch {
	case rangePart == "*":
		// keep full range
	case strings.ContainsRune(rangePart, '-'):
		bounds := strings.SplitN(rangePart, "-", 2)
		a, err1 := strconv.Atoi(bounds[0])
		b, err2 := strconv.Atoi(bounds[1])
		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("invalid range %q", part)
		}
		lo, hi = a, b
	default:
		v, err := strconv.Atoi(rangePart)
		if err != nil {
			return 0, fmt.Errorf("invalid value %q", part)
		}
		lo, hi = v, v
	}

	if lo < spec.min || hi > spec.max || lo > hi {
		return 0, fmt.Errorf("value out of range %q (allowed %d-%d)", part, spec.min, spec.max)
	}
	var bits uint64
	for v := lo; v <= hi; v += step {
		bits |= 1 << uint(v)
	}
	return bits, nil
}

func (s Schedule) matches(t time.Time) bool {
	if s.minute&(1<<uint(t.Minute())) == 0 {
		return false
	}
	if s.hour&(1<<uint(t.Hour())) == 0 {
		return false
	}
	if s.month&(1<<uint(t.Month())) == 0 {
		return false
	}
	domMatch := s.dom&(1<<uint(t.Day())) != 0
	dowMatch := s.dow&(1<<uint(t.Weekday())) != 0
	// Cron rule: when both day fields are restricted, a match in either fires;
	// when one is '*', the other governs.
	switch {
	case s.domStar && s.dowStar:
		return true
	case s.domStar:
		return dowMatch
	case s.dowStar:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}

// Next returns the first scheduled time strictly after t (minute resolution, UTC).
// It scans minute by minute, bounded so an impossible expression cannot loop
// forever.
func (s Schedule) Next(t time.Time) time.Time {
	t = t.UTC().Truncate(time.Minute).Add(time.Minute)
	// Up to ~4 years of minutes covers every representable schedule (e.g. Feb 29).
	const maxMinutes = 4 * 366 * 24 * 60
	for i := 0; i < maxMinutes; i++ {
		if s.matches(t) {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}
}

// IsDue reports whether a schedule whose last trigger was lastRun should fire at
// or before now. When lastRun is zero (no history) the schedule is anchored at
// the scheduler start time so a fresh deploy does not back-fire. A schedule
// missed while the app was down fires on the next tick because Next(lastRun)
// lands in the past (plan 17.4 NFR reliability, AC-5).
func (s Schedule) IsDue(lastRun, now time.Time) bool {
	next := s.Next(lastRun)
	if next.IsZero() {
		return false
	}
	return !next.After(now)
}
