package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	modelenrollmentgroup "github.com/lextures/lextures/server/internal/models/enrollmentgroup"
	"github.com/lextures/lextures/server/internal/repos/course"
	egrepo "github.com/lextures/lextures/server/internal/repos/enrollmentgroups"
)

func (d Deps) registerEnrollmentGroupRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/enrollment-groups/enable", d.handleEnrollmentGroupsEnable())
	r.Get("/api/v1/courses/{course_code}/enrollment-groups", d.handleEnrollmentGroupsTree())
	r.Post("/api/v1/courses/{course_code}/enrollment-groups/sets", d.handleEnrollmentGroupSetCreate())
	r.Patch("/api/v1/courses/{course_code}/enrollment-groups/sets/{set_id}", d.handleEnrollmentGroupSetPatch())
	r.Delete("/api/v1/courses/{course_code}/enrollment-groups/sets/{set_id}", d.handleEnrollmentGroupSetDelete())
	r.Post("/api/v1/courses/{course_code}/enrollment-groups/sets/{set_id}/groups", d.handleEnrollmentGroupCreate())
	r.Patch("/api/v1/courses/{course_code}/enrollment-groups/groups/{group_id}", d.handleEnrollmentGroupPatch())
	r.Delete("/api/v1/courses/{course_code}/enrollment-groups/groups/{group_id}", d.handleEnrollmentGroupDelete())
	r.Put("/api/v1/courses/{course_code}/enrollment-groups/memberships", d.handleEnrollmentGroupMembershipPut())
}

func (d Deps) resolveCourseIDFromCode(w http.ResponseWriter, r *http.Request, courseCode string) (uuid.UUID, bool) {
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return uuid.Nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return uuid.Nil, false
	}
	return *cid, true
}

func (d Deps) requireEnrollmentGroupsManage(w http.ResponseWriter, r *http.Request) (courseCode string, courseID uuid.UUID, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	can, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:update")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !can {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage enrollment groups.")
		return "", uuid.Nil, uuid.Nil, false
	}
	courseID, ok = d.resolveCourseIDFromCode(w, r, courseCode)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	return courseCode, courseID, viewer, true
}

func (d Deps) handleEnrollmentGroupsEnable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		if err := egrepo.SetEnabled(r.Context(), d.Pool, courseID, true); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enable enrollment groups.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleEnrollmentGroupsTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		courseID, ok := d.resolveCourseIDFromCode(w, r, courseCode)
		if !ok {
			return
		}
		// Always return a tree (empty if disabled or unset) so clients can probe without 404 noise.
		tree, err := egrepo.Tree(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment groups.")
			return
		}
		if tree.GroupSets == nil {
			tree.GroupSets = []modelenrollmentgroup.EnrollmentGroupSetPublic{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(tree)
	}
}

func (d Deps) handleEnrollmentGroupSetCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		var body modelenrollmentgroup.CreateEnrollmentGroupSetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		id, err := egrepo.CreateSet(r.Context(), d.Pool, courseID, body.Name)
		if err != nil {
			if strings.Contains(err.Error(), "name required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name is required.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create group set.")
			return
		}
		// Enabling is implicit when staff create structure.
		_ = egrepo.SetEnabled(r.Context(), d.Pool, courseID, true)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleEnrollmentGroupCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		setID, err := uuid.Parse(chi.URLParam(r, "set_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid set id.")
			return
		}
		var body modelenrollmentgroup.CreateEnrollmentGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		id, err := egrepo.CreateGroup(r.Context(), d.Pool, courseID, setID, body.Name)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Group set not found.")
				return
			}
			if strings.Contains(err.Error(), "name required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name is required.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create group.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleEnrollmentGroupSetPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		setID, err := uuid.Parse(chi.URLParam(r, "set_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid set id.")
			return
		}
		var body modelenrollmentgroup.PatchEnrollmentGroupSetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := egrepo.PatchSetName(r.Context(), d.Pool, courseID, setID, body.Name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Group set not found.")
				return
			}
			if strings.Contains(err.Error(), "name required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name is required.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update group set.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleEnrollmentGroupPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid group id.")
			return
		}
		var body modelenrollmentgroup.PatchEnrollmentGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := egrepo.PatchGroupName(r.Context(), d.Pool, courseID, groupID, body.Name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Group not found.")
				return
			}
			if strings.Contains(err.Error(), "name required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name is required.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update group.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleEnrollmentGroupSetDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		setID, err := uuid.Parse(chi.URLParam(r, "set_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid set id.")
			return
		}
		if err := egrepo.DeleteSet(r.Context(), d.Pool, courseID, setID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Group set not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete group set.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleEnrollmentGroupDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid group id.")
			return
		}
		if err := egrepo.DeleteGroup(r.Context(), d.Pool, courseID, groupID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Group not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete group.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleEnrollmentGroupMembershipPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, courseID, _, ok := d.requireEnrollmentGroupsManage(w, r)
		if !ok {
			return
		}
		var body modelenrollmentgroup.PutEnrollmentGroupMembershipRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.EnrollmentID == uuid.Nil || body.GroupSetID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "enrollmentId and groupSetId are required.")
			return
		}
		if err := egrepo.PutMembership(r.Context(), d.Pool, courseID, body.EnrollmentID, body.GroupSetID, body.GroupID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment, group set, or group not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update membership.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
