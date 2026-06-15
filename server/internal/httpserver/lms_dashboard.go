package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefeed"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/coursesections"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/recommendations"
	"github.com/lextures/lextures/server/internal/repos/srs"
)

// Stubs and thin reads for LMS dashboard until full ports.

func (d Deps) handleLearnerReviewStats() http.HandlerFunc {
	type resp struct {
		Streak            int     `json:"streak"`
		DueToday          int64   `json:"dueToday"`
		DueWeek           int64   `json:"dueWeek"`
		RetentionEstimate float64 `json:"retentionEstimate"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		learner, err := uuid.Parse(chi.URLParam(r, "user_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learner id.")
			return
		}
		can, err := assertCanReadLearnerState(r.Context(), d.Pool, viewer, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !can {
			writeLearnerAccessDenied(w)
			return
		}
		streak, dueToday, dueWeek, retention, err := srs.ReviewStats(r.Context(), d.Pool, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review stats.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{
			Streak:            streak,
			DueToday:          dueToday,
			DueWeek:           dueWeek,
			RetentionEstimate: retention,
		})
	}
}

func (d Deps) handleLearnerReviewQueue() http.HandlerFunc {
	type item struct {
		StateID       string           `json:"stateId"`
		QuestionID    string           `json:"questionId"`
		CourseID      string           `json:"courseId"`
		CourseCode    string           `json:"courseCode"`
		CourseTitle   string           `json:"courseTitle"`
		NextReviewAt  string           `json:"nextReviewAt"`
		Stem          string           `json:"stem"`
		QuestionType  string           `json:"questionType"`
		Options       *json.RawMessage `json:"options,omitempty"`
		CorrectAnswer *json.RawMessage `json:"correctAnswer,omitempty"`
		Explanation   *string          `json:"explanation,omitempty"`
	}
	type resp struct {
		Items    []item `json:"items"`
		TotalDue int64  `json:"totalDue"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		learner, err := uuid.Parse(chi.URLParam(r, "user_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learner id.")
			return
		}
		can, err := assertCanReadLearnerState(r.Context(), d.Pool, viewer, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !can {
			writeLearnerAccessDenied(w)
			return
		}
		limit := int64(50)
		if v := r.URL.Query().Get("limit"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				limit = parsed
			}
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 200 {
			limit = 200
		}
		offset := int64(0)
		if v := r.URL.Query().Get("offset"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
				offset = parsed
			}
		}
		total, err := srs.CountDueForUser(r.Context(), d.Pool, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review queue.")
			return
		}
		rows, err := srs.ListReviewQueue(r.Context(), d.Pool, learner, limit, offset)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review queue.")
			return
		}
		items := make([]item, 0, len(rows))
		for _, row := range rows {
			it := item{
				StateID:      row.StateID.String(),
				QuestionID:   row.QuestionID.String(),
				CourseID:     row.CourseID.String(),
				CourseCode:   row.CourseCode,
				CourseTitle:  row.CourseTitle,
				NextReviewAt: row.NextReviewAt.UTC().Format(time.RFC3339),
				Stem:         row.Stem,
				QuestionType: row.QuestionType,
				Explanation:  row.Explanation,
			}
			if len(row.Options) > 0 {
				raw := json.RawMessage(append([]byte(nil), row.Options...))
				it.Options = &raw
			}
			if len(row.CorrectAnswer) > 0 {
				raw := json.RawMessage(append([]byte(nil), row.CorrectAnswer...))
				it.CorrectAnswer = &raw
			}
			items = append(items, it)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Items: items, TotalDue: total})
	}
}

func (d Deps) handleLearnerRecommendations() http.HandlerFunc {
	type item struct {
		ItemID   string  `json:"itemId"`
		ItemType string  `json:"itemType"`
		Title    string  `json:"title"`
		Surface  string  `json:"surface"`
		Reason   string  `json:"reason"`
		Score    float64 `json:"score"`
	}
	type resp struct {
		Recommendations []item `json:"recommendations"`
		Degraded        bool   `json:"degraded,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		learner, err := uuid.Parse(chi.URLParam(r, "user_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learner id.")
			return
		}
		can, err := assertCanReadLearnerState(r.Context(), d.Pool, viewer, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !can {
			writeLearnerAccessDenied(w)
			return
		}
		courseIDStr := strings.TrimSpace(r.URL.Query().Get("courseId"))
		if courseIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseId is required.")
			return
		}
		courseID, err := uuid.Parse(courseIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
			return
		}
		surface := strings.TrimSpace(r.URL.Query().Get("surface"))
		if surface == "" {
			surface = "continue"
		}
		if surface != "continue" && surface != "strengthen" && surface != "challenge" && surface != "review" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "surface must be continue, strengthen, challenge, or review.")
			return
		}
		okAccess, err := enrollment.UserHasAccessByCourseID(r.Context(), d.Pool, courseID, learner)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return
		}
		if !okAccess {
			writeLearnerAccessDenied(w)
			return
		}
		limit := 10
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 10 {
			limit = 10
		}
		cached, expired, err := recommendations.GetCache(r.Context(), d.Pool, learner, courseID, surface)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load recommendations.")
			return
		}
		out := resp{Recommendations: []item{}, Degraded: false}
		if cached != nil && !expired {
			for _, raw := range cached.Recommendations {
				var it item
				if err := json.Unmarshal(raw, &it); err != nil {
					continue
				}
				out.Recommendations = append(out.Recommendations, it)
			}
			if cached.Degraded {
				out.Degraded = true
			}
		} else {
			out.Degraded = true
		}
		if len(out.Recommendations) > limit {
			out.Recommendations = out.Recommendations[:limit]
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) requireCourseAccess(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, bool) {
	viewer, ok := d.meUserID(w, r)
	if !ok {
		return "", uuid.UUID{}, false
	}
	courseCode := chi.URLParam(r, "course_code")
	if courseCode == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
		return "", uuid.UUID{}, false
	}
	has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
		return "", uuid.UUID{}, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, false
	}
	if !auth.AccessKeyAllowsCourse(r.Context(), *cid) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, false
	}
	return courseCode, viewer, true
}

func (d Deps) handleCourseStructure() http.HandlerFunc {
	type resp struct {
		Items []coursestructure.ItemResponse `json:"items"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		staffView, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		items, err := coursestructure.ListForCourseWithEnrichment(r.Context(), d.Pool, *cid, staffView)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course structure.")
			return
		}
		crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err == nil && crow != nil && crow.SectionsEnabled && !staffView {
			secID, err := enrollment.GetStudentSectionID(r.Context(), d.Pool, *cid, viewer)
			if err == nil && secID != nil {
				ovm, err := coursesections.ListOverridesForSection(r.Context(), d.Pool, *secID)
				if err == nil && len(ovm) > 0 {
					applySectionAssignmentOverrides(items, ovm)
				}
			}
		}
		if d.readingLevelEnabled() && staffView {
			_ = coursestructure.ApplyReadingLevelMetadata(r.Context(), d.Pool, *cid, items)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Items: items})
	}
}

func (d Deps) handleFeedChannels() http.HandlerFunc {
	type ch struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SortOrder int    `json:"sortOrder"`
		CreatedAt string `json:"createdAt"`
	}
	type resp struct {
		Channels []ch `json:"channels"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := coursefeed.ListChannels(r.Context(), d.Pool, *cid, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feed channels.")
			return
		}
		channels := make([]ch, 0, len(rows))
		for _, row := range rows {
			channels = append(channels, ch{
				ID:        row.ID.String(),
				Name:      row.Name,
				SortOrder: row.SortOrder,
				CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Channels: channels})
	}
}

func (d Deps) handleCreateFeedChannel() http.HandlerFunc {
	type req struct {
		Name string `json:"name"`
	}
	type resp struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SortOrder int    `json:"sortOrder"`
		CreatedAt string `json:"createdAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Channel name is required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		ch, err := coursefeed.CreateChannel(r.Context(), d.Pool, *cid, viewer, name)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create feed channel.")
			return
		}
		d.FeedHub.ChannelsChanged(courseCode)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{
			ID:        ch.ID.String(),
			Name:      ch.Name,
			SortOrder: ch.SortOrder,
			CreatedAt: ch.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

// handleUpdateFeedChannel is PATCH /api/v1/courses/{course_code}/feed/channels/{channel_id} — rename a channel.
func (d Deps) handleUpdateFeedChannel() http.HandlerFunc {
	type req struct {
		Name string `json:"name"`
	}
	type resp struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SortOrder int    `json:"sortOrder"`
		CreatedAt string `json:"createdAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		channelID, err := uuid.Parse(chi.URLParam(r, "channel_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid channel id.")
			return
		}
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Channel name is required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		ch, err := coursefeed.UpdateChannel(r.Context(), d.Pool, *cid, channelID, name)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update feed channel.")
			return
		}
		if ch == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Channel not found.")
			return
		}
		d.FeedHub.ChannelsChanged(courseCode)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{
			ID:        ch.ID.String(),
			Name:      ch.Name,
			SortOrder: ch.SortOrder,
			CreatedAt: ch.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

// handleDeleteFeedChannel is DELETE /api/v1/courses/{course_code}/feed/channels/{channel_id}.
func (d Deps) handleDeleteFeedChannel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		channelID, err := uuid.Parse(chi.URLParam(r, "channel_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid channel id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		deleted, err := coursefeed.DeleteChannel(r.Context(), d.Pool, *cid, channelID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete feed channel.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Channel not found.")
			return
		}
		d.FeedHub.ChannelsChanged(courseCode)
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleFeedRoster is GET /api/v1/courses/{course_code}/feed/roster — people for @mentions.
func (d Deps) handleFeedRoster() http.HandlerFunc {
	type person struct {
		UserID      string  `json:"userId"`
		Email       string  `json:"email"`
		DisplayName *string `json:"displayName"`
		AvatarURL   *string `json:"avatarUrl,omitempty"`
	}
	type resp struct {
		People []person `json:"people"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		rows, err := enrollment.ListFeedRosterForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feed roster.")
			return
		}
		people := make([]person, 0, len(rows))
		for _, p := range rows {
			people = append(people, person{
				UserID:      p.UserID.String(),
				Email:       p.Email,
				DisplayName: p.DisplayName,
				AvatarURL:   p.AvatarURL,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{People: people})
	}
}

func (d Deps) handleFeedMessagesList() http.HandlerFunc {
	type resp struct {
		Messages []coursefeed.MessagePublic `json:"messages"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		channelID, err := uuid.Parse(chi.URLParam(r, "channel_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid channel id.")
			return
		}
		belongs, err := coursefeed.ChannelBelongsToCourse(r.Context(), d.Pool, *cid, channelID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load channel.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		limitRoots := int64(200)
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid limit.")
				return
			}
			if n > 200 {
				n = 200
			}
			limitRoots = int64(n)
		}
		msgs, err := coursefeed.ListMessagesThreaded(r.Context(), d.Pool, channelID, viewer, limitRoots)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load messages.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Messages: msgs})
	}
}

func (d Deps) handleFeedMessagePost() http.HandlerFunc {
	type req struct {
		Body             string   `json:"body"`
		ParentMessageID  *string  `json:"parentMessageId"`
		MentionUserIDs   []string `json:"mentionUserIds"`
		MentionsEveryone bool     `json:"mentionsEveryone"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		channelID, err := uuid.Parse(chi.URLParam(r, "channel_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid channel id.")
			return
		}
		belongs, err := coursefeed.ChannelBelongsToCourse(r.Context(), d.Pool, *cid, channelID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load channel.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		body := strings.TrimSpace(in.Body)
		if body == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Message body is required.")
			return
		}
		var parentID *uuid.UUID
		if in.ParentMessageID != nil && strings.TrimSpace(*in.ParentMessageID) != "" {
			p, err := uuid.Parse(strings.TrimSpace(*in.ParentMessageID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid parentMessageId.")
				return
			}
			ok, err := coursefeed.ParentIsRootInChannel(r.Context(), d.Pool, channelID, p)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate parent message.")
				return
			}
			if !ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Reply parent must be a root message in this channel.")
				return
			}
			parentID = &p
		}
		mentions := make([]uuid.UUID, 0, len(in.MentionUserIDs))
		for _, m := range in.MentionUserIDs {
			u, err := uuid.Parse(strings.TrimSpace(m))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid mentionUserIds.")
				return
			}
			mentions = append(mentions, u)
		}
		id, err := coursefeed.CreateMessage(r.Context(), d.Pool, channelID, viewer, body, parentID, mentions, in.MentionsEveryone)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create message.")
			return
		}
		d.FeedHub.MessagesChanged(courseCode, channelID.String())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id.String()})
	}
}

func (d Deps) handleCourseEnrollmentsList() http.HandlerFunc {
	type row struct {
		ID             string  `json:"id"`
		UserID         string  `json:"userId"`
		DisplayName    *string `json:"displayName"`
		AvatarURL      *string `json:"avatarUrl,omitempty"`
		Role           string  `json:"role"`
		RoleDisplay    *string `json:"roleDisplay,omitempty"`
		SectionID      *string `json:"sectionId,omitempty"`
		SectionCode    *string `json:"sectionCode,omitempty"`
		SectionName    *string `json:"sectionName,omitempty"`
		State          *string `json:"state,omitempty"`
		StateChangedAt *string `json:"stateChangedAt,omitempty"`
		StateReason    *string `json:"stateReason,omitempty"`
	}
	type resp struct {
		Enrollments []row `json:"enrollments"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canList, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:read")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canList {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the roster.")
			return
		}
		roster, err := enrollment.ListRosterForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollments.")
			return
		}
		out := make([]row, 0, len(roster))
		for _, e := range roster {
			r := row{
				ID:          e.ID.String(),
				UserID:      e.UserID.String(),
				DisplayName: e.DisplayName,
				AvatarURL:   e.AvatarURL,
				Role:        e.Role,
				RoleDisplay: e.RoleDisplay,
			}
			if e.SectionID != nil {
				s := e.SectionID.String()
				r.SectionID = &s
			}
			if e.SectionCode != nil {
				r.SectionCode = e.SectionCode
			}
			if e.SectionName != nil {
				r.SectionName = e.SectionName
			}
			if d.effectiveConfig().FFEnrollmentStateMachine && e.State != "" {
				s := e.State
				r.State = &s
				if e.StateChangedAt != nil {
					ts := e.StateChangedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
					r.StateChangedAt = &ts
				}
				r.StateReason = e.StateReason
			}
			out = append(out, r)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Enrollments: out})
	}
}

// handlePatchFeedMessage is PATCH /api/v1/courses/{course_code}/feed/messages/{message_id}
func (d Deps) handlePatchFeedMessage() http.HandlerFunc {
	type req struct {
		Body string `json:"body"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		belongs, err := coursefeed.MessageBelongsToCourse(r.Context(), d.Pool, *cid, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate message.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		author, _, err := coursefeed.GetMessageAuthorAndIsRoot(r.Context(), d.Pool, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load message.")
			return
		}
		if author != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only the author can edit this message.")
			return
		}
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		body := strings.TrimSpace(in.Body)
		if body == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Message body is required.")
			return
		}
		if len(body) > 8000 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Message body is too long.")
			return
		}
		if err := coursefeed.UpdateMessageBody(r.Context(), d.Pool, messageID, viewer, body); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update message.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

// handleDeleteFeedMessage is DELETE /api/v1/courses/{course_code}/feed/messages/{message_id}
func (d Deps) handleDeleteFeedMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		belongs, err := coursefeed.MessageBelongsToCourse(r.Context(), d.Pool, *cid, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate message.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		author, _, err := coursefeed.GetMessageAuthorAndIsRoot(r.Context(), d.Pool, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load message.")
			return
		}
		if author != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only the author can delete this message.")
			return
		}
		deleted, err := coursefeed.DeleteMessage(r.Context(), d.Pool, messageID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete message.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePinFeedMessage is PATCH /api/v1/courses/{course_code}/feed/messages/{message_id}/pin
func (d Deps) handlePinFeedMessage() http.HandlerFunc {
	type req struct {
		Pinned bool `json:"pinned"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		belongs, err := coursefeed.MessageBelongsToCourse(r.Context(), d.Pool, *cid, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate message.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		_, isRoot, err := coursefeed.GetMessageAuthorAndIsRoot(r.Context(), d.Pool, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load message.")
			return
		}
		if !isRoot {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only root messages can be pinned.")
			return
		}
		canMod, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canMod {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only course staff can pin messages.")
			return
		}
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := coursefeed.SetMessagePinned(r.Context(), d.Pool, messageID, in.Pinned, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update pin.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

// handleLikeFeedMessage is POST /api/v1/courses/{course_code}/feed/messages/{message_id}/like
func (d Deps) handleLikeFeedMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		belongs, err := coursefeed.MessageBelongsToCourse(r.Context(), d.Pool, *cid, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate message.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if err := coursefeed.AddLike(r.Context(), d.Pool, messageID, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to like message.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

// handleUnlikeFeedMessage is DELETE /api/v1/courses/{course_code}/feed/messages/{message_id}/like
func (d Deps) handleUnlikeFeedMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		belongs, err := coursefeed.MessageBelongsToCourse(r.Context(), d.Pool, *cid, messageID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate message.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if err := coursefeed.RemoveLike(r.Context(), d.Pool, messageID, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to unlike message.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

// handleUploadFeedImage is POST /api/v1/courses/{course_code}/feed/upload-image
// Accepts multipart file, stores under course files (same as other uploads), records metadata,
// and returns the usable content_path for use in markdown inside feed posts.
func (d Deps) handleUploadFeedImage() http.HandlerFunc {
	type resp struct {
		ID          string `json:"id"`
		ContentPath string `json:"content_path"`
		MimeType    string `json:"mime_type"`
		ByteSize    int64  `json:"byte_size"`
	}
	const maxImageSize = 10 * 1024 * 1024 // 10 MiB
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}

		// Parse multipart (limit total ~12MB to account for overhead)
		if err := r.ParseMultipartForm(12 << 20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form or file too large.")
			return
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing 'file' part.")
			return
		}
		defer func() { _ = f.Close() }()

		ct := strings.TrimSpace(header.Header.Get("Content-Type"))
		if ct == "" {
			ct = "application/octet-stream"
		}
		if !strings.HasPrefix(ct, "image/") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only image/* uploads are allowed.")
			return
		}
		if header.Size > maxImageSize || header.Size <= 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Image exceeds 10MB limit.")
			return
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg" // fallback
		}
		fileUUID := uuid.New().String()
		storageKey := fmt.Sprintf("files/%s/%s%s", courseCode, fileUUID, ext)

		cfg := d.effectiveConfig()
		if d.Storage != nil {
			if perr := d.Storage.PutObject(r.Context(), storageKey, f, header.Size, ct); perr != nil {
				log.Printf("feed-upload-image: PutObject key=%s err=%v", storageKey, perr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store image.")
				return
			}
		} else {
			root := strings.TrimSpace(cfg.CourseFilesRoot)
			if root == "" {
				root = "data/course-files"
			}
			p := coursefiles.BlobDiskPath(root, courseCode, storageKey)
			if werr := writeLocalFile(p, f); werr != nil {
				log.Printf("feed-upload-image: local write key=%s err=%v", storageKey, werr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store image.")
				return
			}
		}

		fileID, err := coursefiles.Create(r.Context(), d.Pool, *cid, viewer, storageKey, header.Filename, ct, header.Size)
		if err != nil {
			log.Printf("feed-upload-image: db insert err=%v", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record upload.")
			return
		}

		contentPath := fmt.Sprintf("/api/v1/courses/%s/course-files/%s/content", courseCode, fileID.String())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{
			ID:          fileID.String(),
			ContentPath: contentPath,
			MimeType:    ct,
			ByteSize:    header.Size,
		})
	}
}
