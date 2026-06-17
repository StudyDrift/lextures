package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/communication"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

func (d Deps) handleEnrollmentInvitationApprove() http.HandlerFunc {
	return d.handleEnrollmentInvitationDecision(true)
}

func (d Deps) handleEnrollmentInvitationDecline() http.HandlerFunc {
	return d.handleEnrollmentInvitationDecision(false)
}

func (d Deps) handleEnrollmentInvitationDecision(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
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
		eid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		ctx := r.Context()
		tx, err := d.Pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start transaction.")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var courseID uuid.UUID
		var resolvedCourseCode string
		if approve {
			courseID, resolvedCourseCode, err = enrollment.ApproveInvitation(ctx, tx, eid, viewer)
		} else {
			courseID, resolvedCourseCode, err = enrollment.DeclineInvitation(ctx, tx, eid, viewer)
		}
		if errors.Is(err, enrollment.ErrInvitationNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment invitation not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update invitation.")
			return
		}
		if !strings.EqualFold(resolvedCourseCode, courseCode) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment invitation not found.")
			return
		}
		if approve {
			if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, viewer, courseID, courseCode); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to sync course permissions.")
				return
			}
		} else {
			if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, viewer, courseID, courseCode); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to sync course permissions.")
				return
			}
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save invitation response.")
			return
		}

		resolved := "declined"
		if approve {
			resolved = "approved"
		}
		_ = communication.ResolveEnrollmentInvitationMessages(ctx, d.Pool, viewer, eid, resolved)

		d.notifyMailbox(viewer)
		d.notifyCourses(viewer)
		d.notifyEnrollmentsForCourse(ctx, courseCode)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}