package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// courseIDFromCode fetches the course UUID for a given course code.
func (d Deps) courseIDFromCode(ctx context.Context, courseCode string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := d.Pool.QueryRow(ctx, `SELECT id FROM course.courses WHERE course_code = $1`, courseCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, false, nil
	}
	return id, err == nil, err
}

// handleGetCourseFiles is GET /api/v1/courses/{course_code}/files
// Returns root-level folders and files.
func (d Deps) handleGetCourseFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		contents, err := filemanager.ListContents(r.Context(), d.Pool, courseID, nil)
		if err != nil {
			log.Printf("course-files-list: course=%q err=%v", courseCode, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list files.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contents)
	}
}

// handleGetCourseFilesFolder is GET /api/v1/courses/{course_code}/files/folders/{folder_id}
// Returns the folder's children.
func (d Deps) handleGetCourseFilesFolder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		folderID, err := uuid.Parse(chi.URLParam(r, "folder_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folder id.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		folder, err := filemanager.GetFolder(r.Context(), d.Pool, courseID, folderID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load folder.")
			return
		}
		if folder == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Folder not found.")
			return
		}
		contents, err := filemanager.ListContents(r.Context(), d.Pool, courseID, &folderID)
		if err != nil {
			log.Printf("course-files-folder: course=%q folder=%s err=%v", courseCode, folderID, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list folder contents.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contents)
	}
}

// handlePostCourseFilesFolder is POST /api/v1/courses/{course_code}/files/folders
func (d Deps) handlePostCourseFilesFolder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to create folders.")
			return
		}
		var body struct {
			Name     string  `json:"name"`
			ParentID *string `json:"parentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" || len(name) > 255 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Folder name must be 1–255 characters.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var parentID *uuid.UUID
		if body.ParentID != nil && *body.ParentID != "" {
			pid, parseErr := uuid.Parse(*body.ParentID)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid parentId.")
				return
			}
			parentID = &pid
		}
		folder, err := filemanager.CreateFolder(r.Context(), d.Pool, courseID, parentID, name, &viewer)
		if err != nil {
			log.Printf("course-files-create-folder: course=%q err=%v", courseCode, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create folder.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(folder)
		broadcastFilesChanged(courseCode)
	}
}

// handlePatchCourseFilesFolder is PATCH /api/v1/courses/{course_code}/files/folders/{folder_id}
// Supports rename (name) and move (parentId).
func (d Deps) handlePatchCourseFilesFolder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit folders.")
			return
		}
		folderID, err := uuid.Parse(chi.URLParam(r, "folder_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folder id.")
			return
		}
		var body struct {
			Name     *string `json:"name"`
			ParentID *string `json:"parentId"` // empty string = move to root
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if body.Name != nil {
			name := strings.TrimSpace(*body.Name)
			if name == "" || len(name) > 255 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Folder name must be 1–255 characters.")
				return
			}
			folder, renameErr := filemanager.RenameFolder(r.Context(), d.Pool, courseID, folderID, name)
			if renameErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to rename folder.")
				return
			}
			if folder == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Folder not found.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(folder)
			broadcastFilesChanged(courseCode)
			return
		}
		if body.ParentID != nil {
			var parentID *uuid.UUID
			if *body.ParentID != "" {
				fid, parseErr := uuid.Parse(*body.ParentID)
				if parseErr != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid parentId.")
					return
				}
				parentID = &fid
			}
			folder, moveErr := filemanager.MoveFolder(r.Context(), d.Pool, courseID, folderID, parentID)
			if moveErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, moveErr.Error())
				return
			}
			if folder == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Folder not found.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(folder)
			broadcastFilesChanged(courseCode)
			return
		}
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide name or parentId to update.")
	}
}

// handleDeleteCourseFilesFolder is DELETE /api/v1/courses/{course_code}/files/folders/{folder_id}
func (d Deps) handleDeleteCourseFilesFolder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to delete folders.")
			return
		}
		folderID, err := uuid.Parse(chi.URLParam(r, "folder_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folder id.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		// Collect storage keys before deleting (cascade will remove records)
		storageKeys, err := filemanager.StorageKeysInFolder(r.Context(), d.Pool, courseID, folderID)
		if err != nil {
			log.Printf("course-files-delete-folder: collect keys course=%q folder=%s err=%v", courseCode, folderID, err)
		}
		deleted, err := filemanager.DeleteFolder(r.Context(), d.Pool, courseID, folderID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete folder.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Folder not found.")
			return
		}
		storage := d.Storage
		cfg := d.effectiveConfig()
		go func() {
			for _, key := range storageKeys {
				if storage != nil {
					if delErr := storage.DeleteObject(context.Background(), key); delErr != nil {
						log.Printf("course-files-delete-folder: blob delete key=%q err=%v", key, delErr)
					}
				} else {
					root := strings.TrimSpace(cfg.CourseFilesRoot)
					if root == "" {
						root = "data/course-files"
					}
					path := root + "/" + courseCode + "/" + key
					_ = deleteLocalFile(path)
				}
			}
		}()
		w.WriteHeader(http.StatusNoContent)
		broadcastFilesChanged(courseCode)
	}
}

// handlePostCourseFileItem is POST /api/v1/courses/{course_code}/files/items
// Stores blob (or returns presigned URL) and registers file metadata.
func (d Deps) handlePostCourseFileItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to upload files.")
			return
		}
		filename := strings.TrimSpace(r.URL.Query().Get("filename"))
		if filename == "" {
			filename = "upload"
		}
		folderIDStr := strings.TrimSpace(r.URL.Query().Get("folderId"))
		var folderID *uuid.UUID
		if folderIDStr != "" {
			fid, parseErr := uuid.Parse(folderIDStr)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folderId.")
				return
			}
			folderID = &fid
		}

		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		ext := filepath.Ext(filename)
		fileUUID := uuid.New().String()
		objectKey := fmt.Sprintf("managed-files/%s/%s%s", courseCode, fileUUID, ext)
		cfg := d.effectiveConfig()

		// S3-backed: return presigned PUT URL; client uploads directly then calls confirm endpoint
		if d.Storage != nil {
			if s3d, ok := d.Storage.(*filestorage.S3Driver); ok {
				ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
				if ttl <= 0 {
					ttl = time.Hour
				}
				putURL, putErr := s3d.PresignedPutURL(r.Context(), objectKey, ttl)
				if putErr != nil {
					log.Printf("course-file-item-post: presign key=%q err=%v", objectKey, putErr)
					apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Storage unavailable.")
					return
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"objectKey":       objectKey,
					"presignedPutUrl": putURL,
					"expiresAt":       time.Now().Add(ttl).UTC().Format(time.RFC3339),
					"courseId":        courseID.String(),
					"folderId":        folderIDStringOrNull(folderID),
				})
				return
			}
		}

		// Local driver: receive the body
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		byteSize := r.ContentLength
		if byteSize < 0 {
			byteSize = 0
		}
		if d.Storage != nil {
			if err := d.Storage.PutObject(r.Context(), objectKey, r.Body, r.ContentLength, ct); err != nil {
				log.Printf("course-file-item-post: put key=%q err=%v", objectKey, err)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store file.")
				return
			}
		} else {
			root := strings.TrimSpace(cfg.CourseFilesRoot)
			if root == "" {
				root = "data/course-files"
			}
			p := root + "/" + courseCode + "/" + objectKey
			if writeErr := writeLocalFile(p, r.Body); writeErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store file.")
				return
			}
		}
		item, err := filemanager.CreateFileItem(
			r.Context(), d.Pool, courseID, folderID,
			objectKey, filename, filename, ct, byteSize, &viewer,
		)
		if err != nil {
			log.Printf("course-file-item-post: db insert course=%q err=%v", courseCode, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to register file.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(item)
		broadcastFilesChanged(courseCode)
	}
}

// handlePostCourseFileItemConfirm is POST /api/v1/courses/{course_code}/files/items/confirm
// Called by the client after a successful S3 presigned upload to register the metadata.
func (d Deps) handlePostCourseFileItemConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to upload files.")
			return
		}
		var body struct {
			ObjectKey    string  `json:"objectKey"`
			Filename     string  `json:"filename"`
			MimeType     string  `json:"mimeType"`
			ByteSize     int64   `json:"byteSize"`
			FolderID     *string `json:"folderId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.ObjectKey == "" || body.Filename == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "objectKey and filename are required.")
			return
		}
		if body.MimeType == "" {
			body.MimeType = "application/octet-stream"
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var folderID *uuid.UUID
		if body.FolderID != nil && *body.FolderID != "" {
			fid, parseErr := uuid.Parse(*body.FolderID)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folderId.")
				return
			}
			folderID = &fid
		}
		item, err := filemanager.CreateFileItem(
			r.Context(), d.Pool, courseID, folderID,
			body.ObjectKey, body.Filename, body.Filename, body.MimeType, body.ByteSize, &viewer,
		)
		if err != nil {
			log.Printf("course-file-item-confirm: db insert course=%q err=%v", courseCode, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to register file.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(item)
		broadcastFilesChanged(courseCode)
	}
}

// handlePatchCourseFileItem is PATCH /api/v1/courses/{course_code}/files/items/{item_id}
// Supports rename (displayName) and move (folderId).
func (d Deps) handlePatchCourseFileItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit files.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		var body struct {
			DisplayName *string `json:"displayName"`
			FolderID    *string `json:"folderId"` // empty string = move to root
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if body.DisplayName != nil {
			name := strings.TrimSpace(*body.DisplayName)
			if name == "" || len(name) > 255 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Display name must be 1–255 characters.")
				return
			}
			item, renameErr := filemanager.RenameFileItem(r.Context(), d.Pool, courseID, itemID, name)
			if renameErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to rename file.")
				return
			}
			if item == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(item)
			broadcastFilesChanged(courseCode)
			return
		}
		if body.FolderID != nil {
			var folderID *uuid.UUID
			if *body.FolderID != "" {
				fid, parseErr := uuid.Parse(*body.FolderID)
				if parseErr != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid folderId.")
					return
				}
				folderID = &fid
			}
			item, moveErr := filemanager.MoveFileItem(r.Context(), d.Pool, courseID, itemID, folderID)
			if moveErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to move file.")
				return
			}
			if item == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(item)
			broadcastFilesChanged(courseCode)
			return
		}
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide displayName or folderId to update.")
	}
}

// handleDeleteCourseFileItem is DELETE /api/v1/courses/{course_code}/files/items/{item_id}
func (d Deps) handleDeleteCourseFileItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to delete files.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		storageKey, deleted, err := filemanager.DeleteFileItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete file.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		storage := d.Storage
		cfg := d.effectiveConfig()
		go func() {
			if storage != nil {
				if delErr := storage.DeleteObject(context.Background(), storageKey); delErr != nil {
					log.Printf("course-file-item-delete: blob delete key=%q err=%v", storageKey, delErr)
				}
			} else {
				root := strings.TrimSpace(cfg.CourseFilesRoot)
				if root == "" {
					root = "data/course-files"
				}
				_ = deleteLocalFile(root + "/" + courseCode + "/" + storageKey)
			}
		}()
		w.WriteHeader(http.StatusNoContent)
		broadcastFilesChanged(courseCode)
	}
}

// handleGetCourseFileItemContent is GET /api/v1/courses/{course_code}/files/items/{item_id}/content
func (d Deps) handleGetCourseFileItemContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		item, err := filemanager.GetFileItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load file.")
			return
		}
		if item == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		cfg := d.effectiveConfig()
		if d.Storage != nil {
			ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
			if ttl <= 0 {
				ttl = time.Hour
			}
			presignURL, presignErr := d.Storage.GetPresignedURL(r.Context(), item.StorageKey, ttl)
			if presignErr != nil && !errors.Is(presignErr, filestorage.ErrNoPresignedURL) {
				log.Printf("course-file-item-content: presign key=%q err=%v", item.StorageKey, presignErr)
				apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "File temporarily unavailable — try again in a moment.")
				return
			}
			if presignURL != "" {
				http.Redirect(w, r, presignURL, http.StatusFound)
				return
			}
			rc, getErr := d.Storage.GetObject(r.Context(), item.StorageKey)
			if getErr == nil {
				defer func() { _ = rc.Close() }()
				ct := strings.TrimSpace(item.MimeType)
				if ct == "" {
					ct = "application/octet-stream"
				}
				w.Header().Set("Content-Type", ct)
				w.Header().Set("Cache-Control", "private, max-age=86400")
				_, _ = io.Copy(w, rc)
				return
			}
		}
		// Legacy on-disk layout when Storage is nil (courseCode prefix before object key).
		root := strings.TrimSpace(cfg.CourseFilesRoot)
		if root == "" {
			root = "data/course-files"
		}
		legacyPath := filepath.Join(root, courseCode, item.StorageKey)
		if _, statErr := os.Stat(legacyPath); statErr == nil {
			http.ServeFile(w, r, legacyPath)
			return
		}
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
	}
}

func folderIDStringOrNull(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return id.String()
}

func deleteLocalFile(path string) error {
	return os.Remove(path)
}
