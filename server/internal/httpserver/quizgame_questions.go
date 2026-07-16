package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func questionJSON(q quizgame.Question) map[string]any {
	var opts any = []any{}
	if len(q.Options) > 0 {
		_ = json.Unmarshal(q.Options, &opts)
	}
	var corr any
	if len(q.CorrectAnswer) > 0 {
		_ = json.Unmarshal(q.CorrectAnswer, &corr)
	}
	return map[string]any{
		"id":               q.ID,
		"kitId":            q.KitID,
		"position":         q.Position,
		"questionType":     q.QuestionType,
		"prompt":           q.Prompt,
		"promptMediaRef":   q.PromptMediaRef,
		"promptMediaAlt":   q.PromptMediaAlt,
		"options":          opts,
		"correctAnswer":    corr,
		"timeLimitSeconds": q.TimeLimitSeconds,
		"pointsStyle":      q.PointsStyle,
		"answerShuffle":    q.AnswerShuffle,
		"explanation":      q.Explanation,
		"sourceQuestionId": q.SourceQuestionID,
		"version":          q.Version,
		"createdAt":        q.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":        q.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (d Deps) requireQuizKitWrite(w http.ResponseWriter, r *http.Request) (courseCode, kitID string, ok bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return "", "", false
	}
	if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
		return "", "", false
	}
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", "", false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return "", "", false
	}
	kitID = chi.URLParam(r, "kit_id")
	if kitID == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing kit id.")
		return "", "", false
	}
	return courseCode, kitID, true
}

func (d Deps) requireQuizKitRead(w http.ResponseWriter, r *http.Request) (courseCode, kitID string, ok bool) {
	courseCode, _, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", "", false
	}
	if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
		return "", "", false
	}
	kitID = chi.URLParam(r, "kit_id")
	if kitID == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing kit id.")
		return "", "", false
	}
	return courseCode, kitID, true
}

func writeQuestionValidationErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "quizgame:") {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
		return true
	}
	return false
}

// handleListQuizQuestions is GET .../live-quizzes/kits/{kit_id}/questions.
func (d Deps) handleListQuizQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitRead(w, r)
		if !ok {
			return
		}
		qs, err := quizgame.ListQuestions(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list questions.")
			return
		}
		if qs == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		out := make([]map[string]any, 0, len(qs))
		for _, q := range qs {
			out = append(out, questionJSON(q))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"questions": out})
	}
}

// handleCreateQuizQuestion is POST .../live-quizzes/kits/{kit_id}/questions.
func (d Deps) handleCreateQuizQuestion() http.HandlerFunc {
	type optionIn struct {
		ID        string  `json:"id"`
		Text      string  `json:"text"`
		MediaRef  *string `json:"mediaRef"`
		MediaAlt  *string `json:"mediaAlt"`
		IsCorrect bool    `json:"isCorrect"`
	}
	type reqBody struct {
		QuestionType     string          `json:"questionType"`
		Prompt           string          `json:"prompt"`
		PromptMediaRef   *string         `json:"promptMediaRef"`
		PromptMediaAlt   *string         `json:"promptMediaAlt"`
		Options          []optionIn      `json:"options"`
		CorrectAnswer    json.RawMessage `json:"correctAnswer"`
		TimeLimitSeconds int             `json:"timeLimitSeconds"`
		PointsStyle      string          `json:"pointsStyle"`
		AnswerShuffle    *bool           `json:"answerShuffle"`
		Explanation      *string         `json:"explanation"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		opts := make([]quizgame.Option, 0, len(in.Options))
		for _, o := range in.Options {
			opts = append(opts, quizgame.Option{
				ID: o.ID, Text: o.Text, MediaRef: o.MediaRef, MediaAlt: o.MediaAlt, IsCorrect: o.IsCorrect,
			})
		}
		created, err := quizgame.CreateQuestion(r.Context(), d.Pool, courseCode, kitID, quizgame.CreateQuestionInput{
			QuestionType:     in.QuestionType,
			Prompt:           in.Prompt,
			PromptMediaRef:   in.PromptMediaRef,
			PromptMediaAlt:   in.PromptMediaAlt,
			Options:          opts,
			CorrectAnswer:    in.CorrectAnswer,
			TimeLimitSeconds: in.TimeLimitSeconds,
			PointsStyle:      in.PointsStyle,
			AnswerShuffle:    in.AnswerShuffle,
			Explanation:      in.Explanation,
		})
		if writeQuestionValidationErr(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create question.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.question.created")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(questionJSON(*created))
	}
}

// handlePatchQuizQuestion is PATCH .../questions/{qid} with If-Match version.
func (d Deps) handlePatchQuizQuestion() http.HandlerFunc {
	type optionIn struct {
		ID        string  `json:"id"`
		Text      string  `json:"text"`
		MediaRef  *string `json:"mediaRef"`
		MediaAlt  *string `json:"mediaAlt"`
		IsCorrect bool    `json:"isCorrect"`
	}
	type reqBody struct {
		QuestionType     *string         `json:"questionType"`
		Prompt           *string         `json:"prompt"`
		PromptMediaRef   *string         `json:"promptMediaRef"`
		PromptMediaAlt   *string         `json:"promptMediaAlt"`
		Options          []optionIn      `json:"options"`
		CorrectAnswer    json.RawMessage `json:"correctAnswer"`
		TimeLimitSeconds *int            `json:"timeLimitSeconds"`
		PointsStyle      *string         `json:"pointsStyle"`
		AnswerShuffle    *bool           `json:"answerShuffle"`
		Explanation      *string         `json:"explanation"`
		ClearPromptMedia *bool           `json:"clearPromptMedia"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		qid := chi.URLParam(r, "qid")
		ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
		if ifMatch == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "If-Match header with question version is required.")
			return
		}
		ver, err := strconv.Atoi(strings.Trim(ifMatch, `"`))
		if err != nil || ver < 1 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "If-Match must be a positive version integer.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := quizgame.PatchQuestionInput{ExpectedVersion: ver}
		patch.QuestionType = in.QuestionType
		patch.Prompt = in.Prompt
		patch.TimeLimitSeconds = in.TimeLimitSeconds
		patch.PointsStyle = in.PointsStyle
		patch.AnswerShuffle = in.AnswerShuffle
		if in.ClearPromptMedia != nil && *in.ClearPromptMedia {
			var nilStr *string
			patch.PromptMediaRef = &nilStr
			patch.PromptMediaAlt = &nilStr
		} else {
			if in.PromptMediaRef != nil {
				v := in.PromptMediaRef
				patch.PromptMediaRef = &v
			}
			if in.PromptMediaAlt != nil {
				v := in.PromptMediaAlt
				patch.PromptMediaAlt = &v
			}
		}
		if in.Explanation != nil {
			v := in.Explanation
			patch.Explanation = &v
		}
		if in.Options != nil {
			opts := make([]quizgame.Option, 0, len(in.Options))
			for _, o := range in.Options {
				opts = append(opts, quizgame.Option{
					ID: o.ID, Text: o.Text, MediaRef: o.MediaRef, MediaAlt: o.MediaAlt, IsCorrect: o.IsCorrect,
				})
			}
			patch.Options = &opts
		}
		if len(in.CorrectAnswer) > 0 {
			raw := in.CorrectAnswer
			patch.CorrectAnswer = &raw
		}
		updated, err := quizgame.PatchQuestion(r.Context(), d.Pool, courseCode, kitID, qid, patch)
		if errors.Is(err, quizgame.ErrVersionConflict) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Question was modified elsewhere. Reload and try again.")
			return
		}
		if writeQuestionValidationErr(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update question.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(questionJSON(*updated))
	}
}

// handleDeleteQuizQuestion is DELETE .../questions/{qid}.
func (d Deps) handleDeleteQuizQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		qid := chi.URLParam(r, "qid")
		okDel, err := quizgame.DeleteQuestion(r.Context(), d.Pool, courseCode, kitID, qid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete question.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.question.deleted")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleDuplicateQuizQuestion is POST .../questions/{qid}/duplicate.
func (d Deps) handleDuplicateQuizQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		qid := chi.URLParam(r, "qid")
		created, err := quizgame.DuplicateQuestion(r.Context(), d.Pool, courseCode, kitID, qid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not duplicate question.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(questionJSON(*created))
	}
}

// handleReorderQuizQuestions is POST .../questions/reorder.
func (d Deps) handleReorderQuizQuestions() http.HandlerFunc {
	type reqBody struct {
		Items []quizgame.ReorderItem `json:"items"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		qs, err := quizgame.ReorderQuestions(r.Context(), d.Pool, courseCode, kitID, in.Items)
		if writeQuestionValidationErr(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not reorder questions.")
			return
		}
		if qs == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		out := make([]map[string]any, 0, len(qs))
		for _, q := range qs {
			out = append(out, questionJSON(q))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"questions": out})
	}
}

// handleImportBankQuizQuestions is POST .../questions/import-bank.
func (d Deps) handleImportBankQuizQuestions() http.HandlerFunc {
	type reqBody struct {
		QuestionIDs []string `json:"questionIds"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitWrite(w, r)
		if !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.QuestionIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "questionIds is required.")
			return
		}
		created, err := quizgame.ImportBankQuestions(r.Context(), d.Pool, courseCode, kitID, in.QuestionIDs)
		if writeQuestionValidationErr(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not import bank questions.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.question.bank_imported")
		out := make([]map[string]any, 0, len(created))
		for _, q := range created {
			out = append(out, questionJSON(q))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"questions": out})
	}
}

// handlePushQuizQuestionToBank is POST .../questions/{qid}/push-to-bank.
func (d Deps) handlePushQuizQuestionToBank() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, okAccess := d.requireCourseAccess(w, r)
		if !okAccess {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		qid := chi.URLParam(r, "qid")
		bankID, err := quizgame.PushQuestionToBank(r.Context(), d.Pool, courseCode, kitID, qid, viewer)
		if writeQuestionValidationErr(w, err) {
			return
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not push question to bank.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.question.pushed_to_bank")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"bankQuestionId": bankID})
	}
}

// handleValidateQuizKit is GET .../live-quizzes/kits/{kit_id}/validate.
func (d Deps) handleValidateQuizKit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, ok := d.requireQuizKitRead(w, r)
		if !ok {
			return
		}
		result, err := quizgame.ValidateKit(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not validate kit.")
			return
		}
		if result == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		if result.Issues == nil {
			result.Issues = []quizgame.ValidIssue{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// handleListImportableBankQuestions is GET .../questions/bank-candidates.
func (d Deps) handleListImportableBankQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireQuizKitRead(w, r)
		if !ok {
			return
		}
		q := r.URL.Query().Get("q")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := quizgame.ListBankCandidates(r.Context(), d.Pool, courseCode, q, limit)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list bank questions.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"questions": rows})
	}
}
