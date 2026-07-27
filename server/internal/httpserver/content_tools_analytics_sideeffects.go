package httpserver

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/user"
	ctanalytics "github.com/lextures/lextures/server/internal/service/contenttools/analytics"
	"github.com/lextures/lextures/server/internal/service/learningevents"
)

// afterContentToolsStateWrite syncs summary projection, gradebook bridge, and xAPI (CT.7).
// Preview-scope writes are ignored for summaries/grades (FR-5).
func (d Deps) afterContentToolsStateWrite(
	ctx context.Context,
	courseID uuid.UUID,
	courseCode string,
	toolID string,
	scope string,
	st *ctrepo.StateRow,
	verbHint string,
) {
	if st == nil || scope == ctrepo.ScopePreview {
		return
	}
	if err := ctanalytics.SyncSummaryFromState(ctx, d.Pool, courseID, toolID, st); err != nil {
		slog.Warn("contenttools.analytics.summary_sync", "err", err, "state_id", st.ID)
	}
	ctanalytics.MaybePushGradebook(ctx, d.Pool, courseID, st, toolID)
	d.emitContentToolXAPI(ctx, courseID, courseCode, toolID, st, verbHint)
}

func (d Deps) emitContentToolXAPI(
	ctx context.Context,
	courseID uuid.UUID,
	courseCode string,
	toolID string,
	st *ctrepo.StateRow,
	verbHint string,
) {
	if d.Pool == nil || st == nil {
		return
	}
	u, err := user.FindByID(ctx, d.Pool, st.UserID)
	if err != nil || u == nil {
		return
	}
	orgPtr, _ := ctrepo.CourseOrgID(ctx, d.Pool, courseID)
	orgID := uuid.Nil
	if orgPtr != nil {
		orgID = *orgPtr
	}
	em := learningevents.Emitter{Pool: d.Pool, Cfg: d.Config}
	title := toolID
	objectPath := "content-tools/" + st.InstanceID.String()
	dn := u.Email
	if u.DisplayName != nil && *u.DisplayName != "" {
		dn = *u.DisplayName
	}

	verb := verbHint
	if verb == "" {
		verb = "interacted"
		if st.Status == "completed" || st.Status == "submitted" {
			verb = "completed"
		}
	}
	switch verb {
	case "answered":
		em.ContentToolAnswered(ctx, orgID, courseID, courseCode, u.Email, dn, objectPath, title)
		ctanalytics.IncXAPI("answered")
	case "completed":
		em.ContentToolCompleted(ctx, orgID, courseID, courseCode, u.Email, dn, objectPath, title)
		ctanalytics.IncXAPI("completed")
		if st.ScoreRaw != nil && st.ScoreMax != nil && *st.ScoreMax > 0 {
			scaled := *st.ScoreRaw / *st.ScoreMax
			em.ContentToolScored(ctx, orgID, courseID, courseCode, u.Email, dn, objectPath, title, scaled)
			ctanalytics.IncXAPI("scored")
		}
	case "scored":
		if st.ScoreRaw != nil && st.ScoreMax != nil && *st.ScoreMax > 0 {
			scaled := *st.ScoreRaw / *st.ScoreMax
			em.ContentToolScored(ctx, orgID, courseID, courseCode, u.Email, dn, objectPath, title, scaled)
			ctanalytics.IncXAPI("scored")
		}
	default:
		em.ContentToolInteracted(ctx, orgID, courseID, courseCode, u.Email, dn, objectPath, title)
		ctanalytics.IncXAPI("interacted")
	}
}
