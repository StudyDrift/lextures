package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/notificationevents"
	badgerepo "github.com/lextures/lextures/server/internal/repos/badges"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	badgesvc "github.com/lextures/lextures/server/internal/service/badges"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) badgesFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCompetencyBadges {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Competency badges are not enabled.")
		return true
	}
	return false
}

func (d Deps) registerBadgesRoutes(r chi.Router) {
	// Instructor: definitions under course
	r.Get("/api/v1/courses/{courseId}/badge-definitions", d.handleListBadgeDefinitions())
	r.Post("/api/v1/courses/{courseId}/badge-definitions", d.handleCreateBadgeDefinition())
	r.Patch("/api/v1/badge-definitions/{id}", d.handlePatchBadgeDefinition())
	r.Delete("/api/v1/badge-definitions/{id}", d.handleDeleteBadgeDefinition())
	r.Post("/api/v1/badge-definitions/{id}/award", d.handleAwardBadge())
	r.Get("/api/v1/badge-definitions/{id}/candidates", d.handleBadgeCandidates())

	// Award lifecycle
	r.Post("/api/v1/badges/{awardedId}/revoke", d.handleRevokeBadge())
	r.Get("/api/v1/badges/{awardedId}/linkedin-params", d.handleBadgeLinkedInParams())
	r.Get("/api/v1/badges/{awardedId}/badge-export", d.handleCompetencyBadgeExportURL())
	r.Get("/api/v1/badges/{awardedId}/badge-export/download", d.handleCompetencyBadgeExportDownload())

	// Learner me
	r.Get("/api/v1/me/badges", d.handleListMyBadges())
	r.Patch("/api/v1/me/badges/{awardedId}", d.handlePatchMyBadge())
	r.Get("/api/v1/me/badge-profile", d.handleGetBadgeProfile())
	r.Patch("/api/v1/me/badge-profile", d.handlePatchBadgeProfile())
	r.Get("/api/v1/badge-handle-available", d.handleBadgeHandleAvailable())

	// Public
	r.Get("/api/v1/public/badges/{handle}", d.handlePublicBadgeList())
	r.Get("/api/v1/public/badges/{handle}/{badgeSlug}", d.handlePublicBadgeDetail())
	r.Get("/api/v1/badges/verify/{shareSlug}", d.handleVerifyBadge())
	r.Get("/achievements/badge/{definitionId}", d.handlePublicAchievement())
}

func (d Deps) requireCourseStaffByCourseID(w http.ResponseWriter, r *http.Request, courseID uuid.UUID) (userID uuid.UUID, ok bool) {
	userID, ok = d.meUserID(w, r)
	if !ok {
		return uuid.Nil, false
	}
	isStaff, err := enrollment.UserIsCourseStaffByID(r.Context(), d.Pool, courseID, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
		return uuid.Nil, false
	}
	if !isStaff {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return uuid.Nil, false
	}
	return userID, true
}

func (d Deps) resolveCourseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if id, err := uuid.Parse(raw); err == nil {
		return id, true
	}
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
		return uuid.Nil, false
	}
	id, err := badgerepo.CourseIDByCode(r.Context(), d.Pool, raw)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve course.")
		return uuid.Nil, false
	}
	if id == uuid.Nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return uuid.Nil, false
	}
	return id, true
}

func (d Deps) handleListBadgeDefinitions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		courseID, ok := d.resolveCourseID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireCourseStaffByCourseID(w, r, courseID); !ok {
			return
		}
		rows, err := badgerepo.ListDefinitionsByCourse(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list badge definitions.")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			out = append(out, definitionJSON(&rows[i]))
		}
		writeBadgeJSON(w, map[string]any{"definitions": out})
	}
}

func (d Deps) handleCreateBadgeDefinition() http.HandlerFunc {
	type body struct {
		OutcomeID         *string  `json:"outcomeId"`
		SubOutcomeID      *string  `json:"subOutcomeId"`
		Name              string   `json:"name"`
		Slug              string   `json:"slug"`
		Description       string   `json:"description"`
		CriteriaNarrative string   `json:"criteriaNarrative"`
		Tags              []string `json:"tags"`
		Alignment         any      `json:"alignment"`
		AutoAward         bool     `json:"autoAward"`
		ImageUploadRef    *string  `json:"imageUploadRef"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		courseID, ok := d.resolveCourseID(w, r)
		if !ok {
			return
		}
		userID, ok := d.requireCourseStaffByCourseID(w, r, courseID)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := badgerepo.CreateDefinitionInput{
			CourseID:          courseID,
			Slug:              req.Slug,
			Name:              req.Name,
			Description:       req.Description,
			CriteriaNarrative: req.CriteriaNarrative,
			Tags:              req.Tags,
			AutoAward:         req.AutoAward,
			CreatedBy:         userID,
			ImageKey:          req.ImageUploadRef,
		}
		if req.OutcomeID != nil && strings.TrimSpace(*req.OutcomeID) != "" {
			oid, err := uuid.Parse(strings.TrimSpace(*req.OutcomeID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcomeId.")
				return
			}
			in.OutcomeID = &oid
		}
		if req.SubOutcomeID != nil && strings.TrimSpace(*req.SubOutcomeID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*req.SubOutcomeID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid subOutcomeId.")
				return
			}
			in.SubOutcomeID = &sid
		}
		if req.Alignment != nil {
			b, _ := json.Marshal(req.Alignment)
			in.AlignmentJSON = b
		}
		def, err := badgesvc.CreateDefinition(r.Context(), d.Pool, d.effectiveConfig(), in)
		if err != nil {
			writeBadgeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeBadgeJSON(w, definitionJSON(def))
	}
}

func (d Deps) handlePatchBadgeDefinition() http.HandlerFunc {
	type body struct {
		Name              *string  `json:"name"`
		Slug              *string  `json:"slug"`
		Description       *string  `json:"description"`
		CriteriaNarrative *string  `json:"criteriaNarrative"`
		Tags              *[]string `json:"tags"`
		AutoAward         *bool    `json:"autoAward"`
		Alignment         any      `json:"alignment"`
		OutcomeID         *string  `json:"outcomeId"`
		SubOutcomeID      *string  `json:"subOutcomeId"`
		ImageUploadRef    *string  `json:"imageUploadRef"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		def, ok := d.loadDefinitionWithStaff(w, r)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := badgerepo.UpdateDefinitionInput{
			Name:              req.Name,
			Slug:              req.Slug,
			Description:       req.Description,
			CriteriaNarrative: req.CriteriaNarrative,
			Tags:              req.Tags,
			AutoAward:         req.AutoAward,
			ImageKey:          req.ImageUploadRef,
		}
		if req.Slug != nil {
			if err := badgesvc.ValidateSlugFormat(*req.Slug); err != nil {
				writeBadgeErr(w, err)
				return
			}
			taken, err := badgerepo.SlugExistsInCourse(r.Context(), d.Pool, def.CourseID, *req.Slug, &def.ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate slug.")
				return
			}
			if taken {
				writeBadgeErr(w, badgesvc.ErrSlugTaken)
				return
			}
		}
		if req.Alignment != nil {
			b, _ := json.Marshal(req.Alignment)
			in.AlignmentJSON = b
		}
		if req.OutcomeID != nil {
			if strings.TrimSpace(*req.OutcomeID) == "" {
				in.ClearOutcomeID = true
			} else {
				oid, err := uuid.Parse(strings.TrimSpace(*req.OutcomeID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcomeId.")
					return
				}
				in.OutcomeID = &oid
			}
		}
		if req.SubOutcomeID != nil {
			if strings.TrimSpace(*req.SubOutcomeID) == "" {
				in.ClearSubOutcomeID = true
			} else {
				sid, err := uuid.Parse(strings.TrimSpace(*req.SubOutcomeID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid subOutcomeId.")
					return
				}
				in.SubOutcomeID = &sid
			}
		}
		updated, err := badgerepo.UpdateDefinition(r.Context(), d.Pool, def.ID, in)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update definition.")
			return
		}
		writeBadgeJSON(w, definitionJSON(updated))
	}
}

func (d Deps) handleDeleteBadgeDefinition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		def, ok := d.loadDefinitionWithStaff(w, r)
		if !ok {
			return
		}
		if err := badgerepo.DeleteDefinition(r.Context(), d.Pool, def.ID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete definition.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAwardBadge() http.HandlerFunc {
	type body struct {
		RecipientIDs []string `json:"recipientIds"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		def, ok := d.loadDefinitionWithStaff(w, r)
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(req.RecipientIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "recipientIds is required.")
			return
		}
		ids := make([]uuid.UUID, 0, len(req.RecipientIDs))
		for _, s := range req.RecipientIDs {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid recipient id.")
				return
			}
			ids = append(ids, id)
		}
		awardedBy := userID
		results, err := badgesvc.Award(r.Context(), d.Pool, d.effectiveConfig(), badgesvc.AwardParams{
			DefinitionID: def.ID,
			RecipientIDs: ids,
			AwardedBy:    &awardedBy,
			AwardSource:  badgerepo.AwardSourceManual,
		})
		if err != nil {
			writeBadgeErr(w, err)
			return
		}
		awarded := make([]map[string]any, 0)
		skipped := make([]map[string]any, 0)
		for _, res := range results {
			if res.Skipped {
				skipped = append(skipped, map[string]any{
					"recipientId": res.RecipientID.String(),
					"reason":      res.Reason,
					"award":       awardSummaryJSON(res.Awarded, def),
				})
				continue
			}
			awarded = append(awarded, awardSummaryJSON(res.Awarded, def))
			if res.Awarded != nil {
				d.notifyBadgeAwarded(r, res.Awarded.RecipientID, def, res.Awarded)
			}
		}
		writeBadgeJSON(w, map[string]any{"awarded": awarded, "skipped": skipped})
	}
}

func (d Deps) handleBadgeCandidates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		def, ok := d.loadDefinitionWithStaff(w, r)
		if !ok {
			return
		}
		// List active student enrollments for the course.
		rows, err := d.Pool.Query(r.Context(), `
SELECT u.id, COALESCE(u.display_name, split_part(u.email, '@', 1)),
       EXISTS (
         SELECT 1 FROM badges.awarded_badges ab
         WHERE ab.definition_id = $2 AND ab.recipient_id = u.id AND ab.revoked = FALSE
       ) AS already_awarded
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_staff = FALSE
WHERE ce.course_id = $1 AND ce.active = TRUE
ORDER BY u.display_name NULLS LAST, u.email
LIMIT 500
`, def.CourseID, def.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load candidates.")
			return
		}
		defer rows.Close()
		type cand struct {
			UserID         string `json:"userId"`
			DisplayName    string `json:"displayName"`
			AlreadyAwarded bool   `json:"alreadyAwarded"`
			MasteryReached bool   `json:"masteryReached"`
		}
		out := make([]cand, 0)
		for rows.Next() {
			var id uuid.UUID
			var name string
			var already bool
			if err := rows.Scan(&id, &name, &already); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load candidates.")
				return
			}
			mastery := false
			if def.OutcomeID != nil {
				mastery, _ = badgerepo.MasteryReached(r.Context(), d.Pool, def.CourseID, id, *def.OutcomeID)
			}
			out = append(out, cand{
				UserID:         id.String(),
				DisplayName:    name,
				AlreadyAwarded: already,
				MasteryReached: mastery,
			})
		}
		writeBadgeJSON(w, map[string]any{"candidates": out})
	}
}

func (d Deps) handleRevokeBadge() http.HandlerFunc {
	type body struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		award, def, ok := d.loadAwardWithStaff(w, r)
		if !ok {
			return
		}
		_ = def
		var req body
		_ = json.NewDecoder(r.Body).Decode(&req)
		updated, err := badgerepo.RevokeAward(r.Context(), d.Pool, award.ID, req.Reason)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke badge.")
			return
		}
		writeBadgeJSON(w, awardSummaryJSON(updated, def))
	}
}

func (d Deps) handleListMyBadges() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if _, err := badgerepo.EnsureProfile(r.Context(), d.Pool, userID, ""); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to ensure badge profile.")
			return
		}
		rows, err := badgerepo.ListAwardsByRecipient(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badges.")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			def, _ := badgerepo.GetDefinitionByID(r.Context(), d.Pool, rows[i].DefinitionID)
			out = append(out, awardSummaryJSON(&rows[i], def))
		}
		writeBadgeJSON(w, map[string]any{"badges": out})
	}
}

func (d Deps) handlePatchMyBadge() http.HandlerFunc {
	type body struct {
		IsPublic *bool `json:"isPublic"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		awardID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "awardedId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid badge id.")
			return
		}
		award, err := badgerepo.GetAwardByID(r.Context(), d.Pool, awardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge.")
			return
		}
		if award == nil || award.RecipientID != userID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.IsPublic == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "isPublic is required.")
			return
		}
		if *req.IsPublic {
			// Per-badge public requires guardian consent for minors (same gate as page public).
			if minor, _ := badgerepo.UserIsMinor(r.Context(), d.Pool, userID); minor {
				okConsent, _ := badgerepo.HasActiveGuardianConsent(r.Context(), d.Pool, userID)
				if !okConsent {
					writeBadgeErr(w, badgesvc.ErrMinorNeedsConsent)
					return
				}
			}
		}
		updated, err := badgerepo.SetAwardPublic(r.Context(), d.Pool, award.ID, *req.IsPublic)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update badge.")
			return
		}
		def, _ := badgerepo.GetDefinitionByID(r.Context(), d.Pool, updated.DefinitionID)
		writeBadgeJSON(w, awardSummaryJSON(updated, def))
	}
}

func (d Deps) handleGetBadgeProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		profile, err := badgerepo.EnsureProfile(r.Context(), d.Pool, userID, "")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge profile.")
			return
		}
		writeBadgeJSON(w, badgeProfileJSON(profile, d.effectiveConfig().PublicWebOrigin))
	}
}

func (d Deps) handlePatchBadgeProfile() http.HandlerFunc {
	type body struct {
		Handle              *string `json:"handle"`
		PagePublic          *bool   `json:"pagePublic"`
		SearchIndexable     *bool   `json:"searchIndexable"`
		DisplayNameOverride *string `json:"displayNameOverride"`
		HideRealName        *bool   `json:"hideRealName"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.Handle != nil {
			if _, err := badgesvc.UpdateHandle(r.Context(), d.Pool, d.effectiveConfig(), userID, *req.Handle); err != nil {
				writeBadgeErr(w, err)
				return
			}
		}
		if req.PagePublic != nil {
			if _, err := badgesvc.SetPagePublic(r.Context(), d.Pool, d.effectiveConfig(), userID, *req.PagePublic); err != nil {
				writeBadgeErr(w, err)
				return
			}
		}
		in := badgerepo.UpdateProfileInput{
			SearchIndexable:     req.SearchIndexable,
			HideRealName:        req.HideRealName,
			DisplayNameOverride: req.DisplayNameOverride,
		}
		if req.DisplayNameOverride != nil && strings.TrimSpace(*req.DisplayNameOverride) == "" {
			in.ClearDisplayName = true
			in.DisplayNameOverride = nil
		}
		if req.SearchIndexable != nil || req.HideRealName != nil || req.DisplayNameOverride != nil {
			if _, err := badgerepo.UpdateProfile(r.Context(), d.Pool, userID, in); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update profile.")
				return
			}
		}
		profile, err := badgerepo.GetProfile(r.Context(), d.Pool, userID)
		if err != nil || profile == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge profile.")
			return
		}
		writeBadgeJSON(w, badgeProfileJSON(profile, d.effectiveConfig().PublicWebOrigin))
	}
}

func (d Deps) handleBadgeHandleAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		handle := strings.TrimSpace(r.URL.Query().Get("handle"))
		if err := badgesvc.ValidateHandleFormat(handle); err != nil {
			writeBadgeJSON(w, map[string]any{
				"handle":    handle,
				"available": false,
				"valid":     false,
				"reason":    err.Error(),
			})
			return
		}
		taken, err := badgerepo.IsHandleTaken(r.Context(), d.Pool, handle, &userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check handle.")
			return
		}
		writeBadgeJSON(w, map[string]any{
			"handle":    strings.ToLower(handle),
			"available": !taken,
			"valid":     true,
		})
	}
}

func (d Deps) handlePublicBadgeList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		handle := strings.TrimSpace(chi.URLParam(r, "handle"))
		meta, uid, err := badgesvc.ResolvePublicPage(r.Context(), d.Pool, d.effectiveConfig(), handle)
		if err != nil {
			writeBadgeErr(w, err)
			return
		}
		if meta.RedirectTo != "" {
			w.Header().Set("Location", "/api/v1/public/badges/"+meta.RedirectTo)
			// Also expose current handle for SPA.
			writeBadgeJSON(w, map[string]any{
				"redirectTo": meta.RedirectTo,
				"handle":     meta.RedirectTo,
			})
			return
		}
		if !meta.PagePublic {
			writeBadgeJSON(w, map[string]any{
				"handle":      meta.Handle,
				"displayName": meta.DisplayName,
				"pagePublic":  false,
				"badges":      []any{},
				"status":      "private",
			})
			return
		}
		_ = badgerepo.IncrementPageView(r.Context(), d.Pool, uid, nil)
		awards, err := badgerepo.ListPublicAwardsForUser(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load public badges.")
			return
		}
		cfg := d.effectiveConfig()
		items := make([]map[string]any, 0, len(awards))
		for _, a := range awards {
			items = append(items, publicAwardJSON(&a, cfg.PublicWebOrigin))
		}
		writeBadgeJSON(w, map[string]any{
			"handle":          meta.Handle,
			"displayName":     meta.DisplayName,
			"pagePublic":      true,
			"searchIndexable": meta.SearchIndexable,
			"badges":          items,
			"status":          "ok",
		})
	}
}

func (d Deps) handlePublicBadgeDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		handle := strings.TrimSpace(chi.URLParam(r, "handle"))
		slug := strings.TrimSpace(chi.URLParam(r, "badgeSlug"))
		meta, uid, err := badgesvc.ResolvePublicPage(r.Context(), d.Pool, d.effectiveConfig(), handle)
		if err != nil {
			writeBadgeErr(w, err)
			return
		}
		if meta.RedirectTo != "" {
			writeBadgeJSON(w, map[string]any{"redirectTo": meta.RedirectTo, "handle": meta.RedirectTo})
			return
		}
		if !meta.PagePublic {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "This badge page is private.")
			return
		}
		pub, award, err := badgerepo.GetPublicAwardByHandleAndSlug(r.Context(), d.Pool, uid, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge.")
			return
		}
		if pub == nil || award == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
			return
		}
		_ = badgerepo.IncrementPageView(r.Context(), d.Pool, uid, &award.ID)
		cfg := d.effectiveConfig()
		item := publicAwardJSON(pub, cfg.PublicWebOrigin)
		item["recipientDisplayName"] = meta.DisplayName
		item["issuerName"] = issuerNameFromConfig(cfg)
		item["criteriaNarrative"] = pub.CriteriaNarrative
		item["searchIndexable"] = meta.SearchIndexable
		writeBadgeJSON(w, item)
	}
}

func (d Deps) handleVerifyBadge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		shareSlug := strings.TrimSpace(chi.URLParam(r, "shareSlug"))
		result, err := badgesvc.VerifyShareSlug(r.Context(), d.Pool, d.effectiveConfig(), shareSlug)
		if err != nil {
			writeBadgeErr(w, err)
			return
		}
		writeBadgeJSON(w, result)
	}
}

func (d Deps) handlePublicAchievement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "definitionId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid definition id.")
			return
		}
		def, err := badgerepo.GetDefinitionByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load achievement.")
			return
		}
		if def == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Achievement not found.")
			return
		}
		w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(badgesvc.AchievementJSON(d.effectiveConfig(), def))
	}
}

func (d Deps) handleBadgeLinkedInParams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		award, def, ok := d.loadOwnedAward(w, r, userID)
		if !ok {
			return
		}
		params := badgesvc.LinkedInParamsForAward(d.effectiveConfig(), def, award)
		writeBadgeJSON(w, params)
	}
}

func (d Deps) handleCompetencyBadgeExportURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		award, _, ok := d.loadOwnedAward(w, r, userID)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		token, expires, err := badgesvc.BadgeExportToken(cfg, award.ID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create download URL.")
			return
		}
		base := strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/")
		downloadURL := fmt.Sprintf("%s/api/v1/badges/%s/badge-export/download?token=%s", base, award.ID, token)
		writeBadgeJSON(w, map[string]any{
			"downloadUrl": downloadURL,
			"expiresAt":   expires.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleCompetencyBadgeExportDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.badgesFeatureOff(w) {
			return
		}
		awardID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "awardedId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid badge id.")
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "token is required.")
			return
		}
		cfg := d.effectiveConfig()
		parsedID, err := badgesvc.VerifyBadgeExportToken(cfg, token, time.Now().UTC())
		if err != nil || parsedID != awardID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Invalid or expired download token.")
			return
		}
		award, err := badgerepo.GetAwardByID(r.Context(), d.Pool, awardID)
		if err != nil || award == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
			return
		}
		def, _ := badgerepo.GetDefinitionByID(r.Context(), d.Pool, award.DefinitionID)
		name := "badge"
		if def != nil {
			name = def.Name
		}
		w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-badge.json"`, sanitizeFilename(name)))
		_, _ = w.Write(award.Proof)
	}
}

func (d Deps) loadDefinitionWithStaff(w http.ResponseWriter, r *http.Request) (*badgerepo.Definition, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid definition id.")
		return nil, false
	}
	def, err := badgerepo.GetDefinitionByID(r.Context(), d.Pool, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load definition.")
		return nil, false
	}
	if def == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Definition not found.")
		return nil, false
	}
	if _, ok := d.requireCourseStaffByCourseID(w, r, def.CourseID); !ok {
		return nil, false
	}
	return def, true
}

func (d Deps) loadAwardWithStaff(w http.ResponseWriter, r *http.Request) (*badgerepo.AwardedBadge, *badgerepo.Definition, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "awardedId")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid badge id.")
		return nil, nil, false
	}
	award, err := badgerepo.GetAwardByID(r.Context(), d.Pool, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge.")
		return nil, nil, false
	}
	if award == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
		return nil, nil, false
	}
	def, err := badgerepo.GetDefinitionByID(r.Context(), d.Pool, award.DefinitionID)
	if err != nil || def == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
		return nil, nil, false
	}
	if _, ok := d.requireCourseStaffByCourseID(w, r, def.CourseID); !ok {
		return nil, nil, false
	}
	return award, def, true
}

func (d Deps) loadOwnedAward(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (*badgerepo.AwardedBadge, *badgerepo.Definition, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "awardedId")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid badge id.")
		return nil, nil, false
	}
	award, err := badgerepo.GetAwardByID(r.Context(), d.Pool, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load badge.")
		return nil, nil, false
	}
	if award == nil || award.RecipientID != userID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
		return nil, nil, false
	}
	def, err := badgerepo.GetDefinitionByID(r.Context(), d.Pool, award.DefinitionID)
	if err != nil || def == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Badge not found.")
		return nil, nil, false
	}
	return award, def, true
}

func (d Deps) notifyBadgeAwarded(r *http.Request, userID uuid.UUID, def *badgerepo.Definition, award *badgerepo.AwardedBadge) {
	cfg := d.effectiveConfig()
	if !cfg.EmailNotificationsEnabled || d.Pool == nil || def == nil || award == nil {
		return
	}
	base := strings.TrimRight(cfg.PublicWebOrigin, "/")
	verifyURL := fmt.Sprintf("%s/api/v1/badges/verify/%s", base, award.ShareSlug)
	svc := notifications.Service{Pool: d.Pool, Config: cfg}
	_ = svc.EnqueueEmail(r.Context(), userID, notificationevents.CertificateIssued, "badge_awarded", map[string]string{
		"badgeName":  def.Name,
		"verifyUrl":  verifyURL,
		"badgesUrl":  base + "/me/badges",
	}, nil)
}

func definitionJSON(d *badgerepo.Definition) map[string]any {
	if d == nil {
		return nil
	}
	m := map[string]any{
		"id":                d.ID.String(),
		"courseId":          d.CourseID.String(),
		"slug":              d.Slug,
		"name":              d.Name,
		"description":       d.Description,
		"criteriaNarrative": d.CriteriaNarrative,
		"tags":              d.Tags,
		"autoAward":         d.AutoAward,
		"createdBy":         d.CreatedBy.String(),
		"createdAt":         d.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":         d.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if d.OutcomeID != nil {
		m["outcomeId"] = d.OutcomeID.String()
	}
	if d.SubOutcomeID != nil {
		m["subOutcomeId"] = d.SubOutcomeID.String()
	}
	if d.ImageKey != nil {
		m["imageKey"] = *d.ImageKey
	}
	if len(d.AlignmentJSON) > 0 {
		m["alignment"] = json.RawMessage(d.AlignmentJSON)
	}
	return m
}

func awardSummaryJSON(a *badgerepo.AwardedBadge, def *badgerepo.Definition) map[string]any {
	if a == nil {
		return nil
	}
	m := map[string]any{
		"id":           a.ID.String(),
		"definitionId": a.DefinitionID.String(),
		"recipientId":  a.RecipientID.String(),
		"awardSource":  string(a.AwardSource),
		"shareSlug":    a.ShareSlug,
		"isPublic":     a.IsPublic,
		"revoked":      a.Revoked,
		"issuedAt":     a.IssuedAt.UTC().Format(time.RFC3339),
	}
	if def != nil {
		m["name"] = def.Name
		m["slug"] = def.Slug
		m["description"] = def.Description
		m["criteriaNarrative"] = def.CriteriaNarrative
		m["courseId"] = def.CourseID.String()
		if def.ImageKey != nil {
			m["imageKey"] = *def.ImageKey
		}
	}
	if a.AwardedBy != nil {
		m["awardedBy"] = a.AwardedBy.String()
	}
	if a.RevokedReason != nil {
		m["revokedReason"] = *a.RevokedReason
	}
	if a.RevokedAt != nil {
		m["revokedAt"] = a.RevokedAt.UTC().Format(time.RFC3339)
	}
	return m
}

func publicAwardJSON(a *badgerepo.PublicAward, origin string) map[string]any {
	base := strings.TrimRight(origin, "/")
	m := map[string]any{
		"id":          a.AwardedID.String(),
		"slug":        a.Slug,
		"name":        a.Name,
		"description": a.Description,
		"tags":        a.Tags,
		"issuedAt":    a.IssuedAt.UTC().Format(time.RFC3339),
		"shareSlug":   a.ShareSlug,
		"verifyUrl":   base + "/api/v1/badges/verify/" + a.ShareSlug,
		"courseTitle": a.CourseTitle,
	}
	if a.ImageKey != nil {
		m["imageKey"] = *a.ImageKey
	}
	return m
}

func badgeProfileJSON(p *badgerepo.BadgeProfile, origin string) map[string]any {
	if p == nil {
		return nil
	}
	base := strings.TrimRight(origin, "/")
	handle := ""
	if p.Handle != nil {
		handle = *p.Handle
	}
	m := map[string]any{
		"handle":          handle,
		"pagePublic":      p.PagePublic,
		"searchIndexable": p.SearchIndexable,
		"hideRealName":    p.HideRealName,
		"publicUrl":       base + "/badges/" + handle,
		"handleChangeCount30d": p.HandleChangeCount30d,
	}
	if p.DisplayNameOverride != nil {
		m["displayNameOverride"] = *p.DisplayNameOverride
	}
	return m
}

func writeBadgeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeBadgeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, badgesvc.ErrFeatureDisabled):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Competency badges are not enabled.")
	case errors.Is(err, badgesvc.ErrNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
	case errors.Is(err, badgesvc.ErrForbidden):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
	case errors.Is(err, badgesvc.ErrInvalidHandle):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Handle must be 3–32 characters: lowercase letters, digits, and hyphens (no leading/trailing hyphen).")
	case errors.Is(err, badgesvc.ErrHandleReserved):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "That handle is reserved.")
	case errors.Is(err, badgesvc.ErrHandleTaken):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "That handle is already taken.")
	case errors.Is(err, badgesvc.ErrHandleRateLimited):
		apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Handle can be changed at most 5 times per 30 days.")
	case errors.Is(err, badgesvc.ErrInvalidSlug):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid slug.")
	case errors.Is(err, badgesvc.ErrSlugTaken):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Slug already used in this course.")
	case errors.Is(err, badgesvc.ErrMinorNeedsConsent):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Guardian consent is required before making your badge page public.")
	case errors.Is(err, badgesvc.ErrInvalidInput):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Request failed.")
	}
}
