package studyreminders

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	repostudyreminders "github.com/lextures/lextures/server/internal/repos/studyreminders"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// APIConfig is the GET/PATCH reminder-config payload.
type APIConfig struct {
	DailyGoalMinutes    int      `json:"dailyGoalMinutes"`
	ReminderTime        string   `json:"reminderTime"`
	ReminderChannels    []string `json:"reminderChannels"`
	WeeklySummary       bool     `json:"weeklySummary"`
	Enabled             bool     `json:"enabled"`
	PausedUntil         *string  `json:"pausedUntil,omitempty"`
	MinutesStudiedToday int      `json:"minutesStudiedToday"`
	GoalMetToday        bool     `json:"goalMetToday"`
	StreakAtRiskBanner  bool     `json:"streakAtRiskBanner"`
}

// Service orchestrates reminder scheduling and delivery (plan 15.10).
type Service struct {
	Pool   *pgxpool.Pool
	Config config.Config
}

func defaultConfig() APIConfig {
	return APIConfig{
		DailyGoalMinutes: 20,
		ReminderTime:     "19:00",
		ReminderChannels: []string{"email"},
		WeeklySummary:    true,
		Enabled:          false,
	}
}

// LoadAPIConfig returns preferences with today's progress for the dashboard.
func (s *Service) LoadAPIConfig(ctx context.Context, userID uuid.UUID, now time.Time, timezone *string) (APIConfig, error) {
	out := defaultConfig()
	row, err := repostudyreminders.Get(ctx, s.Pool, userID)
	if err != nil {
		return out, err
	}
	if row != nil {
		out.DailyGoalMinutes = row.DailyGoalMinutes
		out.ReminderTime = row.ReminderTime.Format("15:04")
		out.ReminderChannels = append([]string(nil), row.ReminderChannels...)
		out.WeeklySummary = row.WeeklySummary
		out.Enabled = row.Enabled
		if row.PausedUntil != nil {
			iso := row.PausedUntil.Format("2006-01-02")
			out.PausedUntil = &iso
		}
	}
	today := UserLocalDate(now, timezone)
	minutes, err := MinutesStudiedToday(ctx, s.Pool, userID, today, timezone)
	if err != nil {
		return out, err
	}
	out.MinutesStudiedToday = minutes
	out.GoalMetToday = minutes >= out.DailyGoalMinutes
	if s.Config.FFGamification {
		streak, last, err := loadStreak(ctx, s.Pool, userID)
		if err == nil {
			localNow := UserLocalNow(now, timezone)
			out.StreakAtRiskBanner = StreakAtRiskBanner(localNow, today, streak, last, timezone)
		}
	}
	return out, nil
}

// SaveAPIConfig updates reminder preferences.
func (s *Service) SaveAPIConfig(ctx context.Context, userID uuid.UUID, patch APIConfig) (APIConfig, error) {
	current, err := s.LoadAPIConfig(ctx, userID, time.Now().UTC(), nil)
	if err != nil {
		return APIConfig{}, err
	}
	if patch.DailyGoalMinutes > 0 {
		current.DailyGoalMinutes = patch.DailyGoalMinutes
	}
	if strings.TrimSpace(patch.ReminderTime) != "" {
		current.ReminderTime = patch.ReminderTime
	}
	if patch.ReminderChannels != nil {
		current.ReminderChannels = normalizeChannels(patch.ReminderChannels)
	}
	current.WeeklySummary = patch.WeeklySummary
	current.Enabled = patch.Enabled

	rt, err := time.Parse("15:04", current.ReminderTime)
	if err != nil {
		return APIConfig{}, fmt.Errorf("invalid reminder time")
	}
	if err := repostudyreminders.Upsert(ctx, s.Pool, userID, current.DailyGoalMinutes, rt, current.ReminderChannels, current.WeeklySummary, current.Enabled); err != nil {
		return APIConfig{}, err
	}
	tz, _ := loadTimezone(ctx, s.Pool, userID)
	return s.LoadAPIConfig(ctx, userID, time.Now().UTC(), tz)
}

// Pause pauses reminders for N days from the user's local today.
func (s *Service) Pause(ctx context.Context, userID uuid.UUID, days int, now time.Time, timezone *string) (APIConfig, error) {
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	today := UserLocalDate(now, timezone)
	until := today.AddDate(0, 0, days)
	if err := repostudyreminders.PauseUntil(ctx, s.Pool, userID, until); err != nil {
		return APIConfig{}, err
	}
	return s.LoadAPIConfig(ctx, userID, now, timezone)
}

// Disable turns off reminders (unsubscribe flow).
func (s *Service) Disable(ctx context.Context, userID uuid.UUID) error {
	return repostudyreminders.SetEnabled(ctx, s.Pool, userID, false)
}

// RunSweep processes due reminders for all enabled users.
func (s *Service) RunSweep(ctx context.Context, now time.Time) (int, error) {
	if !s.Config.FFStudyReminders || s.Pool == nil {
		return 0, nil
	}
	candidates, err := repostudyreminders.ListEnabledCandidates(ctx, s.Pool, 1000)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, c := range candidates {
		n, err := s.processCandidate(ctx, c, now)
		if err != nil {
			continue
		}
		sent += n
	}
	return sent, nil
}

func (s *Service) processCandidate(ctx context.Context, c repostudyreminders.Candidate, now time.Time) (int, error) {
	localNow := UserLocalNow(now, c.Timezone)
	localDate := UserLocalDate(now, c.Timezone)
	if c.PausedUntil != nil && !localDate.After(*c.PausedUntil) {
		return 0, nil
	}
	studied, err := HasStudiedToday(ctx, s.Pool, c.UserID, localDate, c.Timezone)
	if err != nil {
		return 0, err
	}
	streak, _, _ := loadStreak(ctx, s.Pool, c.UserID)
	sent := 0

	if ShouldSendStreakAtRisk(localNow, localDate, c.ReminderTime, c.Timezone, studied, streak) {
		n, err := s.deliver(ctx, c, localDate, repostudyreminders.ReminderStreakAtRisk, studied, streak)
		if err == nil {
			sent += n
		}
	}
	if ShouldSendDaily(localNow, localDate, c.ReminderTime, c.Timezone, studied) {
		n, err := s.deliver(ctx, c, localDate, repostudyreminders.ReminderDaily, studied, streak)
		if err == nil {
			sent += n
		}
	}
	if ShouldSendWeeklySummary(localNow, localDate, c.ReminderTime, c.Timezone, c.WeeklySummary) {
		n, err := s.deliverWeekly(ctx, c, localDate, streak)
		if err == nil {
			sent += n
		}
	}
	return sent, nil
}

func (s *Service) deliver(ctx context.Context, c repostudyreminders.Candidate, localDate time.Time, reminderType string, studied bool, streak int) (int, error) {
	if studied && reminderType == repostudyreminders.ReminderDaily {
		return 0, nil
	}
	sent := 0
	for _, ch := range c.ReminderChannels {
		already, err := repostudyreminders.WasSentToday(ctx, s.Pool, c.UserID, localDate, reminderType, ch)
		if err != nil || already {
			continue
		}
		key := IdempotencyKey(c.UserID, localDate, reminderType, ch)
		if err := s.sendChannel(ctx, c, localDate, reminderType, ch, streak); err != nil {
			continue
		}
		_ = repostudyreminders.LogSend(ctx, s.Pool, c.UserID, localDate, reminderType, ch, key)
		RecordReminderSent(ch)
		sent++
	}
	return sent, nil
}

func (s *Service) deliverWeekly(ctx context.Context, c repostudyreminders.Candidate, localDate time.Time, streak int) (int, error) {
	ch := "email"
	already, err := repostudyreminders.WasSentToday(ctx, s.Pool, c.UserID, localDate, repostudyreminders.ReminderWeeklySummary, ch)
	if err != nil || already {
		return 0, err
	}
	summary, err := loadWeeklySummary(ctx, s.Pool, c.UserID, localDate, c.Timezone)
	if err != nil {
		return 0, err
	}
	summary.Streak = streak
	if err := s.sendWeeklyEmail(ctx, c.UserID, summary); err != nil {
		return 0, err
	}
	key := IdempotencyKey(c.UserID, localDate, repostudyreminders.ReminderWeeklySummary, ch)
	_ = repostudyreminders.LogSend(ctx, s.Pool, c.UserID, localDate, repostudyreminders.ReminderWeeklySummary, ch, key)
	RecordReminderSent(ch)
	return 1, nil
}

func (s *Service) sendChannel(ctx context.Context, c repostudyreminders.Candidate, localDate time.Time, reminderType, channel string, streak int) error {
	origin := strings.TrimRight(strings.TrimSpace(s.Config.PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	link := origin + "/dashboard"
	switch channel {
	case "email":
		ns := &notifications.Service{Pool: s.Pool, Config: s.Config}
		vars := map[string]string{
			"link":         link,
			"dailyGoal":    fmt.Sprintf("%d", c.DailyGoalMinutes),
			"streak":       fmt.Sprintf("%d", streak),
		}
		eventType := notificationevents.StudyReminderDaily
		template := "study_reminder_daily"
		if reminderType == repostudyreminders.ReminderStreakAtRisk {
			eventType = notificationevents.StudyReminderStreakAtRisk
			template = "study_reminder_streak_at_risk"
			vars["subject"] = fmt.Sprintf("Your %d-day streak is at risk", streak)
		} else {
			vars["subject"] = "Time for your daily study session"
		}
		return ns.EnqueueEmail(ctx, c.UserID, eventType, template, vars, nil)
	case "push":
		if !s.Config.PushNotificationsEnabled {
			return nil
		}
		ps := &notifications.PushService{Pool: s.Pool, Config: s.Config}
		title := "Study reminder"
		body := fmt.Sprintf("You haven't studied yet today. Your goal is %d minutes.", c.DailyGoalMinutes)
		eventType := notificationevents.StudyReminderDaily
		if reminderType == repostudyreminders.ReminderStreakAtRisk {
			eventType = notificationevents.StudyReminderStreakAtRisk
			title = "Streak at risk"
			body = fmt.Sprintf("Complete a lesson now to keep your %d-day streak.", streak)
		}
		return ps.Enqueue(ctx, c.UserID, eventType, title, body, link)
	default:
		return nil
	}
}

type weeklySummary struct {
	Streak          int
	XPEarned        int
	CoursesHTML     string
	CoursesText     string
}

func (s *Service) sendWeeklyEmail(ctx context.Context, userID uuid.UUID, summary weeklySummary) error {
	ns := &notifications.Service{Pool: s.Pool, Config: s.Config}
	origin := strings.TrimRight(strings.TrimSpace(s.Config.PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return ns.EnqueueEmail(ctx, userID, notificationevents.StudyReminderWeeklySummary, "study_reminder_weekly_summary", map[string]string{
		"subject":     "Your weekly learning summary",
		"streak":      fmt.Sprintf("%d", summary.Streak),
		"xpEarned":    fmt.Sprintf("%d", summary.XPEarned),
		"coursesHtml": summary.CoursesHTML,
		"coursesText": summary.CoursesText,
		"link":        origin + "/dashboard",
	}, nil)
}

func normalizeChannels(channels []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, ch := range channels {
		ch = strings.TrimSpace(strings.ToLower(ch))
		if ch != "email" && ch != "push" {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		out = append(out, ch)
	}
	if len(out) == 0 {
		return []string{"email"}
	}
	return out
}

func loadTimezone(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*string, error) {
	var tz *string
	err := pool.QueryRow(ctx, `SELECT timezone FROM "user".users WHERE id = $1`, userID).Scan(&tz)
	return tz, err
}

func loadStreak(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (current int, last *time.Time, err error) {
	var lastAct *time.Time
	err = pool.QueryRow(ctx, `
SELECT current_streak, last_activity_date FROM gamification.user_gamification WHERE user_id = $1
`, userID).Scan(&current, &lastAct)
	if err != nil {
		return 0, nil, nil
	}
	return current, lastAct, nil
}

// HasStudiedToday reports LMS activity for the user's local calendar day.
func HasStudiedToday(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, localDate time.Time, timezone *string) (bool, error) {
	_, last, err := loadStreak(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	if last != nil {
		lastDay := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, time.UTC)
		if lastDay.Equal(localDate) {
			return true, nil
		}
	}
	start, end := dayBounds(localDate, timezone)
	var heartbeats int
	err = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.engagement_events
WHERE user_id = $1 AND event_type = 'heartbeat' AND occurred_at >= $2 AND occurred_at < $3
`, userID, start, end).Scan(&heartbeats)
	if err != nil {
		return false, err
	}
	return heartbeats > 0, nil
}

// MinutesStudiedToday counts heartbeat-based study minutes for the local day.
func MinutesStudiedToday(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, localDate time.Time, timezone *string) (int, error) {
	start, end := dayBounds(localDate, timezone)
	var heartbeats int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.engagement_events
WHERE user_id = $1 AND event_type = 'heartbeat' AND occurred_at >= $2 AND occurred_at < $3
`, userID, start, end).Scan(&heartbeats)
	if err != nil {
		return 0, err
	}
	return heartbeats / 2, nil
}

func dayBounds(localDate time.Time, timezone *string) (time.Time, time.Time) {
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		if l, err := time.LoadLocation(*timezone); err == nil {
			loc = l
		}
	}
	y, m, d := localDate.Year(), localDate.Month(), localDate.Day()
	start := time.Date(y, m, d, 0, 0, 0, 0, loc)
	return start.UTC(), start.Add(24 * time.Hour).UTC()
}

func loadWeeklySummary(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, localDate time.Time, timezone *string) (weeklySummary, error) {
	out := weeklySummary{}
	weekStart := localDate
	if localDate.Weekday() != time.Sunday {
		weekStart = localDate.AddDate(0, 0, -int(localDate.Weekday()))
	}
	start, _ := dayBounds(weekStart, timezone)
	end, _ := dayBounds(localDate.AddDate(0, 0, 1), timezone)
	_ = pool.QueryRow(ctx, `
SELECT COALESCE(SUM(xp_awarded), 0) FROM gamification.xp_events
WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
`, userID, start, end).Scan(&out.XPEarned)

	rows, err := pool.Query(ctx, `
SELECT c.id, c.title, ce.id
FROM course.course_enrollments ce
JOIN course.courses c ON c.id = ce.course_id
WHERE ce.user_id = $1 AND ce.active AND c.course_mode = 'self_paced' AND NOT c.archived
ORDER BY c.title
LIMIT 10
`, userID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var textLines []string
	var htmlParts []string
	for rows.Next() {
		var courseID, enrollmentID uuid.UUID
		var title string
		if err := rows.Scan(&courseID, &title, &enrollmentID); err != nil {
			return out, err
		}
		totals, err := learnerprogress.CourseProgress(ctx, pool, courseID, enrollmentID)
		if err != nil {
			continue
		}
		pct := 0
		if totals.TotalItems > 0 {
			pct = int(float64(totals.CompletedItems) / float64(totals.TotalItems) * 100)
		}
		textLines = append(textLines, fmt.Sprintf("%s — %d%% complete", title, pct))
		htmlParts = append(htmlParts, fmt.Sprintf(`<p style="margin:8px 0;"><strong>%s</strong><br/>%d%% complete</p>`, title, pct))
	}
	out.CoursesText = strings.Join(textLines, "\n")
	out.CoursesHTML = strings.Join(htmlParts, "")
	if out.CoursesHTML == "" {
		out.CoursesHTML = `<p style="color:#64748b;">No courses in progress this week.</p>`
		out.CoursesText = "No courses in progress this week."
	}
	return out, rows.Err()
}
