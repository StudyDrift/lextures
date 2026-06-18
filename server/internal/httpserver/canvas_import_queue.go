package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/canvasimportevents"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

const canvasImportMaxAttempts int16 = 3

// handleCourseImportCanvasPost is POST /api/v1/courses/{course_code}/import/canvas.
func (d Deps) handleCourseImportCanvasPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil || d.CanvasImportQueue == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		hasAccess, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
		if err != nil || !hasAccess {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Course not found or you do not have access.")
			return
		}
		canImport, err := courseroles.UserHasPermission(r.Context(), d.Pool, userID, "course:"+courseCode+":item:create")
		if err != nil || !canImport {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to import into this course.")
			return
		}

		var req struct {
			Mode                   string              `json:"mode"`
			CanvasBaseURL          string              `json:"canvasBaseUrl"`
			CanvasCourseID         string              `json:"canvasCourseId"`
			AccessToken            string              `json:"accessToken"`
			Include                canvasImportInclude `json:"include"`
			CanvasGradeSyncEnabled *bool               `json:"canvasGradeSyncEnabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.CanvasBaseURL == "" || req.CanvasCourseID == "" || req.AccessToken == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas base URL, course id, and access token are required.")
			return
		}
		include := req.Include.withDefaults()

		jobID, err := canvasimportjobs.Insert(r.Context(), d.Pool, userID, courseCode, req.Mode, req.CanvasBaseURL, req.CanvasCourseID, includeToRepo(include))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to queue Canvas import.")
			return
		}
		if req.CanvasGradeSyncEnabled != nil && *req.CanvasGradeSyncEnabled {
			if _, err := course.SetCanvasGradeSyncEnabled(r.Context(), d.Pool, courseCode, true); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enable Canvas grade sync.")
				return
			}
		}
		msg := canvasimportjobs.QueueMessage{
			JobID:          jobID,
			UserID:         userID,
			CourseCode:     courseCode,
			Mode:           req.Mode,
			CanvasBaseURL:  req.CanvasBaseURL,
			CanvasCourseID: req.CanvasCourseID,
			AccessToken:    req.AccessToken,
			Include:        includeToRepo(include),
		}
		if err := d.CanvasImportQueue.Publish(r.Context(), msg); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue Canvas import.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"jobId":   jobID.String(),
			"message": "Canvas import queued. You can leave this page and refresh later — we will notify you when it finishes.",
		})
	}
}

// handleCanvasImportJobWS is GET /api/v1/ws/canvas-import/{job_id}.
func (d Deps) handleCanvasImportJobWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			http.Error(w, "invalid job_id", http.StatusBadRequest)
			return
		}
		if d.Pool == nil {
			http.Error(w, "server misconfiguration", http.StatusServiceUnavailable)
			return
		}

		conn, wsErr := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if wsErr != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		readAuthCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()
		typ, b, err := conn.Read(readAuthCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}
		var m struct {
			AuthToken string `json:"authToken"`
		}
		if err := json.Unmarshal(b, &m); err != nil || m.AuthToken == "" {
			return
		}
		u, err := d.JWTSigner.Verify(r.Context(), m.AuthToken)
		if err != nil {
			return
		}
		uid, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}

		job, err := canvasimportjobs.Load(r.Context(), d.Pool, jobID)
		if err != nil || job == nil {
			_ = wsWriteJSON(r.Context(), conn, map[string]any{"type": "error", "message": "Import job not found."})
			return
		}
		if job.UserID != uid {
			return
		}

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		if job.LastProgress != nil && *job.LastProgress != "" {
			_ = wsWriteJSON(runCtx, conn, map[string]any{"type": "progress", "message": *job.LastProgress})
		}
		if terminalWSMessage(runCtx, conn, job) {
			return
		}

		if d.CanvasImportHub != nil {
			recv, unsub := d.CanvasImportHub.Subscribe(jobID)
			defer unsub()
			go func() {
				for {
					select {
					case ev, ok := <-recv:
						if !ok {
							return
						}
						_ = wsWriteJSON(runCtx, conn, ev)
						if ev.Type == "complete" || ev.Type == "error" {
							stop()
							return
						}
					case <-runCtx.Done():
						return
					}
				}
			}()
		}

		for {
			if _, _, err := conn.Read(runCtx); err != nil {
				return
			}
		}
	}
}

func terminalWSMessage(ctx context.Context, conn *websocket.Conn, job *canvasimportjobs.Job) bool {
	switch job.Status {
	case canvasimportjobs.StatusCompleted:
		_ = wsWriteJSON(ctx, conn, map[string]any{"type": "complete"})
		return true
	case canvasimportjobs.StatusFailed:
		msg := "Canvas import failed."
		if job.ErrorMessage != nil && *job.ErrorMessage != "" {
			msg = *job.ErrorMessage
		}
		_ = wsWriteJSON(ctx, conn, map[string]any{"type": "error", "message": msg})
		return true
	default:
		return false
	}
}

func includeToRepo(i canvasImportInclude) canvasimportjobs.Include {
	return canvasimportjobs.Include{
		Modules:     i.Modules,
		Assignments: i.Assignments,
		Quizzes:     i.Quizzes,
		Enrollments: i.Enrollments,
		Grades:      i.Grades,
		Settings:    i.Settings,
		Files:       i.Files,
	}
}

// HandleCanvasImportQueueMessage is invoked by the background consumer for each queued job.
func (d Deps) HandleCanvasImportQueueMessage(ctx context.Context, msg canvasimportjobs.QueueMessage) error {
	emit := func(text string) bool {
		if d.CanvasImportHub != nil {
			d.CanvasImportHub.Broadcast(msg.JobID, canvasimportevents.Message{Type: "progress", Message: text})
		}
		_ = canvasimportjobs.UpdateProgress(ctx, d.Pool, msg.JobID, text)
		return ctx.Err() == nil
	}
	if !emit("Connecting to Canvas...") {
		return context.Canceled
	}
	return d.processCanvasImportQueueMessage(ctx, msg, emit)
}

func canvasImportJobAlreadyTerminal(status canvasimportjobs.Status) bool {
	return status == canvasimportjobs.StatusCompleted || status == canvasimportjobs.StatusFailed
}

func (d Deps) processCanvasImportQueueMessage(ctx context.Context, msg canvasimportjobs.QueueMessage, progress func(string) bool) error {
	if d.Pool == nil {
		return errors.New("server misconfiguration")
	}
	job, err := canvasimportjobs.Load(ctx, d.Pool, msg.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("canvas import job %s not found", msg.JobID)
	}
	if canvasImportJobAlreadyTerminal(job.Status) {
		if job.Status == canvasimportjobs.StatusCompleted && d.CanvasImportHub != nil {
			d.CanvasImportHub.Broadcast(msg.JobID, canvasimportevents.Message{Type: "complete"})
		}
		if job.Status == canvasimportjobs.StatusFailed && d.CanvasImportHub != nil {
			errMsg := "Canvas import failed."
			if job.ErrorMessage != nil && *job.ErrorMessage != "" {
				errMsg = *job.ErrorMessage
			}
			d.CanvasImportHub.Broadcast(msg.JobID, canvasimportevents.Message{Type: "error", Message: errMsg})
		}
		return nil
	}
	if err := canvasimportjobs.MarkProcessing(ctx, d.Pool, msg.JobID); err != nil {
		return err
	}
	include := canvasImportInclude{
		Modules:     msg.Include.Modules,
		Assignments: msg.Include.Assignments,
		Quizzes:     msg.Include.Quizzes,
		Enrollments: msg.Include.Enrollments,
		Grades:      msg.Include.Grades,
		Settings:    msg.Include.Settings,
		Files:       msg.Include.Files,
	}.withDefaults()

	importErr := d.runCanvasImport(ctx, msg.UserID, msg.CourseCode, msg.Mode, msg.CanvasBaseURL, msg.CanvasCourseID, msg.AccessToken, include, progress)
	if importErr != nil {
		if markErr := canvasimportjobs.MarkFailed(ctx, d.Pool, msg.JobID, importErr.Error(), canvasImportMaxAttempts); markErr != nil {
			return fmt.Errorf("import failed: %w; mark failed: %v", importErr, markErr)
		}
		if d.CanvasImportHub != nil {
			d.CanvasImportHub.Broadcast(msg.JobID, canvasimportevents.Message{Type: "error", Message: importErr.Error()})
		}
		return importErr
	}

	var courseTitle string
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(NULLIF(title, ''), $1) FROM course.courses WHERE course_code = $2`, msg.CourseCode, msg.CourseCode).Scan(&courseTitle)
	if markErr := canvasimportjobs.MarkCompleted(ctx, d.Pool, msg.JobID, courseTitle); markErr != nil {
		return markErr
	}
	d.pushNotificationService().EnqueueCanvasCourseImported(ctx, msg.UserID, courseTitle, msg.CourseCode)
	d.notifyCourses(msg.UserID)
	broadcastStructureChanged(msg.CourseCode)
	if d.CanvasImportHub != nil {
		d.CanvasImportHub.Broadcast(msg.JobID, canvasimportevents.Message{Type: "complete"})
	}
	return nil
}

func (d Deps) registerCanvasImportRoutes(r chi.Router) {
	r.Get("/api/v1/ws/canvas-import/{job_id}", d.handleCanvasImportJobWS())
}
