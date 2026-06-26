package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

type gradingAgentModuleItem struct {
	CourseID uuid.UUID
	ItemID   uuid.UUID
	Kind     string
	Title    string
}

func loadGradingAgentModuleItemByIDs(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	itemID uuid.UUID,
) (*gradingAgentModuleItem, error) {
	cid, err := course.GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	if cid == nil {
		return nil, errAssignmentNotFound
	}
	row, err := coursestructure.GetItemRow(ctx, pool, *cid, itemID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errAssignmentNotFound
	}
	kind := row.Kind
	switch kind {
	case "assignment":
		assignRow, err := coursemoduleassignments.GetForCourseItem(ctx, pool, *cid, itemID)
		if err != nil {
			return nil, err
		}
		if assignRow == nil {
			return nil, errAssignmentNotFound
		}
	case "quiz":
		quizRow, err := coursemodulequizzes.GetForCourseItem(ctx, pool, *cid, itemID)
		if err != nil {
			return nil, err
		}
		if quizRow == nil {
			return nil, errAssignmentNotFound
		}
	default:
		return nil, errAssignmentNotFound
	}
	return &gradingAgentModuleItem{
		CourseID: *cid,
		ItemID:   itemID,
		Kind:     kind,
		Title:    row.Title,
	}, nil
}

func (d Deps) loadGradingAgentModuleItem(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	itemID uuid.UUID,
) (*gradingAgentModuleItem, bool) {
	if d.Pool == nil {
		apierr.WriteJSONWithErr(w, r, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.", nil)
		return nil, false
	}
	item, err := loadGradingAgentModuleItemByIDs(r.Context(), d.Pool, courseCode, itemID)
	if err != nil {
		if err == errAssignmentNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return nil, false
		}
		apierr.WriteInternal(w, r, "Failed to load activity.", err)
		return nil, false
	}
	return item, true
}