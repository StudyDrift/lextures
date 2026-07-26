package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

const contentToolsDuplicateRateLimitPerMin = 30

type contentToolsDupRateEntry struct {
	count  int
	window time.Time
}

var (
	contentToolsDupRateMu     sync.Mutex
	contentToolsDupRateByUser = map[uuid.UUID]contentToolsDupRateEntry{}
)

func checkContentToolsDuplicateRateLimit(userID uuid.UUID) bool {
	contentToolsDupRateMu.Lock()
	defer contentToolsDupRateMu.Unlock()
	now := time.Now()
	e, ok := contentToolsDupRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Minute {
		contentToolsDupRateByUser[userID] = contentToolsDupRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= contentToolsDuplicateRateLimitPerMin {
		return false
	}
	e.count++
	contentToolsDupRateByUser[userID] = e
	return true
}

func (d Deps) registerContentToolsAuthoringRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/duplicate", d.handleContentToolsInstanceDuplicate())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/usage", d.handleContentToolsInstanceUsage())
}

// maybeReconcileContentToolMarkdown runs fence reconciliation when Content Tools is
// available for the course. On failure or when disabled, returns markdown unchanged.
func (d Deps) maybeReconcileContentToolMarkdown(
	ctx context.Context,
	courseCode string,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	markdown string,
) string {
	if d.Pool == nil {
		return markdown
	}
	pub, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || pub == nil || !ctsvc.AvailableForCourse(pub.ContentToolsEnabled) {
		return markdown
	}
	cleaned, err := ctsvc.ReconcileMarkdownFences(ctx, d.Pool, courseID, structureItemID, markdown)
	if err != nil {
		return markdown
	}
	return cleaned
}

// maybeReconcileContentToolMarkdownBodies reconciles multiple bodies that share one
// host item (syllabus sections). Archive runs once against the union of refs.
func (d Deps) maybeReconcileContentToolMarkdownBodies(
	ctx context.Context,
	courseCode string,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	bodies []string,
) []string {
	if d.Pool == nil {
		return bodies
	}
	pub, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || pub == nil || !ctsvc.AvailableForCourse(pub.ContentToolsEnabled) {
		return bodies
	}
	cleaned, err := ctsvc.ReconcileMarkdownBodies(ctx, d.Pool, courseID, structureItemID, bodies)
	if err != nil {
		return bodies
	}
	return cleaned
}

func (d Deps) handleContentToolsInstanceDuplicate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if !checkContentToolsDuplicateRateLimit(viewer) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Duplicate rate limit exceeded.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		existing, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		settings, err := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		maxInst := int16(50)
		if settings != nil {
			maxInst = settings.MaxInstancesPerItem
		}
		n, err := ctrepo.CountActiveForItem(r.Context(), d.Pool, courseID, existing.StructureItemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to count instances.")
			return
		}
		if int16(n) >= maxInst {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrMaxInstances.Error())
			return
		}
		actor := viewer
		created, err := ctrepo.CreateInstance(r.Context(), d.Pool, ctrepo.InstanceRow{
			CourseID:            courseID,
			StructureItemID:     existing.StructureItemID,
			HostKind:            existing.HostKind,
			SectionKey:          existing.SectionKey,
			ToolID:              existing.ToolID,
			ToolVersion:         existing.ToolVersion,
			Title:               existing.Title,
			ConfigJSON:          existing.ConfigJSON,
			ConfigSchemaVersion: existing.ConfigSchemaVersion,
			CreatedBy:           &actor,
		})
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, ctsvc.ErrConfigTooLarge.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to duplicate instance.")
			return
		}
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &created.ID, nil, &actor, created.ToolID, ctsvc.EventInstanceDuplicated, map[string]any{
			"sourceInstanceId": existing.ID.String(),
			"hostKind":         created.HostKind,
		})
		ctsvc.IncInstanceAction(created.ToolID, "duplicate")
		ctsvc.IncInsert(created.ToolID, "duplicate")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(instanceToAPI(*created, created.ConfigJSON))
	}
}

func (d Deps) handleContentToolsInstanceUsage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		withState, completed, lastAt, err := ctrepo.CountInstanceUsage(r.Context(), d.Pool, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load usage.")
			return
		}
		referenced, err := contentToolInstanceReferencedInBody(r.Context(), d.Pool, inst)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check body reference.")
			return
		}
		var lastStr *string
		if lastAt != nil {
			s := lastAt.UTC().Format(time.RFC3339Nano)
			lastStr = &s
		}
		out := ctmodel.ToolInstanceUsage{
			InstanceID:        instanceID,
			LearnersWithState: withState,
			LearnersCompleted: completed,
			LastInteractionAt: lastStr,
			ReferencedInBody:  referenced,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func contentToolInstanceReferencedInBody(ctx context.Context, pool *pgxpool.Pool, inst *ctrepo.InstanceRow) (bool, error) {
	if inst == nil || pool == nil {
		return false, nil
	}
	needle := inst.ID.String()
	switch inst.HostKind {
	case "content_page":
		return markdownContainsInstanceID(ctx, pool, `
SELECT markdown FROM course.module_content_pages WHERE structure_item_id = $1
`, inst.StructureItemID, needle)
	case "assignment":
		return markdownContainsInstanceID(ctx, pool, `
SELECT markdown FROM course.module_assignments WHERE structure_item_id = $1
`, inst.StructureItemID, needle)
	case "quiz":
		return markdownContainsInstanceID(ctx, pool, `
SELECT markdown FROM course.module_quizzes WHERE structure_item_id = $1
`, inst.StructureItemID, needle)
	case "syllabus":
		var raw []byte
		err := pool.QueryRow(ctx, `
SELECT COALESCE(sections, '[]'::jsonb) FROM course.course_syllabus WHERE course_id = $1
`, inst.CourseID).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return strings.Contains(string(raw), needle), nil
	case "portfolio_artifact":
		return false, nil
	default:
		return false, nil
	}
}

func markdownContainsInstanceID(ctx context.Context, pool *pgxpool.Pool, query string, itemID *uuid.UUID, needle string) (bool, error) {
	if itemID == nil {
		return false, nil
	}
	var md string
	err := pool.QueryRow(ctx, query, *itemID).Scan(&md)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.Contains(md, needle), nil
}
