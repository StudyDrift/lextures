package httpserver

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func (d Deps) iqGradebookPushOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFIqGradebookPush {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return true
	}
	return false
}

func (d Deps) requireQuizGameInstructor(w http.ResponseWriter, r *http.Request, courseCode string) (uuid.UUID, bool) {
	_, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return uuid.Nil, false
	}
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.Nil, false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view this report.")
		return uuid.Nil, false
	}
	return viewer, true
}

func (d Deps) requireQuizGameGrader(w http.ResponseWriter, r *http.Request, courseCode string) (uuid.UUID, bool) {
	_, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return uuid.Nil, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.Nil, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage gradebook links.")
		return uuid.Nil, false
	}
	return viewer, true
}

// handleGetQuizGameReport is GET .../games/{game_id}/report
func (d Deps) handleGetQuizGameReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if _, ok := d.requireQuizGameInstructor(w, r, courseCode); !ok {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		rep, err := quizgame.EnsureGameReport(r.Context(), d.Pool, sess.ID)
		if errors.Is(err, quizgame.ErrGameNotEnded) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Report is available after the game ends.")
			return
		}
		if err != nil || rep == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not build report.")
			return
		}
		players, err := quizgame.BuildPlayerResults(r.Context(), d.Pool, sess.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load player results.")
			return
		}
		lb, _ := quizgame.ComputeLeaderboard(r.Context(), d.Pool, sess.ID, len(players)+1)
		link, _ := quizgame.GetGradebookLinkBySession(r.Context(), d.Pool, sess.ID)
		out := map[string]any{
			"report":      rep,
			"players":     players,
			"leaderboard": lb,
			"title":       sess.KitSnapshot.Title,
			"status":      sess.Status,
			"mode":        sess.Mode,
		}
		if link != nil {
			out["gradebookLink"] = gradebookLinkJSON(link)
		}
		guestCount := 0
		for _, p := range players {
			if p.IsGuest {
				guestCount++
			}
		}
		out["guestCount"] = guestCount
		telemetry.RecordBusinessEvent("quizgame.report.view")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleGetQuizGameMyResults is GET .../games/{game_id}/my-results
func (d Deps) handleGetQuizGameMyResults() http.HandlerFunc {
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
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		results, err := quizgame.BuildMyResults(r.Context(), d.Pool, sess.ID, viewer)
		if errors.Is(err, quizgame.ErrPlayerNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "You did not play this game.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load results.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.my_results.view")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(results)
	}
}

// handleRebuildQuizGameReport is POST .../games/{game_id}/report/rebuild
func (d Deps) handleRebuildQuizGameReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if _, ok := d.requireQuizGameInstructor(w, r, courseCode); !ok {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		rep, err := quizgame.BuildAndStoreReport(r.Context(), d.Pool, sess.ID)
		if errors.Is(err, quizgame.ErrGameNotEnded) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Report is available after the game ends.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not rebuild report.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.report.rebuild")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rep)
	}
}

// handleExportQuizGameReport is GET .../games/{game_id}/report/export?format=csv|pdf|html
func (d Deps) handleExportQuizGameReport() http.HandlerFunc {
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
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "csv"
		}
		if format != "csv" && format != "pdf" && format != "html" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be csv, pdf, or html.")
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}

		isInstructor, _ := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if isInstructor {
			rep, err := quizgame.EnsureGameReport(r.Context(), d.Pool, sess.ID)
			if err != nil || rep == nil {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Report not available yet.")
				return
			}
			players, err := quizgame.BuildPlayerResults(r.Context(), d.Pool, sess.ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load players.")
				return
			}
			responses, _ := quizgame.ListAllResponses(r.Context(), d.Pool, sess.ID)
			if format == "csv" {
				writeQuizGameReportCSV(w, sess, rep, players, responses, false, "")
			} else {
				writeQuizGameReportHTML(w, sess, rep, players, format == "pdf")
			}
			telemetry.RecordBusinessEvent("quizgame.report.export")
			return
		}

		// Student: own results only.
		results, err := quizgame.BuildMyResults(r.Context(), d.Pool, sess.ID, viewer)
		if errors.Is(err, quizgame.ErrPlayerNotFound) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You can only export your own results.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load results.")
			return
		}
		player, _ := quizgame.GetPlayerByUser(r.Context(), d.Pool, sess.ID, viewer)
		responses, _ := quizgame.ListPlayerResponses(r.Context(), d.Pool, sess.ID, player.ID)
		rep := &quizgame.GameReport{SessionID: sess.ID, PlayerCount: 1, AnsweredCount: results.Answered}
		players := []quizgame.PlayerResultRow{{
			PlayerID: player.ID, Nickname: results.Nickname, UserID: player.UserID,
			TotalScore: results.TotalScore, Rank: results.Rank, Answered: results.Answered, Correct: results.Correct,
		}}
		if format == "csv" {
			writeQuizGameReportCSV(w, sess, rep, players, responses, true, player.ID)
		} else {
			writeQuizGameMyResultsHTML(w, sess, results, format == "pdf")
		}
		telemetry.RecordBusinessEvent("quizgame.report.export")
	}
}

func writeQuizGameReportCSV(
	w http.ResponseWriter,
	sess *quizgame.Session,
	rep *quizgame.GameReport,
	players []quizgame.PlayerResultRow,
	responses []quizgame.Response,
	selfOnly bool,
	selfPlayerID string,
) {
	qCount := len(sess.KitSnapshot.Questions)
	var buf bytes.Buffer
	// UTF-8 BOM for Excel
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(&buf)
	header := []string{"nickname", "userId", "isGuest", "rank", "totalScore", "answered", "correct"}
	for i := 0; i < qCount; i++ {
		header = append(header, fmt.Sprintf("q%d_correct", i+1), fmt.Sprintf("q%d_points", i+1))
	}
	_ = cw.Write(header)

	type key struct {
		pid string
		qi  int
	}
	by := map[key]quizgame.Response{}
	for _, r := range responses {
		by[key{r.PlayerID, r.QuestionIndex}] = r
	}
	for _, p := range players {
		if selfOnly && p.PlayerID != selfPlayerID {
			continue
		}
		uid := ""
		if p.UserID != nil {
			uid = *p.UserID
		}
		row := []string{
			p.Nickname, uid, strconv.FormatBool(p.IsGuest),
			strconv.Itoa(p.Rank), strconv.Itoa(p.TotalScore),
			strconv.Itoa(p.Answered), strconv.Itoa(p.Correct),
		}
		for i := 0; i < qCount; i++ {
			r, ok := by[key{p.PlayerID, i}]
			if !ok {
				row = append(row, "", "")
				continue
			}
			row = append(row, strconv.FormatBool(r.IsCorrect), strconv.Itoa(r.Points))
		}
		_ = cw.Write(row)
	}
	cw.Flush()
	_ = rep
	filename := fmt.Sprintf("live-quiz-%s.csv", sess.ID[:8])
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = w.Write(buf.Bytes())
}

func writeQuizGameReportHTML(w http.ResponseWriter, sess *quizgame.Session, rep *quizgame.GameReport, players []quizgame.PlayerResultRow, asPDF bool) {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"/>")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(sess.KitSnapshot.Title))
	b.WriteString(" — Report</title>")
	b.WriteString("<style>body{font-family:system-ui,sans-serif;margin:2rem}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:.4rem .6rem;text-align:left}th{background:#f5f5f5}@media print{button{display:none}}</style>")
	b.WriteString("</head><body>")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(sess.KitSnapshot.Title))
	b.WriteString("</h1>")
	fmt.Fprintf(&b, "<p>Players: %d · Answered: %d", rep.PlayerCount, rep.AnsweredCount)
	if rep.ScoreAvg != nil {
		fmt.Fprintf(&b, " · Avg score: %.2f", *rep.ScoreAvg)
	}
	if rep.ScoreMedian != nil {
		fmt.Fprintf(&b, " · Median: %.2f", *rep.ScoreMedian)
	}
	b.WriteString("</p>")
	b.WriteString("<h2>Per-question</h2><table><thead><tr><th>#</th><th>Prompt</th><th>Correct %</th><th>Avg ms</th><th>Answers</th></tr></thead><tbody>")
	for _, q := range rep.PerQuestion {
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%.2f</td><td>%.0f</td><td>%d</td></tr>",
			q.Index+1, html.EscapeString(q.Prompt), q.CorrectPct, q.AvgMs, q.AnswerCount)
	}
	b.WriteString("</tbody></table>")
	b.WriteString("<h2>Players</h2><table><thead><tr><th>Rank</th><th>Nickname</th><th>Score</th><th>Correct</th><th>Guest</th></tr></thead><tbody>")
	for _, p := range players {
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%v</td></tr>",
			p.Rank, html.EscapeString(p.Nickname), p.TotalScore, p.Correct, p.IsGuest)
	}
	b.WriteString("</tbody></table>")
	fmt.Fprintf(&b, "<p><small>Generated %s</small></p>", html.EscapeString(time.Now().UTC().Format(time.RFC3339)))
	if !asPDF {
		b.WriteString(`<p><button onclick="window.print()">Print / Save as PDF</button></p>`)
	}
	b.WriteString("</body></html>")
	ct := "text/html; charset=utf-8"
	w.Header().Set("Content-Type", ct)
	if asPDF {
		w.Header().Set("Content-Disposition", `inline; filename="live-quiz-report.html"`)
	}
	_, _ = w.Write([]byte(b.String()))
}

func writeQuizGameMyResultsHTML(w http.ResponseWriter, sess *quizgame.Session, results *quizgame.MyResults, asPDF bool) {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"/><title>My results</title>")
	b.WriteString("<style>body{font-family:system-ui,sans-serif;margin:2rem}details{margin:.5rem 0;border:1px solid #ddd;padding:.5rem}@media print{button{display:none}}</style></head><body>")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(sess.KitSnapshot.Title))
	b.WriteString(" — My results</h1>")
	fmt.Fprintf(&b, "<p>Score: %d · Rank: %d of %d · Correct: %d / %d</p>",
		results.TotalScore, results.Rank, results.PlayerCount, results.Correct, results.Answered)
	b.WriteString("<h2>Questions to review</h2>")
	if len(results.ReviewThese) == 0 {
		b.WriteString("<p>Nothing to review — nice work!</p>")
	}
	for _, item := range results.ReviewThese {
		b.WriteString("<details open><summary>")
		b.WriteString(html.EscapeString(fmt.Sprintf("Q%d — %s (%s)", item.Index+1, item.Prompt, item.Reason)))
		b.WriteString("</summary>")
		if len(item.CorrectOptionIDs) > 0 {
			b.WriteString("<p>Correct options: ")
			b.WriteString(html.EscapeString(strings.Join(item.CorrectOptionIDs, ", ")))
			b.WriteString("</p>")
		}
		if item.Explanation != nil && *item.Explanation != "" {
			b.WriteString("<p>")
			b.WriteString(html.EscapeString(*item.Explanation))
			b.WriteString("</p>")
		}
		b.WriteString("</details>")
	}
	if !asPDF {
		b.WriteString(`<p><button onclick="window.print()">Print / Save as PDF</button></p>`)
	}
	b.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

// handlePostQuizGameGradebookLink is POST .../games/{game_id}/gradebook-link
func (d Deps) handlePostQuizGameGradebookLink() http.HandlerFunc {
	type body struct {
		Mapping          string   `json:"mapping"`
		PointsPossible   float64  `json:"pointsPossible"`
		ParticipationPct float64  `json:"participationPct"`
		Title            string   `json:"title"`
		PreviewOnly      bool     `json:"previewOnly"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if d.iqGradebookPushOff(w) {
			return
		}
		viewer, ok := d.requireQuizGameGrader(w, r, courseCode)
		if !ok {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.PreviewOnly {
			previews, err := quizgame.PreviewGradebook(r.Context(), d.Pool, courseCode, sess.ID, b.Mapping, b.PointsPossible, b.ParticipationPct)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not preview grades.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"preview": previews})
			return
		}
		sid := sess.ID
		link, previews, err := quizgame.PushGradebookLink(r.Context(), d.Pool, quizgame.CreateGradebookLinkInput{
			CourseCode:       courseCode,
			SessionID:        &sid,
			Mapping:          b.Mapping,
			PointsPossible:   b.PointsPossible,
			ParticipationPct: b.ParticipationPct,
			Title:            b.Title,
			CreatedBy:        viewer,
		})
		if errors.Is(err, quizgame.ErrInvalidMapping) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid mapping.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not push to gradebook.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.gradebook.push")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"link":    gradebookLinkJSON(link),
			"preview": previews,
		})
	}
}

// handleDeleteQuizGameGradebookLink is DELETE .../games/{game_id}/gradebook-link/{link_id}
func (d Deps) handleDeleteQuizGameGradebookLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		if d.iqGradebookPushOff(w) {
			return
		}
		viewer, ok := d.requireQuizGameGrader(w, r, courseCode)
		if !ok {
			return
		}
		linkID := chi.URLParam(r, "link_id")
		err := quizgame.UnlinkGradebook(r.Context(), d.Pool, courseCode, linkID, viewer)
		if errors.Is(err, quizgame.ErrGradebookLinkNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Gradebook link not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not unlink gradebook item.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.gradebook.unlink")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePostQuizAssignmentGradebookLink is POST .../assignments/{assignment_id}/gradebook-link
func (d Deps) handlePostQuizAssignmentGradebookLink() http.HandlerFunc {
	type body struct {
		Mapping          string  `json:"mapping"`
		PointsPossible   float64 `json:"pointsPossible"`
		ParticipationPct float64 `json:"participationPct"`
		Title            string  `json:"title"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqHomeworkFeatureOff(w, r, courseCode) {
			return
		}
		if d.iqGradebookPushOff(w) {
			return
		}
		viewer, ok := d.requireQuizGameGrader(w, r, courseCode)
		if !ok {
			return
		}
		assignmentID := chi.URLParam(r, "assignment_id")
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		aid := assignmentID
		link, previews, err := quizgame.PushGradebookLink(r.Context(), d.Pool, quizgame.CreateGradebookLinkInput{
			CourseCode:       courseCode,
			AssignmentID:     &aid,
			Mapping:          b.Mapping,
			PointsPossible:   b.PointsPossible,
			ParticipationPct: b.ParticipationPct,
			Title:            b.Title,
			CreatedBy:        viewer,
		})
		if errors.Is(err, quizgame.ErrAssignmentNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assignment not found.")
			return
		}
		if errors.Is(err, quizgame.ErrInvalidMapping) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid mapping.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not push to gradebook.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.gradebook.push")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"link":    gradebookLinkJSON(link),
			"preview": previews,
		})
	}
}

func gradebookLinkJSON(l *quizgame.GradebookLink) map[string]any {
	out := map[string]any{
		"id":              l.ID,
		"courseId":        l.CourseID,
		"gradebookItemId": l.GradebookItemID,
		"mapping":         l.Mapping,
		"participationPct": l.ParticipationPct,
	}
	if l.SessionID != nil {
		out["sessionId"] = *l.SessionID
	}
	if l.AssignmentID != nil {
		out["assignmentId"] = *l.AssignmentID
	}
	if l.PointsPossible != nil {
		out["pointsPossible"] = *l.PointsPossible
	}
	return out
}
