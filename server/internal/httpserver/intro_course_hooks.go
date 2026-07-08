package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

func (d Deps) maybeAutogradeIntroQuiz(ctx context.Context, courseID, studentID, itemID uuid.UUID) {
	if d.Pool == nil {
		return
	}
	_ = introcourseservice.OnQuizAttempt(ctx, d.Pool, d.effectiveConfig(), courseID, studentID, itemID)
}

func (d Deps) maybeAutogradeIntroAssignment(r *http.Request, courseCode string, courseID, studentID, itemID uuid.UUID) {
	if d.Pool == nil {
		return
	}
	req, err := introcourseservice.OnAssignmentSubmit(
		r.Context(), d.Pool, d.effectiveConfig(), courseID, studentID, itemID, courseCode,
	)
	if err != nil || req.CourseID == uuid.Nil {
		return
	}
	d.enqueueIntroCourseGraderFeedback(r, req)
}

func (d Deps) enqueueIntroCourseGraderFeedback(r *http.Request, req introcourseservice.GraderAgentRequest) {
	if !d.graderAgentEnabled() || !d.graderAgentTextEntryGradingEnabled() || d.GradingAgentQueue == nil || d.Pool == nil {
		return
	}
	cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, req.ItemID)
	if err != nil || cfg == nil || cfg.Status != gradingagentrepo.StatusAccepted {
		return
	}
	run, err := gradingagentrepo.CreateRun(
		r.Context(), d.Pool, cfg.ID,
		gradingagentrepo.RunScopeAuto, gradingagentrepo.RunModeSuggest,
		nil, nil, 1, nil, nil,
	)
	if err != nil || run == nil {
		return
	}
	_ = gradingagentrepo.MarkRunRunning(r.Context(), d.Pool, run.ID)
	_ = d.GradingAgentQueue.Publish(r.Context(), gradingagentqueue.QueueMessage{
		RunID: run.ID, ConfigID: cfg.ID, SubmissionID: req.SubmissionID,
		CourseID: req.CourseID, ItemID: req.ItemID, CourseCode: req.CourseCode,
	})
}