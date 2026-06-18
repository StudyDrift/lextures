package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/gamification"
)

func (d Deps) gamificationEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFGamification {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Gamification is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerGamificationRoutes(r chi.Router) {
	r.Get("/api/v1/me/gamification", d.handleGetMyGamification())
	r.Get("/api/v1/me/badges", d.handleGetMyBadges())
	r.Post("/api/v1/me/gamification/freeze-streak", d.handlePostFreezeStreak())
	r.Get("/api/v1/courses/{course_code}/leaderboard", d.handleGetCourseLeaderboard())
}

func (d Deps) handleGetMyGamification() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gamificationEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		profile, err := gamification.LoadProfile(r.Context(), d.Pool, userID, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load gamification profile.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(profile)
	}
}

func (d Deps) handleGetMyBadges() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gamificationEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		profile, err := gamification.LoadProfile(r.Context(), d.Pool, userID, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load badges.")
			return
		}
		type resp struct {
			Badges []gamification.Badge `json:"badges"`
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Badges: profile.Badges})
	}
}

func (d Deps) handlePostFreezeStreak() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.gamificationEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		err := gamification.SpendStreakFreeze(r.Context(), d.Pool, userID, time.Now().UTC(), tz)
		if err != nil {
			switch err {
			case gamification.ErrNoFreezes:
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "No streak freezes available.")
			case gamification.ErrNoActiveStreak:
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "You do not have an active streak to protect.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not apply streak freeze.")
			}
			return
		}
		profile, err := gamification.LoadProfile(r.Context(), d.Pool, userID, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load gamification profile.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(profile)
	}
}

func (d Deps) handleGetCourseLeaderboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gamificationEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		var courseID uuid.UUID
		var enabled bool
		err := d.Pool.QueryRow(r.Context(), `
SELECT id, gamification_enabled FROM course.courses WHERE course_code = $1
`, courseCode).Scan(&courseID, &enabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if !enabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Gamification is not enabled for this course.")
			return
		}
		resp, err := gamification.LoadCourseLeaderboard(r.Context(), d.Pool, courseID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load leaderboard.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
