package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/portfolios"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// requireEportfolioEnabled returns false and writes 501 when the feature flag is off.
func (d Deps) requireEportfolioEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFEportfolio {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "ePortfolio is not enabled.")
		return false
	}
	return true
}

// ─── JSON response types ──────────────────────────────────────────────────────

type portfolioJSON struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	IntroText  string   `json:"introText"`
	IsPublic   bool     `json:"isPublic"`
	PublicSlug *string  `json:"publicSlug"`
	Order      []string `json:"order"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}

func portfolioToJSON(p *portfolios.Portfolio) portfolioJSON {
	return portfolioJSON{
		ID:         p.ID.String(),
		Title:      p.Title,
		IntroText:  p.IntroText,
		IsPublic:   p.IsPublic,
		PublicSlug: p.PublicSlug,
		Order:      uuidsToStrings(p.SectionOrder),
		CreatedAt:  p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type artifactJSON struct {
	ID                 string   `json:"id"`
	PortfolioID        string   `json:"portfolioId"`
	ArtifactType       string   `json:"artifactType"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	SourceSubmissionID *string  `json:"sourceSubmissionId"`
	SourceCourseID     *string  `json:"sourceCourseId"`
	FileName           string   `json:"fileName"`
	FileMime           string   `json:"fileMime"`
	TextContent        string   `json:"textContent"`
	ExternalURL        string   `json:"externalUrl"`
	OutcomeIDs         []string `json:"outcomeIds"`
	IsPublic           bool     `json:"isPublic"`
	SortOrder          int      `json:"sortOrder"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
}

func artifactToJSON(a *portfolios.Artifact) artifactJSON {
	out := artifactJSON{
		ID:           a.ID.String(),
		PortfolioID:  a.PortfolioID.String(),
		ArtifactType: a.ArtifactType,
		Title:        a.Title,
		Description:  a.Description,
		FileName:     a.FileName,
		FileMime:     a.FileMime,
		TextContent:  a.TextContent,
		ExternalURL:  a.ExternalURL,
		OutcomeIDs:   uuidsToStrings(a.OutcomeIDs),
		IsPublic:     a.IsPublic,
		SortOrder:    a.SortOrder,
		CreatedAt:    a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    a.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if a.SourceSubmissionID != nil {
		s := a.SourceSubmissionID.String()
		out.SourceSubmissionID = &s
	}
	if a.SourceCourseID != nil {
		s := a.SourceCourseID.String()
		out.SourceCourseID = &s
	}
	return out
}

type evaluationJSON struct {
	ID         string          `json:"id"`
	ArtifactID string          `json:"artifactId"`
	ReviewerID string          `json:"reviewerId"`
	Reviewer   string          `json:"reviewer"`
	RubricJSON json.RawMessage `json:"rubric"`
	ScoresJSON json.RawMessage `json:"scores"`
	TotalScore *float64        `json:"totalScore"`
	Feedback   string          `json:"feedback"`
	UpdatedAt  string          `json:"updatedAt"`
}

func evaluationToJSON(e *portfolios.Evaluation, reviewerLabel string) evaluationJSON {
	rubric := e.RubricJSON
	if len(rubric) == 0 {
		rubric = json.RawMessage("null")
	}
	scores := e.ScoresJSON
	if len(scores) == 0 {
		scores = json.RawMessage("{}")
	}
	return evaluationJSON{
		ID:         e.ID.String(),
		ArtifactID: e.ArtifactID.String(),
		ReviewerID: e.ReviewerID.String(),
		Reviewer:   reviewerLabel,
		RubricJSON: rubric,
		ScoresJSON: scores,
		TotalScore: e.TotalScore,
		Feedback:   e.Feedback,
		UpdatedAt:  e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

// parseUUIDList parses a list of string ids, skipping blanks; returns an error on a malformed id.
func parseUUIDList(ss []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// ─── My portfolios (owner, authenticated) ──────────────────────────────────────

func (d Deps) handleListMyPortfolios() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		rows, err := portfolios.ListPortfoliosByOwner(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load portfolios.")
			return
		}
		out := make([]portfolioJSON, 0, len(rows))
		for i := range rows {
			out = append(out, portfolioToJSON(&rows[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"portfolios": out})
	}
}

func (d Deps) handleCreatePortfolio() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Title     string `json:"title"`
			IntroText string `json:"introText"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.Title) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
			return
		}
		p, err := portfolios.CreatePortfolio(r.Context(), d.Pool, uid, strings.TrimSpace(body.Title), strings.TrimSpace(body.IntroText))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create portfolio.")
			return
		}
		writeJSON(w, http.StatusCreated, portfolioToJSON(p))
	}
}

// portfolioDetailJSON bundles a portfolio with its artifacts and evaluations for the owner view.
type portfolioDetailJSON struct {
	Portfolio   portfolioJSON    `json:"portfolio"`
	Artifacts   []artifactJSON   `json:"artifacts"`
	Evaluations []evaluationJSON `json:"evaluations"`
}

func (d Deps) handleGetMyPortfolio() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pid, ok := parsePathUUID(w, r, "pid")
		if !ok {
			return
		}
		p, err := portfolios.GetPortfolioOwned(r.Context(), d.Pool, uid, pid)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "portfolio")
			return
		}
		arts, err := portfolios.ListArtifacts(r.Context(), d.Pool, pid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load artifacts.")
			return
		}
		evals, err := portfolios.ListEvaluationsForPortfolio(r.Context(), d.Pool, pid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluations.")
			return
		}
		writeJSON(w, http.StatusOK, portfolioDetailJSON{
			Portfolio:   portfolioToJSON(p),
			Artifacts:   artifactsToJSON(arts),
			Evaluations: d.evaluationsToJSON(r.Context(), evals),
		})
	}
}

func (d Deps) handlePatchMyPortfolio() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pid, ok := parsePathUUID(w, r, "pid")
		if !ok {
			return
		}
		var body struct {
			Title     *string   `json:"title"`
			IntroText *string   `json:"introText"`
			IsPublic  *bool     `json:"isPublic"`
			Order     *[]string `json:"order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		// Title / intro update.
		if body.Title != nil || body.IntroText != nil {
			cur, err := portfolios.GetPortfolioOwned(r.Context(), d.Pool, uid, pid)
			if err != nil {
				d.writePortfolioRepoErr(w, err, "portfolio")
				return
			}
			title := cur.Title
			if body.Title != nil {
				if strings.TrimSpace(*body.Title) == "" {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
					return
				}
				title = strings.TrimSpace(*body.Title)
			}
			intro := cur.IntroText
			if body.IntroText != nil {
				intro = strings.TrimSpace(*body.IntroText)
			}
			if _, err := portfolios.UpdatePortfolio(r.Context(), d.Pool, uid, pid, portfolios.UpdatePortfolioInput{Title: title, IntroText: intro}); err != nil {
				d.writePortfolioRepoErr(w, err, "portfolio")
				return
			}
		}
		// Reorder.
		if body.Order != nil {
			ids, err := parseUUIDList(*body.Order)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid artifact id in order.")
				return
			}
			if err := portfolios.ReorderArtifacts(r.Context(), d.Pool, uid, pid, ids); err != nil {
				d.writePortfolioRepoErr(w, err, "portfolio")
				return
			}
		}
		// Visibility toggle (last so the returned slug is current).
		if body.IsPublic != nil {
			if _, err := portfolios.SetPortfolioVisibility(r.Context(), d.Pool, uid, pid, *body.IsPublic); err != nil {
				d.writePortfolioRepoErr(w, err, "portfolio")
				return
			}
		}
		p, err := portfolios.GetPortfolioOwned(r.Context(), d.Pool, uid, pid)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "portfolio")
			return
		}
		writeJSON(w, http.StatusOK, portfolioToJSON(p))
	}
}

func (d Deps) handleDeleteMyPortfolio() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pid, ok := parsePathUUID(w, r, "pid")
		if !ok {
			return
		}
		if err := portfolios.DeletePortfolio(r.Context(), d.Pool, uid, pid); err != nil {
			d.writePortfolioRepoErr(w, err, "portfolio")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ─── Artifacts (owner) ─────────────────────────────────────────────────────────

func (d Deps) handleCreateArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pid, ok := parsePathUUID(w, r, "pid")
		if !ok {
			return
		}
		var body struct {
			ArtifactType string   `json:"artifactType"`
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			SubmissionID *string  `json:"sourceSubmissionId"`
			TextContent  string   `json:"textContent"`
			ExternalURL  string   `json:"externalUrl"`
			OutcomeIDs   []string `json:"outcomeIds"`
			IsPublic     bool     `json:"isPublic"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.Title) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
			return
		}
		switch body.ArtifactType {
		case "submission", "upload", "text_page", "url":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid artifactType.")
			return
		}
		outcomes, err := parseUUIDList(body.OutcomeIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcome id.")
			return
		}
		in := portfolios.CreateArtifactInput{
			ArtifactType: body.ArtifactType,
			Title:        strings.TrimSpace(body.Title),
			Description:  strings.TrimSpace(body.Description),
			TextContent:  body.TextContent,
			ExternalURL:  strings.TrimSpace(body.ExternalURL),
			OutcomeIDs:   outcomes,
			IsPublic:     body.IsPublic,
		}
		// For submission artifacts, snapshot the file from a submission the caller owns.
		if body.ArtifactType == "submission" {
			if body.SubmissionID == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sourceSubmissionId is required for submission artifacts.")
				return
			}
			subID, err := uuid.Parse(strings.TrimSpace(*body.SubmissionID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sourceSubmissionId.")
				return
			}
			sub, err := moduleassignmentsubmissions.GetByID(r.Context(), d.Pool, subID)
			if err != nil || sub == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
				return
			}
			if sub.SubmittedBy != uid {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You can only add your own submissions.")
				return
			}
			in.SourceSubmissionID = &sub.ID
			in.SourceCourseID = &sub.CourseID
			if sub.AttachmentFileID != nil {
				if f, ferr := getCourseFileByID(r.Context(), d.Pool, *sub.AttachmentFileID); ferr == nil && f != nil {
					in.FileKey = f.StorageKey
					in.FileName = f.OriginalFilename
					in.FileMime = f.MimeType
				}
			}
		}
		a, err := portfolios.CreateArtifact(r.Context(), d.Pool, uid, pid, in)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "portfolio")
			return
		}
		writeJSON(w, http.StatusCreated, artifactToJSON(a))
	}
}

func (d Deps) handlePatchArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		aid, ok := parsePathUUID(w, r, "aid")
		if !ok {
			return
		}
		cur, err := portfolios.GetArtifactOwned(r.Context(), d.Pool, uid, aid)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "artifact")
			return
		}
		var body struct {
			Title       *string   `json:"title"`
			Description *string   `json:"description"`
			TextContent *string   `json:"textContent"`
			ExternalURL *string   `json:"externalUrl"`
			OutcomeIDs  *[]string `json:"outcomeIds"`
			IsPublic    *bool     `json:"isPublic"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := portfolios.UpdateArtifactInput{
			Title:       cur.Title,
			Description: cur.Description,
			TextContent: cur.TextContent,
			ExternalURL: cur.ExternalURL,
			OutcomeIDs:  cur.OutcomeIDs,
			IsPublic:    cur.IsPublic,
		}
		if body.Title != nil {
			if strings.TrimSpace(*body.Title) == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
				return
			}
			in.Title = strings.TrimSpace(*body.Title)
		}
		if body.Description != nil {
			in.Description = strings.TrimSpace(*body.Description)
		}
		if body.TextContent != nil {
			in.TextContent = *body.TextContent
		}
		if body.ExternalURL != nil {
			in.ExternalURL = strings.TrimSpace(*body.ExternalURL)
		}
		if body.OutcomeIDs != nil {
			outcomes, err := parseUUIDList(*body.OutcomeIDs)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcome id.")
				return
			}
			in.OutcomeIDs = outcomes
		}
		if body.IsPublic != nil {
			in.IsPublic = *body.IsPublic
		}
		a, err := portfolios.UpdateArtifact(r.Context(), d.Pool, uid, aid, in)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "artifact")
			return
		}
		writeJSON(w, http.StatusOK, artifactToJSON(a))
	}
}

func (d Deps) handleDeleteArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		aid, ok := parsePathUUID(w, r, "aid")
		if !ok {
			return
		}
		if err := portfolios.DeleteArtifact(r.Context(), d.Pool, uid, aid); err != nil {
			d.writePortfolioRepoErr(w, err, "artifact")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ─── Public portfolio view (no auth) ───────────────────────────────────────────

type publicPortfolioJSON struct {
	Title     string         `json:"title"`
	IntroText string         `json:"introText"`
	OwnerName string         `json:"ownerName"`
	Artifacts []artifactJSON `json:"artifacts"`
	ViewCount int64          `json:"viewCount"`
}

func (d Deps) handleGetPublicPortfolio() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		if slug == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Portfolio not found.")
			return
		}
		p, err := portfolios.GetPortfolioBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			if errors.Is(err, portfolios.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Portfolio not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load portfolio.")
			return
		}
		arts, err := portfolios.ListPublicArtifacts(r.Context(), d.Pool, p.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load portfolio.")
			return
		}
		// Record a privacy-safe view (best-effort; no PII).
		_ = portfolios.RecordView(r.Context(), d.Pool, p.ID)
		count, _ := portfolios.ViewCount(r.Context(), d.Pool, p.ID)

		ownerName := ""
		if labels, lerr := user.DisplayLabelsByIDs(r.Context(), d.Pool, []uuid.UUID{p.OwnerID}); lerr == nil {
			ownerName = labels[p.OwnerID]
		}
		writeJSON(w, http.StatusOK, publicPortfolioJSON{
			Title:     p.Title,
			IntroText: p.IntroText,
			OwnerName: ownerName,
			Artifacts: artifactsToJSON(arts),
			ViewCount: count,
		})
	}
}

// ─── Rubric evaluation (reviewer) ──────────────────────────────────────────────

func (d Deps) handleEvaluateArtifact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		aid, ok := parsePathUUID(w, r, "aid")
		if !ok {
			return
		}
		art, err := portfolios.GetArtifactForReview(r.Context(), d.Pool, aid)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "artifact")
			return
		}
		// Authorize: the reviewer must hold a gradebook permission in the artifact's source course.
		if !d.canReviewArtifact(w, r, uid, art) {
			return
		}
		var body struct {
			Rubric     json.RawMessage `json:"rubric"`
			Scores     json.RawMessage `json:"scores"`
			TotalScore *float64        `json:"totalScore"`
			Feedback   string          `json:"feedback"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		e, err := portfolios.UpsertEvaluation(r.Context(), d.Pool, aid, uid, portfolios.UpsertEvaluationInput{
			RubricJSON: body.Rubric,
			ScoresJSON: body.Scores,
			TotalScore: body.TotalScore,
			Feedback:   strings.TrimSpace(body.Feedback),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save evaluation.")
			return
		}
		label := ""
		if labels, lerr := user.DisplayLabelsByIDs(r.Context(), d.Pool, []uuid.UUID{uid}); lerr == nil {
			label = labels[uid]
		}
		writeJSON(w, http.StatusCreated, evaluationToJSON(e, label))
	}
}

// canReviewArtifact returns true when uid may evaluate art, writing the failure response otherwise.
func (d Deps) canReviewArtifact(w http.ResponseWriter, r *http.Request, uid uuid.UUID, art *portfolios.Artifact) bool {
	if art.SourceCourseID == nil {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This artifact has no associated course to authorize a reviewer.")
		return false
	}
	code, err := repoCourse.GetCourseCodeByID(r.Context(), d.Pool, *art.SourceCourseID)
	if err != nil || code == nil {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return false
	}
	hasPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, uid, "course:"+*code+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return false
	}
	return true
}

// ─── Outcome coverage report (program director) ────────────────────────────────

type outcomeCoverageJSON struct {
	OutcomeID      string  `json:"outcomeId"`
	Title          string  `json:"title"`
	StudentCount   int     `json:"studentCount"`
	ArtifactCount  int     `json:"artifactCount"`
	SubmissionRate float64 `json:"submissionRate"` // percentage of cohort with evidence
}

func (d Deps) handlePortfolioOutcomesReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		// prog_id is interpreted as the course whose learning outcomes define the program rubric.
		progID, ok := parsePathUUID(w, r, "prog_id")
		if !ok {
			return
		}
		rows, cohort, err := portfolios.OutcomeCoverageReport(r.Context(), d.Pool, progID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build report.")
			return
		}
		out := make([]outcomeCoverageJSON, 0, len(rows))
		for _, row := range rows {
			rate := 0.0
			if cohort > 0 {
				rate = float64(row.StudentCount) / float64(cohort) * 100
			}
			out = append(out, outcomeCoverageJSON{
				OutcomeID:      row.OutcomeID.String(),
				Title:          row.Title,
				StudentCount:   row.StudentCount,
				ArtifactCount:  row.ArtifactCount,
				SubmissionRate: rate,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"cohortSize": cohort, "outcomes": out})
	}
}

// ─── helpers ───────────────────────────────────────────────────────────────────

func artifactsToJSON(arts []portfolios.Artifact) []artifactJSON {
	out := make([]artifactJSON, 0, len(arts))
	for i := range arts {
		out = append(out, artifactToJSON(&arts[i]))
	}
	return out
}

// evaluationsToJSON resolves reviewer display labels for a set of evaluations.
func (d Deps) evaluationsToJSON(ctx context.Context, evals []portfolios.Evaluation) []evaluationJSON {
	ids := make([]uuid.UUID, 0, len(evals))
	for i := range evals {
		ids = append(ids, evals[i].ReviewerID)
	}
	labels, _ := user.DisplayLabelsByIDs(ctx, d.Pool, ids)
	out := make([]evaluationJSON, 0, len(evals))
	for i := range evals {
		out = append(out, evaluationToJSON(&evals[i], labels[evals[i].ReviewerID]))
	}
	return out
}

func (d Deps) writePortfolioRepoErr(w http.ResponseWriter, err error, noun string) {
	if errors.Is(err, portfolios.ErrNotFound) {
		title := noun
		if len(title) > 0 {
			title = strings.ToUpper(title[:1]) + title[1:]
		}
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, title+" not found.")
		return
	}
	apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load "+noun+".")
}

func parsePathUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid "+param+".")
		return uuid.UUID{}, false
	}
	return id, true
}

// getCourseFileByID looks up a course file's metadata by id (used to snapshot submission attachments).
func getCourseFileByID(ctx context.Context, pool *pgxpool.Pool, fileID uuid.UUID) (*coursefiles.Row, error) {
	var row coursefiles.Row
	err := pool.QueryRow(ctx, `
SELECT id, course_id, storage_key, original_filename, mime_type, byte_size
FROM course.course_files WHERE id = $1`, fileID).Scan(
		&row.ID, &row.CourseID, &row.StorageKey, &row.OriginalFilename, &row.MimeType, &row.ByteSize)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ─── Route registration ────────────────────────────────────────────────────────

func (d Deps) registerEportfolioRoutes(r chi.Router) {
	r.Get("/api/v1/me/portfolios", d.handleListMyPortfolios())
	r.Post("/api/v1/me/portfolios", d.handleCreatePortfolio())
	r.Get("/api/v1/me/portfolios/{pid}", d.handleGetMyPortfolio())
	r.Patch("/api/v1/me/portfolios/{pid}", d.handlePatchMyPortfolio())
	r.Delete("/api/v1/me/portfolios/{pid}", d.handleDeleteMyPortfolio())
	r.Post("/api/v1/me/portfolios/{pid}/artifacts", d.handleCreateArtifact())
	r.Patch("/api/v1/me/portfolios/{pid}/artifacts/{aid}", d.handlePatchArtifact())
	r.Delete("/api/v1/me/portfolios/{pid}/artifacts/{aid}", d.handleDeleteArtifact())
	// Public, unauthenticated read-only view.
	r.Get("/api/v1/portfolios/{slug}", d.handleGetPublicPortfolio())
	// Reviewer rubric evaluation.
	r.Post("/api/v1/portfolios/{pid}/artifacts/{aid}/evaluate", d.handleEvaluateArtifact())
	// Program director outcome coverage report.
	r.Get("/api/v1/admin/programs/{prog_id}/portfolio-outcomes-report", d.handlePortfolioOutcomesReport())
}
