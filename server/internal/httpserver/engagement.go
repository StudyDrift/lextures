package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
)

func (d Deps) engagementFeatureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().EngagementTrackingEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Engagement tracking is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireEngagementInstructor(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, uuid.UUID, bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	if !d.engagementFeatureEnabled(w) {
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view engagement data.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	return courseCode, viewer, *cid, true
}

type engagementEventInput struct {
	ItemID     *string  `json:"itemId"`
	ItemType   *string  `json:"itemType"`
	CourseID   *string  `json:"courseId"`
	EventType  string   `json:"eventType"`
	Value      *float32 `json:"value"`
	OccurredAt *string  `json:"occurredAt"`
}

// handlePostEngagementEvents is POST /api/v1/analytics/events
func (d Deps) handlePostEngagementEvents() http.HandlerFunc {
	const maxBatchSize = 50
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.engagementFeatureEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var events []engagementEventInput
		if err := json.Unmarshal(b, &events); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(events) == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"stored": 0})
			return
		}
		if len(events) > maxBatchSize {
			events = events[:maxBatchSize]
		}
		stored, err := insertEngagementEvents(r.Context(), d.Pool, userID, events)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store engagement events.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"stored": stored})
	}
}

func insertEngagementEvents(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, events []engagementEventInput) (int, error) {
	type row struct {
		courseID   *uuid.UUID
		itemID     *uuid.UUID
		itemType   *string
		eventType  string
		value      *float32
		occurredAt time.Time
	}
	rows := make([]row, 0, len(events))
	for _, e := range events {
		if e.EventType == "" {
			continue
		}
		ro := row{
			eventType:  e.EventType,
			value:      e.Value,
			occurredAt: time.Now().UTC(),
			itemType:   e.ItemType,
		}
		if e.CourseID != nil {
			if id, err := uuid.Parse(*e.CourseID); err == nil {
				ro.courseID = &id
			}
		}
		if e.ItemID != nil {
			if id, err := uuid.Parse(*e.ItemID); err == nil {
				ro.itemID = &id
			}
		}
		if e.OccurredAt != nil {
			if t, err := time.Parse(time.RFC3339, *e.OccurredAt); err == nil {
				ro.occurredAt = t.UTC()
			}
		}
		rows = append(rows, ro)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	batch := &pgx.Batch{}
	for _, ro := range rows {
		batch.Queue(`
INSERT INTO analytics.engagement_events (user_id, course_id, item_id, item_type, event_type, value, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, userID, ro.courseID, ro.itemID, ro.itemType, ro.eventType, ro.value, ro.occurredAt)
	}
	results := pool.SendBatch(ctx, batch)
	stored := 0
	for range rows {
		if _, err := results.Exec(); err != nil {
			continue
		}
		stored++
	}
	return stored, results.Close()
}

// handleGetEnrollmentEngagement is GET /api/v1/courses/{course_code}/enrollments/{enrollment_id}/engagement
func (d Deps) handleGetEnrollmentEngagement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.engagementFeatureEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		eidStr := chi.URLParam(r, "enrollment_id")
		eid, err := uuid.Parse(eidStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		ctx := r.Context()

		var enrolledUserID, courseID uuid.UUID
		if err := d.Pool.QueryRow(ctx, `
SELECT e.user_id, e.course_id
FROM course.course_enrollments e
WHERE e.id = $1 AND e.active = true
`, eid).Scan(&enrolledUserID, &courseID); err != nil {
			if err == pgx.ErrNoRows {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}

		isSelf := viewer == enrolledUserID
		if !isSelf {
			var cc string
			if err := d.Pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&cc); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
				return
			}
			has, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+cc+":gradebook:view")
			if err != nil || !has {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view this enrollment's engagement.")
				return
			}
		}

		summary, err := loadEngagementSummary(ctx, d.Pool, eid, enrolledUserID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load engagement data.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

type engagementSummaryJSON struct {
	EnrollmentID            string   `json:"enrollmentId"`
	AvgTimeOnTaskPerSession float64  `json:"avgTimeOnTaskPerSession"`
	LoginsLast7Days         int      `json:"loginsLast7Days"`
	AvgVideoWatchPct        *float64 `json:"avgVideoWatchPct"`
	AvgScrollDepth          *float64 `json:"avgScrollDepth"`
	DataAsOf                string   `json:"dataAsOf"`
}

func loadEngagementSummary(ctx context.Context, pool *pgxpool.Pool, enrollmentID, userID, courseID uuid.UUID) (engagementSummaryJSON, error) {
	out := engagementSummaryJSON{
		EnrollmentID: enrollmentID.String(),
		DataAsOf:     time.Now().UTC().Format(time.RFC3339),
	}

	var heartbeats, sessions int64
	_ = pool.QueryRow(ctx, `
SELECT
    COALESCE(COUNT(*) FILTER (WHERE event_type = 'heartbeat'), 0),
    COALESCE(COUNT(DISTINCT date_trunc('day', occurred_at)) FILTER (WHERE event_type = 'heartbeat'), 0)
FROM analytics.engagement_events
WHERE user_id = $1 AND course_id = $2
`, userID, courseID).Scan(&heartbeats, &sessions)
	if sessions > 0 {
		out.AvgTimeOnTaskPerSession = float64(heartbeats*30) / float64(sessions)
	}

	var logins int
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM "user".user_audit
WHERE user_id = $1
  AND event_kind = 'course_visit'
  AND occurred_at >= now() - interval '7 days'
`, userID).Scan(&logins)
	out.LoginsLast7Days = logins

	var avgVideoWatchPct *float64
	_ = pool.QueryRow(ctx, `
SELECT AVG(max_pct)
FROM (
    SELECT item_id, MAX(value) AS max_pct
    FROM analytics.engagement_events
    WHERE user_id = $1 AND course_id = $2 AND event_type = 'video_progress'
    GROUP BY item_id
) sub
`, userID, courseID).Scan(&avgVideoWatchPct)
	out.AvgVideoWatchPct = avgVideoWatchPct

	var avgScrollDepth *float64
	_ = pool.QueryRow(ctx, `
SELECT AVG(max_depth)
FROM (
    SELECT item_id, MAX(value) AS max_depth
    FROM analytics.engagement_events
    WHERE user_id = $1 AND course_id = $2 AND event_type = 'scroll_depth'
    GROUP BY item_id
) sub
`, userID, courseID).Scan(&avgScrollDepth)
	out.AvgScrollDepth = avgScrollDepth

	return out, nil
}

type videoDropoffPoint struct {
	Second           int     `json:"second"`
	PctStillWatching float64 `json:"pctStillWatching"`
}

// handleGetVideoDropoff is GET /api/v1/courses/{course_code}/analytics/video-dropoff/{object_id}
func (d Deps) handleGetVideoDropoff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, _, courseID, ok := d.requireEngagementInstructor(w, r)
		if !ok {
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object id.")
			return
		}
		ctx := r.Context()

		var totalWatchers int64
		if err := d.Pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT user_id)
FROM analytics.engagement_events
WHERE course_id = $1 AND item_id = $2 AND event_type = 'video_progress'
`, courseID, objectID).Scan(&totalWatchers); err != nil || totalWatchers == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"objectId":         objectID.String(),
				"totalWatchers":    0,
				"dropoff":          []videoDropoffPoint{},
				"medianStopSecond": nil,
			})
			return
		}

		// Each video_progress event's value is percent watched 0–100.
		// We approximate the stop second as max_pct/100 * estimated_duration.
		// Without knowing exact duration we use relative percentile ranking.
		rows, err := d.Pool.Query(ctx, `
SELECT MAX(value)::float8 AS max_pct
FROM analytics.engagement_events
WHERE course_id = $1 AND item_id = $2 AND event_type = 'video_progress'
GROUP BY user_id
ORDER BY max_pct ASC
`, courseID, objectID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load drop-off data.")
			return
		}
		defer rows.Close()
		var watchPcts []float64
		for rows.Next() {
			var pct float64
			if err := rows.Scan(&pct); err != nil {
				continue
			}
			watchPcts = append(watchPcts, pct)
		}
		_ = rows.Err()

		dropoff := buildDropoffCurve(watchPcts, int(totalWatchers))
		var medianStop *float64
		if len(watchPcts) > 0 {
			m := watchPcts[len(watchPcts)/2]
			medianStop = &m
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objectId":            objectID.String(),
			"totalWatchers":       totalWatchers,
			"dropoff":             dropoff,
			"medianStopPct":       medianStop,
		})
	}
}

func buildDropoffCurve(watchPcts []float64, total int) []videoDropoffPoint {
	if total == 0 || len(watchPcts) == 0 {
		return []videoDropoffPoint{}
	}
	// watchPcts is sorted ascending. Build 20 evenly spaced buckets over 0–100%.
	const buckets = 20
	points := make([]videoDropoffPoint, 0, buckets+1)
	for i := 0; i <= buckets; i++ {
		threshold := float64(i) / float64(buckets) * 100
		// Count watchers who reached at least this percentage.
		stillWatching := 0
		for _, p := range watchPcts {
			if p >= threshold {
				stillWatching++
			}
		}
		points = append(points, videoDropoffPoint{
			Second:           i * (3600 / buckets), // scale to representative seconds
			PctStillWatching: float64(stillWatching) / float64(total) * 100,
		})
	}
	return points
}

type engagementOverviewRow struct {
	EnrollmentID     string   `json:"enrollmentId"`
	UserID           string   `json:"userId"`
	DisplayName      string   `json:"displayName"`
	LoginsLast7Days  int      `json:"loginsLast7Days"`
	AvgTimeOnTaskMin float64  `json:"avgTimeOnTaskMin"`
	AvgVideoWatchPct *float64 `json:"avgVideoWatchPct"`
	AvgScrollDepth   *float64 `json:"avgScrollDepth"`
	EngagementScore  float64  `json:"engagementScore"`
}

// handleGetEngagementOverview is GET /api/v1/courses/{course_code}/analytics/engagement-overview
func (d Deps) handleGetEngagementOverview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, _, courseID, ok := d.requireEngagementInstructor(w, r)
		if !ok {
			return
		}
		ctx := r.Context()

		type enrollmentRow struct {
			EnrollmentID uuid.UUID
			UserID       uuid.UUID
			DisplayName  *string
		}
		enrollRows, err := d.Pool.Query(ctx, `
SELECT e.id, e.user_id,
    COALESCE(u.display_name, u.email) AS display_name
FROM course.course_enrollments e
JOIN "user".users u ON u.id = e.user_id
WHERE e.course_id = $1 AND e.active = true
ORDER BY display_name ASC
`, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollments.")
			return
		}
		defer enrollRows.Close()
		var enrollments []enrollmentRow
		for enrollRows.Next() {
			var er enrollmentRow
			if err := enrollRows.Scan(&er.EnrollmentID, &er.UserID, &er.DisplayName); err != nil {
				continue
			}
			enrollments = append(enrollments, er)
		}
		_ = enrollRows.Err()

		out := make([]engagementOverviewRow, 0, len(enrollments))
		for _, en := range enrollments {
			summary, err := loadEngagementSummary(ctx, d.Pool, en.EnrollmentID, en.UserID, courseID)
			if err != nil {
				continue
			}
			name := "Student"
			if en.DisplayName != nil && *en.DisplayName != "" {
				name = *en.DisplayName
			}
			row := engagementOverviewRow{
				EnrollmentID:     en.EnrollmentID.String(),
				UserID:           en.UserID.String(),
				DisplayName:      name,
				LoginsLast7Days:  summary.LoginsLast7Days,
				AvgTimeOnTaskMin: summary.AvgTimeOnTaskPerSession / 60,
				AvgVideoWatchPct: summary.AvgVideoWatchPct,
				AvgScrollDepth:   summary.AvgScrollDepth,
				EngagementScore:  engagementScore(summary),
			}
			out = append(out, row)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"students": out})
	}
}

func engagementScore(s engagementSummaryJSON) float64 {
	loginScore := clamp(float64(s.LoginsLast7Days)/5.0, 0, 1) * 40
	timeScore := clamp(s.AvgTimeOnTaskPerSession/1800.0, 0, 1) * 30
	videoScore := 0.0
	if s.AvgVideoWatchPct != nil {
		videoScore = clamp(*s.AvgVideoWatchPct/100.0, 0, 1) * 15
	}
	scrollScore := 0.0
	if s.AvgScrollDepth != nil {
		scrollScore = clamp(*s.AvgScrollDepth/100.0, 0, 1) * 15
	}
	return loginScore + timeScore + videoScore + scrollScore
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (d Deps) registerEngagementRoutes(r chi.Router) {
	r.Post("/api/v1/analytics/events", d.handlePostEngagementEvents())
	r.Get("/api/v1/courses/{course_code}/enrollments/{enrollment_id}/engagement", d.handleGetEnrollmentEngagement())
	r.Get("/api/v1/courses/{course_code}/analytics/video-dropoff/{object_id}", d.handleGetVideoDropoff())
	r.Get("/api/v1/courses/{course_code}/analytics/engagement-overview", d.handleGetEngagementOverview())
}
