package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	calendartokens "github.com/lextures/lextures/server/internal/repos/calendartokens"
	courserepo "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	calendarsvc "github.com/lextures/lextures/server/internal/service/calendar"
)

func (d Deps) calendarFeedsEnabled() bool {
	return d.effectiveConfig().FFCalendarFeeds
}

func (d Deps) requireCalendarFeedsEnabled(w http.ResponseWriter) bool {
	if !d.calendarFeedsEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Calendar feeds are not enabled on this server.")
		return false
	}
	return true
}

func (d Deps) registerCalendarFeedRoutes(r chi.Router) {
	r.Get("/api/v1/me/calendar.ics", d.handleMeCalendarICS())
	r.Get("/.well-known/caldav", d.handleCalDAVWellKnown())
	r.HandleFunc("/caldav/users/{user_id}/", d.handleCalDAVCollection())
}

func (d Deps) registerCalendarFeedMeRoutes(r chi.Router) {
	r.Post("/api/v1/me/calendar-token", d.handlePostMeCalendarToken())
	r.Get("/api/v1/me/calendar-token", d.handleGetMeCalendarToken())
}

// calendarFeedUserID resolves the user from ?token= or the authenticated session (in-app download).
func (d Deps) calendarFeedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if strings.TrimSpace(r.URL.Query().Get("token")) != "" {
		return d.calendarTokenUserID(w, r)
	}
	return d.meUserID(w, r)
}

// handleMeCalendarICS is GET /api/v1/me/calendar.ics?token= — personal iCal feed for all enrolled courses.
func (d Deps) handleMeCalendarICS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireCalendarFeedsEnabled(w) {
			return
		}
		userID, ok := d.calendarFeedUserID(w, r)
		if !ok {
			return
		}
		d.serveUserCalendarFeed(w, r, userID, nil)
	}
}

// handleCourseCalendarICS is GET /api/v1/courses/{course_code}/calendar.ics?token=
func (d Deps) handleCourseCalendarICS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}

		if d.calendarFeedsEnabled() {
			userID, ok := d.calendarFeedUserID(w, r)
			if ok {
				ctx := r.Context()
				crow, err := courserepo.GetPublicByCourseCode(ctx, d.Pool, courseCode)
				if err != nil || crow == nil {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
					return
				}
				if !d.userCanAccessCourseCalendar(ctx, userID, courseCode, crow.ID) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
					return
				}
				cid, err := uuid.Parse(crow.ID)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
					return
				}
				d.serveUserCalendarFeed(w, r, userID, &cid)
				return
			}
			if strings.TrimSpace(r.URL.Query().Get("token")) != "" {
				return
			}
		}

		// Legacy session-authenticated term-only feed (pre-16.5).
		d.handleCourseICSLegacy(w, r, courseCode)
	}
}

func (d Deps) userCanAccessCourseCalendar(ctx context.Context, userID uuid.UUID, courseCode string, courseID string) bool {
	hasAccess, err := enrollment.UserHasAccess(ctx, d.Pool, courseCode, userID)
	if err != nil {
		return false
	}
	if hasAccess {
		return true
	}
	isStaff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, userID)
	return err == nil && isStaff
}

func (d Deps) serveUserCalendarFeed(w http.ResponseWriter, r *http.Request, userID uuid.UUID, courseID *uuid.UUID) {
	ctx := r.Context()
	now := time.Now()
	rangeStart, rangeEnd := parseCalendarRange(r)

	var cacheKey string
	var courseIDs []uuid.UUID

	if courseID != nil {
		cacheKey = fmt.Sprintf("course:%s:%s", courseID.String(), userID.String())
		courseIDs = []uuid.UUID{*courseID}
	} else {
		cacheKey = "user:" + userID.String()
		courses, err := courserepo.ListForEnrolledUser(ctx, d.Pool, userID, nil)
		if err != nil {
			d.serveCalendarWithCacheFallback(w, cacheKey, "Failed to load courses.")
			return
		}
		courseIDs = make([]uuid.UUID, 0, len(courses))
		for _, c := range courses {
			id, err := uuid.Parse(c.ID)
			if err != nil {
				continue
			}
			courseIDs = append(courseIDs, id)
		}
	}

	if body, ok := calendarsvc.DefaultFeedCache.Get(cacheKey, now); ok {
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("X-Cache", "hit")
		_, _ = w.Write(body)
		return
	}

	events, err := calendarsvc.LoadEventsForCourses(ctx, d.Pool, courseIDs, userID, rangeStart, rangeEnd)
	if err != nil {
		d.serveCalendarWithCacheFallback(w, cacheKey, "Failed to generate calendar feed.")
		return
	}

	tz := d.userTimezone(ctx, userID)
	body := calendarsvc.BuildICalendar(events, d.effectiveConfig().PublicWebOrigin, tz, now)
	calendarsvc.DefaultFeedCache.Set(cacheKey, body, now)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("X-Cache", "miss")
	_, _ = w.Write(body)
}

func (d Deps) serveCalendarWithCacheFallback(w http.ResponseWriter, cacheKey, errMsg string) {
	if body, ok := calendarsvc.DefaultFeedCache.GetStale(cacheKey); ok {
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("X-Cache", "stale")
		_, _ = w.Write(body)
		return
	}
	apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, errMsg)
	w.Header().Set("Retry-After", "60")
}

func (d Deps) userTimezone(ctx context.Context, userID uuid.UUID) string {
	u, err := userrepo.FindByID(ctx, d.Pool, userID)
	if err != nil || u == nil || u.Timezone == nil || strings.TrimSpace(*u.Timezone) == "" {
		return "UTC"
	}
	return strings.TrimSpace(*u.Timezone)
}

func (d Deps) calendarTokenUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Missing calendar token.")
		return uuid.Nil, false
	}
	userID, err := calendartokens.ResolveQueryParam(r.Context(), d.Pool, token, time.Now())
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid or expired calendar token.")
		return uuid.Nil, false
	}
	return userID, true
}

func parseCalendarRange(r *http.Request) (start, end *time.Time) {
	if s := strings.TrimSpace(r.URL.Query().Get("start")); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			start = &t
		}
	}
	if s := strings.TrimSpace(r.URL.Query().Get("end")); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			end = &endOfDay
		}
	}
	return start, end
}

type calendarTokenStatusJSON struct {
	HasToken  bool       `json:"hasToken"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	FeedURL   string     `json:"feedUrl,omitempty"`
}

type calendarTokenCreatedJSON struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	FeedURL   string    `json:"feedUrl"`
}

type calendarCourseFeedJSON struct {
	CourseID   string `json:"courseId"`
	CourseCode string `json:"courseCode"`
	Title      string `json:"title"`
	FeedURL    string `json:"feedUrl"`
}

type calendarTokenInfoJSON struct {
	HasToken        bool                     `json:"hasToken"`
	PersonalFeedURL string                   `json:"personalFeedUrl"`
	ExpiresAt       time.Time                `json:"expiresAt"`
	CourseFeeds     []calendarCourseFeedJSON `json:"courseFeeds"`
}

func (d Deps) handleGetMeCalendarToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requireCalendarFeedsEnabled(w) {
			return
		}
		ctx := r.Context()
		now := time.Now()
		row, err := calendartokens.GetActiveForUser(ctx, d.Pool, userID, now)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load calendar token.")
			return
		}
		if row == nil {
			writeJSON(w, http.StatusOK, calendarTokenStatusJSON{HasToken: false})
			return
		}
		base := calendarFeedBaseURL(r, d.effectiveConfig().PublicWebOrigin)
		writeJSON(w, http.StatusOK, calendarTokenInfoJSON{
			HasToken:        true,
			PersonalFeedURL: base + "/api/v1/me/calendar.ics?token=<token>",
			ExpiresAt:       row.ExpiresAt,
			CourseFeeds:     d.listCourseFeedURLs(ctx, userID, base),
		})
	}
}

func (d Deps) handlePostMeCalendarToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requireCalendarFeedsEnabled(w) {
			return
		}
		ctx := r.Context()
		now := time.Now()
		row, secret, err := calendartokens.RotateForUser(ctx, d.Pool, userID, now)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate calendar token.")
			return
		}
		calendarsvc.DefaultFeedCache.InvalidateUser(userID.String())
		base := calendarFeedBaseURL(r, d.effectiveConfig().PublicWebOrigin)
		writeJSON(w, http.StatusOK, calendarTokenCreatedJSON{
			Token:     secret,
			ExpiresAt: row.ExpiresAt,
			FeedURL:   base + "/api/v1/me/calendar.ics?token=" + secret,
		})
	}
}

func (d Deps) listCourseFeedURLs(ctx context.Context, userID uuid.UUID, base string) []calendarCourseFeedJSON {
	courses, err := courserepo.ListForEnrolledUser(ctx, d.Pool, userID, nil)
	if err != nil {
		return nil
	}
	out := make([]calendarCourseFeedJSON, 0, len(courses))
	for _, c := range courses {
		out = append(out, calendarCourseFeedJSON{
			CourseID:   c.ID,
			CourseCode: c.CourseCode,
			Title:      c.Title,
			FeedURL:    fmt.Sprintf("%s/api/v1/courses/%s/calendar.ics?token=<token>", base, c.CourseCode),
		})
	}
	return out
}

func calendarFeedBaseURL(r *http.Request, configuredOrigin string) string {
	if o := strings.TrimRight(strings.TrimSpace(configuredOrigin), "/"); o != "" {
		return o
	}
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

// handleCalDAVWellKnown redirects to the per-user CalDAV collection (RFC 4791 discovery).
func (d Deps) handleCalDAVWellKnown() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.calendarFeedsEnabled() {
			http.NotFound(w, r)
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing token query parameter.")
			return
		}
		userID, err := calendartokens.ResolveQueryParam(r.Context(), d.Pool, token, time.Now())
		if err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid or expired calendar token.")
			return
		}
		base := calendarFeedBaseURL(r, d.effectiveConfig().PublicWebOrigin)
		target := fmt.Sprintf("%s/caldav/users/%s/?token=%s", base, userID.String(), token)
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	}
}

func (d Deps) handleCalDAVCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.calendarFeedsEnabled() {
			http.NotFound(w, r)
			return
		}
		userIDParam := chi.URLParam(r, "user_id")
		pathUserID, err := uuid.Parse(userIDParam)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		tokenUserID, ok := d.calendarTokenUserID(w, r)
		if !ok {
			return
		}
		if tokenUserID != pathUserID {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Token does not match user.")
			return
		}

		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Allow", "OPTIONS, PROPFIND, GET")
			w.Header().Set("DAV", "1, calendar-access")
			w.WriteHeader(http.StatusNoContent)
			return
		case "PROPFIND":
			base := calendarFeedBaseURL(r, d.effectiveConfig().PublicWebOrigin)
			feedHref := fmt.Sprintf("%s/api/v1/me/calendar.ics?token=%s", base, strings.TrimSpace(r.URL.Query().Get("token")))
			body := caldavPropfindResponse(feedHref)
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(body))
			return
		case http.MethodGet:
			d.handleMeCalendarICS()(w, r)
			return
		default:
			w.Header().Set("Allow", "OPTIONS, PROPFIND, GET")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func caldavPropfindResponse(calendarHref string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:response>
    <D:href>` + xmlEscape(calendarHref) + `</D:href>
    <D:propstat>
      <D:prop>
        <D:resourcetype><D:collection/><C:calendar/></D:resourcetype>
        <D:displayname>Lextures Calendar</D:displayname>
        <C:calendar-description>Lextures assignment and quiz deadlines</C:calendar-description>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
</D:multistatus>`
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
