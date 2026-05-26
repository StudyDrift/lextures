package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/studyreflection"
	"github.com/lextures/lextures/server/internal/service/studyreflection"
)

func (d Deps) selfReflectionEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().SelfReflectionEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Self-reflection coaching is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerSelfReflectionRoutes(r chi.Router) {
	r.Get("/api/v1/me/study-stats", d.handleGetStudyStats())
	r.Get("/api/v1/me/study-goal", d.handleGetStudyGoal())
	r.Put("/api/v1/me/study-goal", d.handlePutStudyGoal())
	r.Get("/api/v1/me/reflection-journal", d.handleListReflectionJournal())
	r.Post("/api/v1/me/reflection-journal", d.handlePostReflectionJournal())
	r.Delete("/api/v1/me/reflection-journal/{id}", d.handleDeleteReflectionJournal())
	r.Get("/api/v1/me/coaching-tips", d.handleGetCoachingTips())
	r.Post("/api/v1/me/coaching-tips/{id}/rating", d.handleRateCoachingTip())
}

func (d Deps) handleGetStudyStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		stats, err := studyreflection.LoadStats(r.Context(), d.Pool, userID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load study stats.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(stats)
	}
}

type putStudyGoalBody struct {
	WeeklyHours *float32 `json:"weeklyHours"`
	OptedIn     *bool    `json:"optedIn"`
}

func (d Deps) handleGetStudyGoal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		goal, err := repo.GetGoal(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load study goal.")
			return
		}
		hours := float32(0)
		optedIn := false
		if goal != nil {
			hours = goal.WeeklyHours
			optedIn = goal.OptedIn
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"weeklyHours": hours,
			"optedIn":     optedIn,
		})
	}
}

func (d Deps) handlePutStudyGoal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putStudyGoalBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		goal, _ := repo.GetGoal(r.Context(), d.Pool, userID)
		hours := float32(0)
		optedIn := false
		if goal != nil {
			hours = goal.WeeklyHours
			optedIn = goal.OptedIn
		}
		if body.WeeklyHours != nil {
			if *body.WeeklyHours < 0 || *body.WeeklyHours > 168 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Weekly goal must be between 0 and 168 hours.")
				return
			}
			hours = *body.WeeklyHours
		}
		if body.OptedIn != nil {
			optedIn = *body.OptedIn
		}
		if err := repo.UpsertGoal(r.Context(), d.Pool, userID, hours, optedIn); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save study goal.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"weeklyHours": hours,
			"optedIn":     optedIn,
		})
	}
}

type journalEntryJSON struct {
	ID        string  `json:"id"`
	CourseID  *string `json:"courseId,omitempty"`
	EntryText string  `json:"entryText"`
	CreatedAt string  `json:"createdAt"`
}

func (d Deps) handleListReflectionJournal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		goal, _ := repo.GetGoal(r.Context(), d.Pool, userID)
		if goal != nil && !goal.OptedIn {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": []journalEntryJSON{}})
			return
		}
		rows, err := repo.ListJournal(r.Context(), d.Pool, userID, 50, 0)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load journal.")
			return
		}
		out := make([]journalEntryJSON, 0, len(rows))
		for _, e := range rows {
			item := journalEntryJSON{
				ID:        e.ID.String(),
				EntryText: e.EntryText,
				CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
			}
			if e.CourseID != nil {
				s := e.CourseID.String()
				item.CourseID = &s
			}
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}

type postJournalBody struct {
	EntryText string  `json:"entryText"`
	CourseID  *string `json:"courseId"`
}

func (d Deps) handlePostReflectionJournal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		goal, _ := repo.GetGoal(r.Context(), d.Pool, userID)
		if goal != nil && !goal.OptedIn {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Self-reflection is turned off.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postJournalBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(body.EntryText)
		if text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Entry text is required.")
			return
		}
		if utf8.RuneCountInString(text) > 280 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Entry must be 280 characters or fewer.")
			return
		}
		var courseID *uuid.UUID
		if body.CourseID != nil && strings.TrimSpace(*body.CourseID) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.CourseID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
				return
			}
			courseID = &id
		}
		id, err := repo.InsertJournal(r.Context(), d.Pool, userID, courseID, text)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save journal entry.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleDeleteReflectionJournal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		entryID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entry id.")
			return
		}
		deleted, err := repo.DeleteJournal(r.Context(), d.Pool, userID, entryID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete entry.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Entry not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

type coachingTipJSON struct {
	ID        string  `json:"id"`
	TipText   string  `json:"tipText"`
	WeekOf    string  `json:"weekOf"`
	Rating    *int16  `json:"rating,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

func (d Deps) handleGetCoachingTips() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		goal, _ := repo.GetGoal(r.Context(), d.Pool, userID)
		if goal != nil && !goal.OptedIn {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"latest": nil, "history": []coachingTipJSON{}})
			return
		}
		history, err := repo.ListCoachingTips(r.Context(), d.Pool, userID, 12)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load coaching tips.")
			return
		}
		items := make([]coachingTipJSON, 0, len(history))
		for _, t := range history {
			items = append(items, coachingTipToJSON(t))
		}
		var latest *coachingTipJSON
		if len(items) > 0 {
			latest = &items[0]
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"latest": latest, "history": items})
	}
}

func coachingTipToJSON(t repo.CoachingTip) coachingTipJSON {
	out := coachingTipJSON{
		ID:      t.ID.String(),
		TipText: t.TipText,
		WeekOf:  t.WeekOf.Format("2006-01-02"),
		Rating:  t.Rating,
	}
	if t.DeliveredAt != nil {
		out.CreatedAt = t.DeliveredAt.UTC().Format(time.RFC3339)
	} else {
		out.CreatedAt = t.WeekOf.Format(time.RFC3339)
	}
	return out
}

type rateTipBody struct {
	Rating int16 `json:"rating"`
}

func (d Deps) handleRateCoachingTip() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.selfReflectionEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tipID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid tip id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body rateTipBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Rating != -1 && body.Rating != 1 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Rating must be -1 or 1.")
			return
		}
		ok2, err := repo.RateCoachingTip(r.Context(), d.Pool, userID, tipID, body.Rating)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save rating.")
			return
		}
		if !ok2 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tip not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}
