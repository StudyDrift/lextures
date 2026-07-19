package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// iqModeAllowed: team / student-paced / homework are authoring options gated by the
// course Live Quizzes flag only (platform sub-flags collapsed; docs/completed/flags.md).
func (d Deps) iqModeAllowed(_ engine.SessionMode) bool {
	return true
}

func (d Deps) iqHomeworkFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	return d.interactiveQuizzesFeatureOff(w, r, courseCode)
}

func teamJSON(t quizgame.Team) map[string]any {
	out := map[string]any{
		"id":         t.ID,
		"sessionId":  t.SessionID,
		"name":       t.Name,
		"totalScore": t.TotalScore,
	}
	if t.Color != nil {
		out["color"] = *t.Color
	}
	return out
}

// handleCreateQuizGameTeams is POST .../games/{game_id}/teams
func (d Deps) handleCreateQuizGameTeams() http.HandlerFunc {
	type reqBody struct {
		Names  []string `json:"names"`
		Colors []string `json:"colors"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if !d.iqModeAllowed(engine.ModeTeam) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Team mode is not enabled.")
			return
		}
		gameID := chi.URLParam(r, "game_id")
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage teams.")
			return
		}
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		var body reqBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		teams, err := quizgame.CreateTeams(r.Context(), d.Pool, sess.ID, body.Names, body.Colors)
		if errors.Is(err, quizgame.ErrWrongMode) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Game is not in team mode.")
			return
		}
		if errors.Is(err, quizgame.ErrModeImmutable) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Teams cannot be recreated after players have joined.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create teams.")
			return
		}
		out := make([]map[string]any, 0, len(teams))
		for _, t := range teams {
			out = append(out, teamJSON(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"teams": out})
	}
}

// handleAssignQuizGameTeams is POST .../games/{game_id}/teams/assign
func (d Deps) handleAssignQuizGameTeams() http.HandlerFunc {
	type reqBody struct {
		Assignments map[string]string `json:"assignments"`
		AutoBalance bool              `json:"autoBalance"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if !d.iqModeAllowed(engine.ModeTeam) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Team mode is not enabled.")
			return
		}
		gameID := chi.URLParam(r, "game_id")
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to assign teams.")
			return
		}
		var body reqBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		err = quizgame.AssignPlayers(r.Context(), d.Pool, gameID, quizgame.AssignPlayersInput{
			Assignments: body.Assignments,
			AutoBalance: body.AutoBalance || len(body.Assignments) == 0,
		})
		if errors.Is(err, quizgame.ErrWrongMode) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Game is not in team mode.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		teams, _ := quizgame.ListTeams(r.Context(), d.Pool, gameID)
		players, _ := quizgame.ListPlayers(r.Context(), d.Pool, gameID)
		playerOut := make([]map[string]any, 0, len(players))
		for _, p := range players {
			m := map[string]any{"id": p.ID, "nickname": p.Nickname, "teamId": p.TeamID}
			playerOut = append(playerOut, m)
		}
		teamOut := make([]map[string]any, 0, len(teams))
		for _, t := range teams {
			teamOut = append(teamOut, teamJSON(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"teams": teamOut, "players": playerOut})
	}
}

// handleGetQuizGameTeams is GET .../games/{game_id}/teams
func (d Deps) handleGetQuizGameTeams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		teams, err := quizgame.ListTeams(r.Context(), d.Pool, gameID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list teams.")
			return
		}
		board, _ := quizgame.RefreshTeamScores(r.Context(), d.Pool, gameID)
		teamOut := make([]map[string]any, 0, len(teams))
		for _, t := range teams {
			teamOut = append(teamOut, teamJSON(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"teams":       teamOut,
			"leaderboard": board,
		})
	}
}

// handleStartPacedGame is POST .../games/{game_id}/paced/start
func (d Deps) handleStartPacedGame() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if !d.iqModeAllowed(engine.ModeStudentPaced) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Student-paced mode is not enabled.")
			return
		}
		gameID := chi.URLParam(r, "game_id")
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to start this game.")
			return
		}
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if sess == nil || errors.Is(err, quizgame.ErrSessionNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err := quizgame.StartPacedGameForAll(r.Context(), d.Pool, sess.ID, time.Now().UTC()); err != nil {
			if errors.Is(err, quizgame.ErrWrongMode) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Game is not student-paced.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not start paced game.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.paced.start")
		buckets, total, finished, _ := quizgame.PacedHostProgress(r.Context(), d.Pool, sess.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"progress":       buckets,
			"playerCount":    total,
			"finishedCount":  finished,
		})
	}
}

// handleGetPacedProgress is GET .../games/{game_id}/paced/progress
func (d Deps) handleGetPacedProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		buckets, total, finished, err := quizgame.PacedHostProgress(r.Context(), d.Pool, gameID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"progress":      buckets,
			"playerCount":   total,
			"finishedCount": finished,
		})
	}
}

func assignmentJSON(a *quizgame.Assignment) map[string]any {
	out := map[string]any{
		"id":              a.ID,
		"kitId":           a.KitID,
		"courseId":        a.CourseID,
		"title":           a.Title,
		"attemptsAllowed": a.AttemptsAllowed,
		"gradePolicy":     a.GradePolicy,
		"shuffle":         a.Shuffle,
		"scoringProfile":  a.ScoringProfile,
		"scoringConfig":   json.RawMessage(a.ScoringConfig),
		"createdAt":       a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.OpensAt != nil {
		out["opensAt"] = a.OpensAt.UTC().Format(time.RFC3339)
	}
	if a.DueAt != nil {
		out["dueAt"] = a.DueAt.UTC().Format(time.RFC3339)
	}
	if a.ClosesAt != nil {
		out["closesAt"] = a.ClosesAt.UTC().Format(time.RFC3339)
	}
	if a.PointsPossible != nil {
		out["pointsPossible"] = *a.PointsPossible
	}
	if a.GradebookItemID != nil {
		out["gradebookItemId"] = *a.GradebookItemID
	}
	return out
}

// handleCreateQuizAssignment is POST .../kits/{kit_id}/assignments
func (d Deps) handleCreateQuizAssignment() http.HandlerFunc {
	type reqBody struct {
		Title           string     `json:"title"`
		OpensAt         *time.Time `json:"opensAt"`
		DueAt           *time.Time `json:"dueAt"`
		ClosesAt        *time.Time `json:"closesAt"`
		AttemptsAllowed int        `json:"attemptsAllowed"`
		GradePolicy     string     `json:"gradePolicy"`
		Shuffle         *bool      `json:"shuffle"`
		PointsPossible  *float64   `json:"pointsPossible"`
		ScoringProfile  string     `json:"scoringProfile"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to create assignments.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON.")
			return
		}
		a, err := quizgame.CreateAssignment(r.Context(), d.Pool, quizgame.CreateAssignmentInput{
			CourseCode:      courseCode,
			KitID:           kitID,
			Title:           body.Title,
			OpensAt:         body.OpensAt,
			DueAt:           body.DueAt,
			ClosesAt:        body.ClosesAt,
			AttemptsAllowed: body.AttemptsAllowed,
			GradePolicy:     body.GradePolicy,
			Shuffle:         body.Shuffle,
			PointsPossible:  body.PointsPossible,
			ScoringProfile:  body.ScoringProfile,
			CreatedBy:       viewer,
		})
		if errors.Is(err, quizgame.ErrKitNotReady) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Kit is not ready.")
			return
		}
		if errors.Is(err, quizgame.ErrSessionNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create assignment.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.assignment.create")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(assignmentJSON(a))
	}
}

// handleListQuizAssignments is GET .../live-quizzes/assignments
func (d Deps) handleListQuizAssignments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		list, err := quizgame.ListAssignmentsForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list assignments.")
			return
		}
		out := make([]map[string]any, 0, len(list))
		for i := range list {
			out = append(out, assignmentJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"assignments": out})
	}
}

// handleGetQuizAssignment is GET .../assignments/{assignment_id}
func (d Deps) handleGetQuizAssignment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		id := chi.URLParam(r, "assignment_id")
		a, err := quizgame.GetAssignmentByCourse(r.Context(), d.Pool, courseCode, id)
		if errors.Is(err, quizgame.ErrAssignmentNotFound) || a == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assignment not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load assignment.")
			return
		}
		out := assignmentJSON(a)
		now := time.Now().UTC()
		win, allowed, _ := quizgame.AssignmentWindowForUser(r.Context(), d.Pool, a, viewer, now)
		out["effectiveOpensAt"] = formatOptTime(win.OpensAt)
		out["effectiveDueAt"] = formatOptTime(win.DueAt)
		out["effectiveClosesAt"] = formatOptTime(win.ClosesAt)
		out["effectiveAttemptsAllowed"] = allowed
		state := "open"
		if err := engine.CheckPlayWindow(win, now); errors.Is(err, engine.ErrNotYetOpen) {
			state = "not_yet_open"
		} else if errors.Is(err, engine.ErrClosed) {
			state = "closed"
		} else if engine.IsLate(win, now) {
			state = "late"
		}
		used, _ := quizgame.CountAttempts(r.Context(), d.Pool, a.ID, viewer)
		if used >= allowed {
			state = "out_of_attempts"
		}
		if open, _ := quizgame.FindOpenAttempt(r.Context(), d.Pool, a.ID, viewer); open != nil {
			state = "in_progress"
			out["openAttemptId"] = open.ID
			out["openSessionId"] = open.SessionID
		}
		out["state"] = state
		out["attemptsUsed"] = used
		grade, _ := quizgame.GetAssignmentGrade(r.Context(), d.Pool, a.ID, viewer)
		out["gradebookScore"] = grade
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func formatOptTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// handleMyQuizAssignmentAttempts is GET .../assignments/{id}/my-attempts
func (d Deps) handleMyQuizAssignmentAttempts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		id := chi.URLParam(r, "assignment_id")
		if _, err := quizgame.GetAssignmentByCourse(r.Context(), d.Pool, courseCode, id); err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assignment not found.")
			return
		}
		attempts, err := quizgame.ListAttemptsForUser(r.Context(), d.Pool, id, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list attempts.")
			return
		}
		out := make([]map[string]any, 0, len(attempts))
		for _, at := range attempts {
			m := map[string]any{
				"id": at.ID, "attemptNo": at.AttemptNo, "score": at.Score,
				"sessionId": at.SessionID, "isLate": at.IsLate,
			}
			if at.SubmittedAt != nil {
				m["submittedAt"] = at.SubmittedAt.UTC().Format(time.RFC3339)
			}
			out = append(out, m)
		}
		grade, _ := quizgame.GetAssignmentGrade(r.Context(), d.Pool, id, viewer)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"attempts": out, "gradebookScore": grade})
	}
}

// handleStartQuizAssignment is POST .../assignments/{id}/start
func (d Deps) handleStartQuizAssignment() http.HandlerFunc {
	type reqBody struct {
		Nickname string `json:"nickname"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		id := chi.URLParam(r, "assignment_id")
		var body reqBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		at, sess, join, err := quizgame.StartAssignmentAttempt(r.Context(), d.Pool, courseCode, id, viewer, body.Nickname)
		if errors.Is(err, engine.ErrNotYetOpen) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Assignment is not open yet.")
			return
		}
		if errors.Is(err, engine.ErrClosed) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Assignment is closed.")
			return
		}
		if errors.Is(err, engine.ErrOutOfAttempts) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No attempts remaining.")
			return
		}
		if errors.Is(err, quizgame.ErrAssignmentNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assignment not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not start attempt.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.assignment.start")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"attemptId":   at.ID,
			"sessionId":   sess.ID,
			"attemptNo":   at.AttemptNo,
			"playerId":    join.Player.ID,
			"playerToken": join.PlayerToken,
			"rejoined":    join.Rejoined,
			"game":        sessionJSON(sess, false),
		})
	}
}

// handleSubmitQuizAssignmentAttempt is POST .../assignments/{id}/attempts/{aid}/submit
func (d Deps) handleSubmitQuizAssignmentAttempt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		aid := chi.URLParam(r, "assignment_id")
		attemptID := chi.URLParam(r, "attempt_id")
		at, grade, err := quizgame.SubmitAssignmentAttempt(r.Context(), d.Pool, courseCode, aid, attemptID, viewer)
		if errors.Is(err, engine.ErrNotYetOpen) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Assignment is not open yet.")
			return
		}
		if errors.Is(err, quizgame.ErrAssignmentNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit attempt.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.assignment.submit")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"attemptId":      at.ID,
			"score":          at.Score,
			"isLate":         at.IsLate,
			"submittedAt":    at.SubmittedAt.UTC().Format(time.RFC3339),
			"gradebookScore": grade,
		})
	}
}
