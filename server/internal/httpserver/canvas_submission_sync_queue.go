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

	"github.com/lextures/lextures/server/internal/canvassubmissionsyncevents"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncjobs"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
)

// handleCanvasSubmissionSyncJobWS is GET /api/v1/ws/canvas-submission-sync/{job_id}.
func (d Deps) handleCanvasSubmissionSyncJobWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			http.Error(w, "invalid job_id", http.StatusBadRequest)
			return
		}
		if d.CanvasSubmissionSyncJobs == nil {
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

		job := d.CanvasSubmissionSyncJobs.Get(jobID)
		if job == nil {
			_ = wsWriteJSON(r.Context(), conn, map[string]any{"type": "error", "message": "Sync job not found."})
			return
		}
		if job.UserID != uid {
			return
		}

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		if terminalSubmissionSyncWSMessage(runCtx, conn, job) {
			return
		}

		if d.CanvasSubmissionSyncHub != nil {
			recv, unsub := d.CanvasSubmissionSyncHub.Subscribe(jobID)
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

func terminalSubmissionSyncWSMessage(ctx context.Context, conn *websocket.Conn, job *canvassubmissionsyncjobs.Job) bool {
	switch job.Status {
	case canvassubmissionsyncjobs.StatusCompleted:
		_ = wsWriteJSON(ctx, conn, canvassubmissionsyncevents.Message{Type: "complete", Grade: job.Result})
		return true
	case canvassubmissionsyncjobs.StatusFailed:
		msg := "Could not sync to Canvas."
		if job.ErrorMessage != "" {
			msg = job.ErrorMessage
		}
		_ = wsWriteJSON(ctx, conn, canvassubmissionsyncevents.Message{Type: "error", Message: msg})
		return true
	default:
		return false
	}
}

// HandleCanvasSubmissionSyncQueueMessage is invoked by the background consumer for each queued job.
func (d Deps) HandleCanvasSubmissionSyncQueueMessage(ctx context.Context, msg canvassubmissionsyncqueue.QueueMessage) error {
	if d.CanvasSubmissionSyncJobs == nil {
		return errors.New("server misconfiguration")
	}
	job := d.CanvasSubmissionSyncJobs.Get(msg.JobID)
	if job == nil {
		return fmt.Errorf("canvas submission sync job %s not found", msg.JobID)
	}
	if job.Status == canvassubmissionsyncjobs.StatusCompleted || job.Status == canvassubmissionsyncjobs.StatusFailed {
		d.broadcastTerminalSubmissionSyncJob(msg.JobID, job)
		return nil
	}

	d.CanvasSubmissionSyncJobs.MarkProcessing(msg.JobID)

	result, syncErr := d.executeSubmissionSyncToCanvas(ctx, submissionSyncCanvasInput{
		CourseCode:        msg.CourseCode,
		ItemID:            msg.ItemID,
		SubmissionID:      msg.SubmissionID,
		CanvasBaseURL:     msg.CanvasBaseURL,
		AccessToken:       msg.AccessToken,
		PointsEarned:      msg.PointsEarned,
		RubricScores:      msg.RubricScores,
		InstructorComment: msg.InstructorComment,
	})
	if syncErr != nil {
		d.CanvasSubmissionSyncJobs.MarkFailed(msg.JobID, syncErr.Error())
		if d.CanvasSubmissionSyncHub != nil {
			d.CanvasSubmissionSyncHub.Broadcast(msg.JobID, canvassubmissionsyncevents.Message{
				Type:    "error",
				Message: syncErr.Error(),
			})
		}
		return syncErr
	}

	d.CanvasSubmissionSyncJobs.MarkCompleted(msg.JobID, result)
	if d.CanvasSubmissionSyncHub != nil {
		d.CanvasSubmissionSyncHub.Broadcast(msg.JobID, canvassubmissionsyncevents.Message{
			Type:  "complete",
			Grade: result,
		})
	}
	return nil
}

func (d Deps) broadcastTerminalSubmissionSyncJob(jobID uuid.UUID, job *canvassubmissionsyncjobs.Job) {
	if d.CanvasSubmissionSyncHub == nil || job == nil {
		return
	}
	switch job.Status {
	case canvassubmissionsyncjobs.StatusCompleted:
		d.CanvasSubmissionSyncHub.Broadcast(jobID, canvassubmissionsyncevents.Message{Type: "complete", Grade: job.Result})
	case canvassubmissionsyncjobs.StatusFailed:
		msg := "Could not sync to Canvas."
		if job.ErrorMessage != "" {
			msg = job.ErrorMessage
		}
		d.CanvasSubmissionSyncHub.Broadcast(jobID, canvassubmissionsyncevents.Message{Type: "error", Message: msg})
	}
}

func (d Deps) registerCanvasSubmissionSyncRoutes(r chi.Router) {
	r.Get("/api/v1/ws/canvas-submission-sync/{job_id}", d.handleCanvasSubmissionSyncJobWS())
}