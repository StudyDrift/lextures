package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	qbmodels "github.com/lextures/lextures/server/internal/models/questionbank"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
)

// handleCreateCourseBankQuestion is POST /api/v1/courses/{course_code}/questions
func (d Deps) handleCreateCourseBankQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		_, courseID, viewer, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		var req qbmodels.CreateQuestionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		points := 1.0
		if req.Points != nil {
			points = *req.Points
		}
		status := "draft"
		if req.Status != nil {
			status = strings.TrimSpace(*req.Status)
		}
		shared := false
		if req.Shared != nil {
			shared = *req.Shared
		}
		srs := false
		if req.SrsEligible != nil {
			srs = *req.SrsEligible
		}
		in := questionbank.QuestionWrite{
			QuestionType:           strings.TrimSpace(req.QuestionType),
			Stem:                   req.Stem,
			Options:                req.Options,
			CorrectAnswer:          req.CorrectAnswer,
			Explanation:            trimmedExplanation(req.Explanation),
			Points:                 points,
			Status:                 status,
			Shared:                 shared,
			Metadata:               req.Metadata,
			ShuffleChoicesOverride: req.ShuffleChoicesOverride,
			SRSEligible:            srs,
		}
		if err := questionbank.ValidateQuestionWrite(in, true); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, questionWriteClientMessage(err))
			return
		}
		createdBy := viewer
		row, err := questionbank.CreateAuthoredQuestion(r.Context(), d.Pool, courseID, &createdBy, in)
		if err != nil {
			if msg := questionWriteClientMessage(err); msg != "Could not save question." {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save question.")
			return
		}
		writeJSON(w, http.StatusOK, questionEntityToAPI(*row, true))
	}
}

// handleUpdateCourseBankQuestion is PUT /api/v1/courses/{course_code}/questions/{question_id}
func (d Deps) handleUpdateCourseBankQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		_, courseID, viewer, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		qid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "question_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}
		cur, err := questionbank.GetQuestionForCourse(r.Context(), d.Pool, courseID, qid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load question.")
			return
		}
		if cur == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		var body struct {
			QuestionType           *string         `json:"questionType"`
			Stem                   *string         `json:"stem"`
			Options                json.RawMessage `json:"options"`
			CorrectAnswer          json.RawMessage `json:"correctAnswer"`
			Explanation            json.RawMessage `json:"explanation"`
			Points                 *float64        `json:"points"`
			Status                 *string         `json:"status"`
			Shared                 *bool           `json:"shared"`
			Metadata               json.RawMessage `json:"metadata"`
			ChangeNote             *string         `json:"changeNote"`
			ShuffleChoicesOverride json.RawMessage `json:"shuffleChoicesOverride"`
			SrsEligible            *bool           `json:"srsEligible"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := questionbank.QuestionWrite{
			QuestionType:           cur.QuestionType,
			Stem:                   cur.Stem,
			Options:                cur.Options,
			CorrectAnswer:          cur.CorrectAnswer,
			Explanation:            cur.Explanation,
			Points:                 cur.Points,
			Status:                 cur.Status,
			Shared:                 cur.Shared,
			Metadata:               cur.Metadata,
			ShuffleChoicesOverride: cur.ShuffleChoicesOverride,
			SRSEligible:            cur.SRSEligible,
			ChangeNote:             body.ChangeNote,
		}
		if body.QuestionType != nil {
			in.QuestionType = strings.TrimSpace(*body.QuestionType)
		}
		if body.Stem != nil {
			in.Stem = *body.Stem
		}
		if len(body.Options) > 0 {
			if string(body.Options) == "null" {
				in.Options = nil
			} else {
				in.Options = body.Options
			}
		}
		if len(body.CorrectAnswer) > 0 {
			if string(body.CorrectAnswer) == "null" {
				in.CorrectAnswer = nil
			} else {
				in.CorrectAnswer = body.CorrectAnswer
			}
		}
		if len(body.Explanation) > 0 {
			if string(body.Explanation) == "null" {
				in.Explanation = nil
			} else {
				var s string
				if err := json.Unmarshal(body.Explanation, &s); err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid explanation.")
					return
				}
				in.Explanation = trimmedExplanation(&s)
			}
		}
		if body.Points != nil {
			in.Points = *body.Points
		}
		if body.Status != nil {
			in.Status = strings.TrimSpace(*body.Status)
		}
		if body.Shared != nil {
			in.Shared = *body.Shared
		}
		if len(body.Metadata) > 0 && string(body.Metadata) != "null" {
			in.Metadata = body.Metadata
		}
		if len(body.ShuffleChoicesOverride) > 0 {
			if string(body.ShuffleChoicesOverride) == "null" {
				in.ShuffleChoicesOverride = nil
			} else {
				var b bool
				if err := json.Unmarshal(body.ShuffleChoicesOverride, &b); err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid shuffleChoicesOverride.")
					return
				}
				in.ShuffleChoicesOverride = &b
			}
		}
		if body.SrsEligible != nil {
			in.SRSEligible = *body.SrsEligible
		}
		if err := questionbank.ValidateQuestionWrite(in, true); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, questionWriteClientMessage(err))
			return
		}
		editor := viewer
		row, err := questionbank.UpdateAuthoredQuestion(r.Context(), d.Pool, courseID, qid, &editor, in)
		if err != nil {
			if msg := questionWriteClientMessage(err); msg != "Could not save question." {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save question.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		writeJSON(w, http.StatusOK, questionEntityToAPI(*row, true))
	}
}

// handleListCourseBankQuestionVersions is GET .../questions/{question_id}/versions
func (d Deps) handleListCourseBankQuestionVersions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		_, courseID, _, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		qid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "question_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}
		rows, err := questionbank.ListQuestionVersions(r.Context(), d.Pool, courseID, qid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load version history.")
			return
		}
		out := make([]qbmodels.QuestionVersionSummaryResponse, 0, len(rows))
		for _, row := range rows {
			out = append(out, qbmodels.QuestionVersionSummaryResponse{
				VersionNumber: row.VersionNumber,
				ChangeNote:    row.ChangeNote,
				ChangeSummary: row.ChangeSummary,
				CreatedBy:     row.CreatedBy,
				CreatedAt:     row.CreatedAt.UTC(),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"versions": out})
	}
}

// handleRestoreCourseBankQuestionVersion is POST .../versions/{version_number}/restore
func (d Deps) handleRestoreCourseBankQuestionVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		_, courseID, viewer, ok := d.requireQuestionBankStaff(w, r)
		if !ok {
			return
		}
		qid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "question_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}
		version, err := strconv.Atoi(strings.TrimSpace(chi.URLParam(r, "version_number")))
		if err != nil || version < 1 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid version.")
			return
		}
		var body qbmodels.RestoreQuestionVersionRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		editor := viewer
		row, err := questionbank.RestoreQuestionVersion(r.Context(), d.Pool, courseID, qid, int32(version), &editor, body.ChangeNote)
		if err != nil {
			if msg := questionWriteClientMessage(err); msg != "Could not save question." {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not restore version.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Version not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"newVersionNumber": row.VersionNumber})
	}
}

func trimmedExplanation(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func questionWriteClientMessage(err error) string {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case "stem is required":
		return "Stem is required."
	case "invalid question type":
		return "Invalid question type."
	case "invalid status":
		return "Invalid status."
	case "points must be zero or greater":
		return "Points must be a number greater than or equal to 0."
	case "invalid version":
		return "Invalid version."
	case "version snapshot is incomplete":
		return "That version cannot be restored."
	default:
		return "Could not save question."
	}
}
