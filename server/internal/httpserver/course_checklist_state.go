package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/service/coursechecklist"
)

func (d Deps) handleCourseChecklistGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("get", "4xx")
			return
		}
		includeNA := r.URL.Query().Get("includeNotApplicable") == "1"
		resp, err := d.checklistService().GetChecklist(r.Context(), courseCode, includeNA)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("get", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, resp)
		coursechecklist.ObserveAPIRequest("get", "2xx")
	}
}

func (d Deps) handleCourseChecklistSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("summary", "4xx")
			return
		}
		summary, err := d.checklistService().GetSummary(r.Context(), courseCode)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("summary", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, summary)
		coursechecklist.ObserveAPIRequest("summary", "2xx")
	}
}

func (d Deps) handleCourseChecklistRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("refresh", "4xx")
			return
		}
		limiter := d.buildRateLimiter()
		key := limiter.UserKey(courseCode, "checklist_refresh")
		if d.checklistRateLimited(w, r, key, 6) {
			coursechecklist.ObserveAPIRequest("refresh", "4xx")
			return
		}
		resp, err := d.checklistService().Refresh(r.Context(), courseCode)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("refresh", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, resp)
		coursechecklist.ObserveAPIRequest("refresh", "2xx")
	}
}

func (d Deps) handleCourseChecklistHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("history", "4xx")
			return
		}
		resp, err := d.checklistService().History(r.Context(), courseCode)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("history", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, resp)
		coursechecklist.ObserveAPIRequest("history", "2xx")
	}
}

func (d Deps) handleCourseChecklistDismiss() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, userID, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("dismiss", "4xx")
			return
		}
		limiter := d.buildRateLimiter()
		key := limiter.UserKey(userID.String(), "checklist_dismiss")
		if d.checklistRateLimited(w, r, key, 60) {
			coursechecklist.ObserveAPIRequest("dismiss", "4xx")
			return
		}
		itemID := strings.TrimSpace(chi.URLParam(r, "item_id"))
		var req coursechecklist.DismissRequest
		body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<10))
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				coursechecklist.ObserveAPIRequest("dismiss", "4xx")
				return
			}
		}
		item, err := d.checklistService().Dismiss(r.Context(), courseCode, itemID, userID, req)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("dismiss", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, item)
		coursechecklist.ObserveAPIRequest("dismiss", "2xx")
	}
}

func (d Deps) handleCourseChecklistRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, userID, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("restore", "4xx")
			return
		}
		limiter := d.buildRateLimiter()
		key := limiter.UserKey(userID.String(), "checklist_restore")
		if d.checklistRateLimited(w, r, key, 60) {
			coursechecklist.ObserveAPIRequest("restore", "4xx")
			return
		}
		itemID := strings.TrimSpace(chi.URLParam(r, "item_id"))
		item, err := d.checklistService().Restore(r.Context(), courseCode, itemID, userID)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("restore", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, item)
		coursechecklist.ObserveAPIRequest("restore", "2xx")
	}
}

func (d Deps) handleCourseChecklistRecheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseChecklistAccess(w, r)
		if !ok {
			coursechecklist.ObserveAPIRequest("recheck", "4xx")
			return
		}
		limiter := d.buildRateLimiter()
		key := limiter.UserKey(courseCode, "checklist_recheck")
		if d.checklistRateLimited(w, r, key, 30) {
			coursechecklist.ObserveAPIRequest("recheck", "4xx")
			return
		}
		itemID := strings.TrimSpace(chi.URLParam(r, "item_id"))
		item, err := d.checklistService().Recheck(r.Context(), courseCode, itemID)
		if err != nil {
			writeChecklistServiceError(w, err)
			coursechecklist.ObserveAPIRequest("recheck", "4xx")
			return
		}
		writeChecklistJSON(w, http.StatusOK, item)
		coursechecklist.ObserveAPIRequest("recheck", "2xx")
	}
}
