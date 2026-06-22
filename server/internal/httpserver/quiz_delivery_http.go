package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/relativeschedule"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
)

func (d Deps) registerQuizDeliveryRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts", d.handleQuizAttemptsList())
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/start", d.handleQuizStart())
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/current-question", d.handleQuizCurrentQuestion())
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/focus-loss", d.handleQuizFocusLossPost())
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/focus-loss-events", d.handleQuizFocusLossEventsGet())
	d.registerQuizSubmitRoutes(r)
}

func (d Deps) ffAccommodationsAuditEnabled() bool {
	cfg := d.effectiveConfig()
	return cfg.FFAccommodationsEngine
}

func (d Deps) handleQuizStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		meta, err := course.GetCourseQuizMeta(ctx, d.Pool, *cid)
		if err != nil || meta == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		now := time.Now().UTC()
		visible, err := coursestructure.QuizVisibleToStudent(ctx, d.Pool, *cid, itemID, viewer, now)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check quiz access.")
			return
		}
		if !visible {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if !d.enforceConditionalReleaseForLearner(w, r, courseCode, *cid, viewer, itemID) {
			return
		}
		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		var body coursemodulequiz.QuizStartRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		if row.QuizAccessCode != nil && strings.TrimSpace(*row.QuizAccessCode) != "" {
			code := ""
			if body.QuizAccessCode != nil {
				code = strings.TrimSpace(*body.QuizAccessCode)
			}
			if code != strings.TrimSpace(*row.QuizAccessCode) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Invalid quiz access code.")
				return
			}
		}
		shift, err := relativeschedule.LoadForUser(ctx, d.Pool, *cid, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course schedule.")
			return
		}
		avFrom := shiftMaybe(shift, row.AvailableFrom)
		avUntil := shiftMaybe(shift, row.AvailableUntil)
		if avFrom != nil && now.Before(*avFrom) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This quiz is not yet available.")
			return
		}
		if avUntil != nil && now.After(*avUntil) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This quiz is no longer available.")
			return
		}
		eff := acsvc.ResolveEffectiveOrDefault(ctx, d.Pool, viewer, *cid)
		if inProg, err := quizattempts.FindInProgressAttempt(ctx, d.Pool, *cid, itemID, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load attempt.")
			return
		} else if inProg != nil {
			d.writeQuizStartResponse(w, row, meta, inProg, eff)
			return
		}
		submitted, err := quizattempts.CountSubmittedAttempts(ctx, d.Pool, *cid, itemID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to count attempts.")
			return
		}
		if !row.UnlimitedAttempts {
			max := row.MaxAttempts + eff.ExtraAttempts
			if max < 1 {
				max = 1
			}
			if int64(max) <= submitted {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No attempts remaining.")
				return
			}
		}
		var baseLimitSec int32
		if row.TimeLimitMinutes != nil && *row.TimeLimitMinutes > 0 {
			baseLimitSec = *row.TimeLimitMinutes * 60
		}
		effectiveLimit := acsvc.AppliedTimeLimit(baseLimitSec, eff.TimeMultiplier)
		extended := eff.TimeMultiplier > 1.000001
		var deadline *time.Time
		var effectiveLimitPtr *int32
		if effectiveLimit > 0 {
			d := now.Add(time.Duration(effectiveLimit) * time.Second)
			deadline = &d
			effectiveLimitPtr = &effectiveLimit
		}
		attemptNum := int32(submitted + 1)
		attempt, err := quizattempts.InsertAttempt(ctx, d.Pool, quizattempts.InsertAttemptParams{
			CourseID:                  *cid,
			StructureItemID:           itemID,
			StudentUserID:             viewer,
			AttemptNumber:             attemptNum,
			DeadlineAt:                deadline,
			EffectiveTimeLimitSeconds: effectiveLimitPtr,
			ExtendedTimeApplied:       extended,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start attempt.")
			return
		}
		if err := acsvc.AuditQuizStart(ctx, d.Pool, viewer, attempt.ID, d.ffAccommodationsAuditEnabled(), eff); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record accommodation audit.")
			return
		}
		d.writeQuizStartResponse(w, row, meta, attempt, eff)
	}
}

func (d Deps) writeQuizStartResponse(
	w http.ResponseWriter,
	row *coursemodulequizzes.CourseItemQuizRow,
	meta *course.CourseQuizMeta,
	attempt *quizattempts.QuizAttemptRow,
	eff acsvc.Effective,
) {
	lockdown := effectiveLockdownMode(meta.LockdownModeEnabled, row.LockdownMode)
	hintsDisabled := lockdown == "kiosk" && !eff.HintsAlwaysEnabled
	reduced := eff.ReducedDistraction
	var maxAttempts *int32
	var remaining *int32
	if !row.UnlimitedAttempts {
		max := row.MaxAttempts + eff.ExtraAttempts
		if max < 1 {
			max = 1
		}
		maxAttempts = &max
		submitted := attempt.AttemptNumber - 1
		if submitted < 0 {
			submitted = 0
		}
		rem := max - submitted
		if rem < 0 {
			rem = 0
		}
		remaining = &rem
	}
	out := coursemodulequiz.QuizStartResponse{
		AttemptID:                     attempt.ID,
		AttemptNumber:               attempt.AttemptNumber,
		StartedAt:                     attempt.StartedAt,
		LockdownMode:                  lockdown,
		HintsDisabled:                 hintsDisabled,
		BackNavigationAllowed:         row.AllowBackNavigation,
		CurrentQuestionIndex:          attempt.CurrentQuestionIndex,
		DeadlineAt:                    attempt.DeadlineAt,
		ReducedDistractionMode:        reduced,
		ExtendedTimeActive:            attempt.ExtendedTimeApplied || eff.TimeMultiplier > 1.000001,
		HintScaffoldingEnabled:        meta.HintScaffoldingEnabled,
		MisconceptionDetectionEnabled: meta.MisconceptionDetectionEnabled,
		RetakePolicy:                  row.GradeAttemptPolicy,
		MaxAttempts:                   maxAttempts,
		RemainingAttempts:             remaining,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (d Deps) handleQuizCurrentQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		attemptID, err := uuid.Parse(chi.URLParam(r, "attempt_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}
		ctx := r.Context()
		attempt, err := quizattempts.GetAttempt(ctx, d.Pool, attemptID)
		if err != nil || attempt == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.StudentUserID != viewer || attempt.StructureItemID != itemID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil || attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		meta, err := course.GetCourseQuizMeta(ctx, d.Pool, *cid)
		if err != nil || meta == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		eff := acsvc.ResolveEffectiveOrDefault(ctx, d.Pool, viewer, *cid)
		if d.ffAccommodationsAuditEnabled() {
			ctxItem := itemID
			_ = acsvc.AuditContentView(ctx, d.Pool, viewer, &ctxItem, true, eff)
		}
		attemptPtr := &attemptID
		questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
			ctx, d.Pool, *cid, itemID, meta.QuestionBankEnabled, row.Questions, attemptPtr, false,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		questions = coursemodulequiz.SanitizeQuizQuestionsForLearner(questions)
		total := uint(len(questions))
		idx := attempt.CurrentQuestionIndex
		if idx < 0 {
			idx = 0
		}
		completed := int(idx) >= len(questions)
		var q *coursemodulequiz.QuizQuestion
		if !completed && len(questions) > 0 {
			qq := questions[idx]
			q = &qq
		}
		out := coursemodulequiz.QuizCurrentQuestionResponse{
			Question:      q,
			QuestionIndex: idx,
			TotalQuestions: total,
			Completed:     completed,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) parseQuizAttemptForViewer(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	viewer uuid.UUID,
) (itemID uuid.UUID, attemptID uuid.UUID, attempt *quizattempts.QuizAttemptRow, ok bool) {
	var err error
	itemID, err = uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return itemID, attemptID, nil, false
	}
	attemptID, err = uuid.Parse(chi.URLParam(r, "attempt_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
		return itemID, attemptID, nil, false
	}
	ctx := r.Context()
	attempt, err = quizattempts.GetAttempt(ctx, d.Pool, attemptID)
	if err != nil || attempt == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
		return itemID, attemptID, nil, false
	}
	if attempt.StudentUserID != viewer || attempt.StructureItemID != itemID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
		return itemID, attemptID, nil, false
	}
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || cid == nil || attempt.CourseID != *cid {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
		return itemID, attemptID, nil, false
	}
	return itemID, attemptID, attempt, true
}

// handleQuizFocusLossPost is POST .../attempts/{attempt_id}/focus-loss — learner reports tab blur / visibility change.
func (d Deps) handleQuizFocusLossPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		_, attemptID, attempt, ok := d.parseQuizAttemptForViewer(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if attempt.Status != "in_progress" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Attempt is not in progress.")
			return
		}
		var body coursemodulequiz.QuizFocusLossRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		eventType := strings.TrimSpace(body.EventType)
		if eventType == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "eventType is required.")
			return
		}
		if len(eventType) > 64 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "eventType is too long.")
			return
		}
		if err := quizattempts.InsertFocusLossEvent(r.Context(), d.Pool, attemptID, eventType, body.DurationMS); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record focus event.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleQuizFocusLossEventsGet is GET .../attempts/{attempt_id}/focus-loss-events — instructor review.
func (d Deps) handleQuizFocusLossEventsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		attemptID, err := uuid.Parse(chi.URLParam(r, "attempt_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		can, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !can {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view focus-loss events.")
			return
		}
		ctx := r.Context()
		attempt, err := quizattempts.GetAttempt(ctx, d.Pool, attemptID)
		if err != nil || attempt == nil || attempt.StructureItemID != itemID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil || attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		events, total, err := quizattempts.ListFocusLossEvents(ctx, d.Pool, attemptID, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load focus-loss events.")
			return
		}
		outEvents := make([]coursemodulequiz.QuizFocusLossEventAPI, 0, len(events))
		for _, ev := range events {
			outEvents = append(outEvents, coursemodulequiz.QuizFocusLossEventAPI{
				ID:         ev.ID,
				EventType:  ev.EventType,
				DurationMS: ev.DurationMS,
				CreatedAt:  ev.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coursemodulequiz.QuizFocusLossEventsResponse{
			Events: outEvents,
			Total:  total,
		})
	}
}
