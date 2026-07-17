package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/telemetry"
	"github.com/lextures/lextures/server/internal/workers/avscan"
)

func boardPostJSON(p board.Post, courseCode string, avEnabled bool) map[string]any {
	return boardPostJSONWithAttribution(p, courseCode, avEnabled, board.AttributionNamed, board.Capabilities{CanManage: true})
}

func boardPostJSONWithAttribution(p board.Post, courseCode string, avEnabled bool, attribution string, caps board.Capabilities) map[string]any {
	status := p.Status
	if status == "" {
		status = board.PostStatusApproved
	}
	out := map[string]any{
		"id":          p.ID,
		"boardId":     p.BoardID,
		"contentType": p.ContentType,
		"title":       p.Title,
		"sortIndex":   p.SortIndex,
		"status":      status,
		"hidden":      p.Hidden,
		"removed":     p.Removed,
		"createdAt":   p.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":   p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	applyAuthorVisibility(out, p.AuthorID, p.GuestDisplayName, attribution, caps)
	if len(p.Body) > 0 {
		out["body"] = json.RawMessage(p.Body)
	}
	if p.LinkURL != nil {
		out["linkUrl"] = *p.LinkURL
	}
	if len(p.LinkPreview) > 0 {
		out["linkPreview"] = json.RawMessage(p.LinkPreview)
	}
	if len(p.DrawingData) > 0 {
		out["drawingData"] = json.RawMessage(p.DrawingData)
	}
	if p.SectionID != nil {
		out["sectionId"] = *p.SectionID
	}
	if len(p.Position) > 0 {
		out["position"] = json.RawMessage(p.Position)
	}
	if p.EventDate != nil {
		out["eventDate"] = p.EventDate.UTC().Format(time.RFC3339)
	}
	if p.Lat != nil {
		out["lat"] = *p.Lat
	}
	if p.Lng != nil {
		out["lng"] = *p.Lng
	}
	if p.Attachment != nil {
		out["attachment"] = attachmentJSON(*p.Attachment, courseCode, p.BoardID, avEnabled)
	}
	return out
}

func attachmentJSON(a board.Attachment, courseCode, boardID string, avEnabled bool) map[string]any {
	m := map[string]any{
		"id":         a.ID,
		"fileName":   a.FileName,
		"mimeType":   a.MimeType,
		"sizeBytes":  a.SizeBytes,
		"altText":    a.AltText,
		"scanStatus": a.ScanStatus,
	}
	if board.AttachmentAccessible(a, avEnabled) {
		m["url"] = fmt.Sprintf(
			"/api/v1/courses/%s/boards/%s/attachments/%s/content",
			courseCode, boardID, a.ID,
		)
	} else {
		m["url"] = nil
	}
	return m
}

func (d Deps) canEditBoardPost(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID, p *board.Post) bool {
	if p.AuthorID != nil && *p.AuthorID == viewer.String() {
		return true
	}
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return false
	}
	return true
}

// handleListBoardPosts is GET .../boards/{board_id}/posts
func (d Deps) handleListBoardPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		posts, err := board.ListPosts(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list posts.")
			return
		}
		posts = board.FilterVisiblePosts(posts, viewer.String(), caps.CanManage)
		eng, err := board.LoadPostEngagements(r.Context(), d.Pool, courseCode, boardID, viewer, b.ReactionMode, caps.CanManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load engagement stats.")
			return
		}
		avOn := d.effectiveConfig().AvScanningEnabled
		out := make([]map[string]any, 0, len(posts))
		for _, p := range posts {
			row := boardPostJSONWithAttribution(p, courseCode, avOn, b.Attribution, caps)
			mergePostEngagement(row, eng[p.ID], b.ReactionMode)
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"posts": out})
	}
}

// handleCreateBoardPost is POST .../boards/{board_id}/posts
func (d Deps) handleCreateBoardPost() http.HandlerFunc {
	type reqBody struct {
		ContentType  string          `json:"contentType"`
		Title        string          `json:"title"`
		Body         json.RawMessage `json:"body"`
		LinkURL      string          `json:"linkUrl"`
		DrawingData  json.RawMessage `json:"drawingData"`
		AttachmentID *string         `json:"attachmentId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanPost {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to post on this board.")
			return
		}
		if d.writeGateReject(w, board.CheckWriteAllowed(b, caps.CanManage, board.WritePost, time.Now().UTC())) {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		matched, term, okFilter := d.screenBoardText(w, r, b, &viewer, in.Title, in.Body)
		if !okFilter {
			return
		}
		createIn := board.CreatePostInput{
			ContentType:  in.ContentType,
			Title:        in.Title,
			Body:         in.Body,
			LinkURL:      in.LinkURL,
			DrawingData:  in.DrawingData,
			AttachmentID: in.AttachmentID,
			Status:       resolveInitialPostStatus(b, caps.CanManage),
		}
		var preview *board.LinkPreview
		ct := strings.ToLower(strings.TrimSpace(in.ContentType))
		if (ct == board.ContentTypeLink || ct == board.ContentTypeVideo) && strings.TrimSpace(in.LinkURL) != "" {
			if p, uerr := board.FetchLinkPreview(r.Context(), in.LinkURL); uerr == nil {
				preview = p
			} else if errors.Is(uerr, board.ErrUnfurlSSRF) {
				// Still allow create with bare link; skip cached preview.
				preview = nil
			}
		}
		created, err := board.CreatePost(r.Context(), d.Pool, courseCode, boardID, viewer, createIn, preview)
		if err != nil {
			if strings.HasPrefix(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			log.Printf("board-post-create: %v", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create post.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		if matched {
			d.flagFilterHit(r.Context(), courseCode, b, created.ID, term)
		}
		if created.Status == board.PostStatusPending {
			d.notifyBoardManagers(r.Context(), courseCode, boardID, "board_moderation_pending",
				"Board post awaiting approval",
				"A new post is waiting for approval on a board.")
		}
		if created.Attachment != nil && created.Attachment.ScanStatus == board.ScanBlocked {
			d.flagAVBlockedAttachment(r.Context(), courseCode, b, created.ID)
		}
		telemetry.RecordBusinessEvent("board.post.created")
		notifyBoardPeers(r.Context(), boardID, "post.created", created.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardPostJSONWithAttribution(*created, courseCode, d.effectiveConfig().AvScanningEnabled, b.Attribution, caps))
	}
}

// handleGetBoardPost is GET .../posts/{post_id}
func (d Deps) handleGetBoardPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		p, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if p == nil || !board.PostVisibleToViewer(*p, viewer.String(), caps.CanManage) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		eng, err := board.LoadPostEngagements(r.Context(), d.Pool, courseCode, boardID, viewer, b.ReactionMode, caps.CanManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load engagement stats.")
			return
		}
		out := boardPostJSONWithAttribution(*p, courseCode, d.effectiveConfig().AvScanningEnabled, b.Attribution, caps)
		mergePostEngagement(out, eng[p.ID], b.ReactionMode)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePatchBoardPost is PATCH .../posts/{post_id}
func (d Deps) handlePatchBoardPost() http.HandlerFunc {
	type reqBody struct {
		Title       *string         `json:"title"`
		Body        json.RawMessage `json:"body"`
		LinkURL     *string         `json:"linkUrl"`
		DrawingData json.RawMessage `json:"drawingData"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		existing, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		if !d.canEditBoardPost(w, r, courseCode, viewer, existing) {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := board.PatchPostInput{
			Title:       in.Title,
			Body:        in.Body,
			LinkURL:     in.LinkURL,
			DrawingData: in.DrawingData,
		}
		if in.LinkURL != nil && strings.TrimSpace(*in.LinkURL) != "" {
			if p, uerr := board.FetchLinkPreview(r.Context(), *in.LinkURL); uerr == nil {
				patch.LinkPreview = p
			}
		}
		updated, err := board.PatchPost(r.Context(), d.Pool, courseCode, boardID, postID, patch)
		if err != nil {
			if strings.HasPrefix(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update post.")
			return
		}
		notifyBoardPeers(r.Context(), boardID, "post.updated", postID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardPostJSON(*updated, courseCode, d.effectiveConfig().AvScanningEnabled))
	}
}

// handleDeleteBoardPost is DELETE .../posts/{post_id}
func (d Deps) handleDeleteBoardPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		existing, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		if !d.canEditBoardPost(w, r, courseCode, viewer, existing) {
			return
		}
		okDel, err := board.DeletePost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete post.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.post.deleted")
		notifyBoardPeers(r.Context(), boardID, "post.deleted", postID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCreateBoardAttachment is POST .../boards/{board_id}/attachments
// Accepts multipart/form-data (file + optional altText) or JSON for presign init.
func (d Deps) handleCreateBoardAttachment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil || b == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		ctHeader := strings.TrimSpace(r.Header.Get("Content-Type"))
		if strings.HasPrefix(ctHeader, "multipart/form-data") {
			d.handleBoardAttachmentMultipart(w, r, courseCode, boardID, viewer)
			return
		}
		d.handleBoardAttachmentPresign(w, r, courseCode, boardID, viewer)
	}
}

func (d Deps) handleBoardAttachmentMultipart(w http.ResponseWriter, r *http.Request, courseCode, boardID string, viewer uuid.UUID) {
	const maxParse = 220 << 20
	if err := r.ParseMultipartForm(maxParse); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form or file too large.")
		return
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing 'file' part.")
		return
	}
	defer func() { _ = f.Close() }()

	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	hint := strings.TrimSpace(r.FormValue("contentType"))
	if !board.AllowedMimeForContent(hint, mimeType) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File type is not allowed.")
		return
	}
	maxBytes := board.MaxBytesForMime(mimeType)
	if header.Size <= 0 || header.Size > maxBytes {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File exceeds size limit.")
		return
	}
	altText := strings.TrimSpace(r.FormValue("altText"))
	ext := filepath.Ext(header.Filename)
	fileUUID := uuid.New().String()
	storageKey := fmt.Sprintf("boards/%s/%s/%s%s", courseCode, boardID, fileUUID, ext)

	cfg := d.effectiveConfig()
	cid, _ := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	tenantID, orgErr := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
	if orgErr != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify storage quota.")
		return
	}
	if d.StorageQuota != nil && header.Size > 0 {
		violation, qErr := d.StorageQuota.CheckAndReserve(r.Context(), tenantID, cid, viewer, header.Size)
		if qErr != nil {
			log.Printf("board-attachment: quota check err=%v", qErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify storage quota.")
			return
		}
		if violation != nil {
			telemetry.RecordBusinessEvent("board.attachment.quota_rejected")
			apierr.WriteJSON(w, http.StatusForbidden, CodeQuotaExceeded,
				"Storage limit reached. Ask an administrator to increase the course storage quota.")
			return
		}
	}

	storage := d.Storage
	if storage == nil {
		root := strings.TrimSpace(cfg.CourseFilesRoot)
		if root == "" {
			root = "data/course-files"
		}
		storage = &filestorage.LocalDriver{Root: root}
	}
	if perr := storage.PutObject(r.Context(), storageKey, f, header.Size, mimeType); perr != nil {
		log.Printf("board-attachment: PutObject key=%s err=%v", storageKey, perr)
		if d.StorageQuota != nil && header.Size > 0 {
			_ = d.StorageQuota.Release(r.Context(), tenantID, cid, viewer, header.Size)
		}
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store file.")
		return
	}

	scanStatus := board.ScanClean
	uploader := viewer
	if cfg.AvScanningEnabled {
		scanStatus = board.ScanPending
		if cid != nil {
			bucket := cfg.StorageBucket
			_, regErr := avscan.RegisterAndEnqueue(r.Context(), d.Pool, tenantID, cid,
				storageKey, bucket, mimeType, header.Size, &uploader, true)
			if regErr != nil {
				log.Printf("board-attachment: av register err=%v", regErr)
			}
		}
	} else if cid != nil {
		if _, upsErr := storageobjects.Upsert(r.Context(), d.Pool, tenantID, cid,
			storageKey, cfg.StorageBucket, mimeType, header.Size, &uploader, false); upsErr != nil {
			log.Printf("board-attachment: storageobjects upsert err=%v", upsErr)
		}
	}

	att, err := board.CreateAttachment(r.Context(), d.Pool, courseCode, boardID, viewer,
		storageKey, header.Filename, mimeType, altText, scanStatus, header.Size)
	if err != nil {
		log.Printf("board-attachment: db err=%v", err)
		if d.StorageQuota != nil && header.Size > 0 {
			_ = d.StorageQuota.Release(r.Context(), tenantID, cid, viewer, header.Size)
		}
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record attachment.")
		return
	}
	if att == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(attachmentJSON(*att, courseCode, boardID, cfg.AvScanningEnabled))
}

func (d Deps) handleBoardAttachmentPresign(w http.ResponseWriter, r *http.Request, courseCode, boardID string, viewer uuid.UUID) {
	type reqBody struct {
		FileName    string `json:"fileName"`
		MimeType    string `json:"mimeType"`
		SizeBytes   int64  `json:"sizeBytes"`
		AltText     string `json:"altText"`
		ContentType string `json:"contentType"`
	}
	var in reqBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return
	}
	if strings.TrimSpace(in.FileName) == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "fileName is required.")
		return
	}
	mimeType := strings.TrimSpace(in.MimeType)
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(in.FileName))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !board.AllowedMimeForContent(in.ContentType, mimeType) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File type is not allowed.")
		return
	}
	maxBytes := board.MaxBytesForMime(mimeType)
	if in.SizeBytes <= 0 || in.SizeBytes > maxBytes {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File exceeds size limit.")
		return
	}
	ext := filepath.Ext(in.FileName)
	storageKey := fmt.Sprintf("boards/%s/%s/%s%s", courseCode, boardID, uuid.New().String(), ext)
	cfg := d.effectiveConfig()
	cid, _ := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	tenantID, orgErr := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
	if orgErr != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify storage quota.")
		return
	}
	if d.StorageQuota != nil && in.SizeBytes > 0 {
		violation, qErr := d.StorageQuota.CheckAndReserve(r.Context(), tenantID, cid, viewer, in.SizeBytes)
		if qErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify storage quota.")
			return
		}
		if violation != nil {
			telemetry.RecordBusinessEvent("board.attachment.quota_rejected")
			apierr.WriteJSON(w, http.StatusForbidden, CodeQuotaExceeded,
				"Storage limit reached. Ask an administrator to increase the course storage quota.")
			return
		}
	}

	var putURL string
	var expiresAt string
	if s3d, ok := d.Storage.(*filestorage.S3Driver); ok {
		ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		u, err := s3d.PresignedPutURL(r.Context(), storageKey, ttl)
		if err != nil {
			if d.StorageQuota != nil && in.SizeBytes > 0 {
				_ = d.StorageQuota.Release(r.Context(), tenantID, cid, viewer, in.SizeBytes)
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Storage unavailable.")
			return
		}
		putURL = u
		expiresAt = time.Now().Add(ttl).UTC().Format(time.RFC3339)
	} else if d.Storage == nil {
		if d.StorageQuota != nil && in.SizeBytes > 0 {
			_ = d.StorageQuota.Release(r.Context(), tenantID, cid, viewer, in.SizeBytes)
		}
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Storage unavailable.")
		return
	}

	scanStatus := board.ScanClean
	uploader := viewer
	if cfg.AvScanningEnabled {
		scanStatus = board.ScanPending
		if cid != nil {
			bucket := cfg.StorageBucket
			_, regErr := avscan.RegisterAndEnqueue(r.Context(), d.Pool, tenantID, cid,
				storageKey, bucket, mimeType, in.SizeBytes, &uploader, true)
			if regErr != nil {
				log.Printf("board-attachment: av register err=%v", regErr)
			}
		}
	} else if cid != nil {
		if _, upsErr := storageobjects.Upsert(r.Context(), d.Pool, tenantID, cid,
			storageKey, cfg.StorageBucket, mimeType, in.SizeBytes, &uploader, false); upsErr != nil {
			log.Printf("board-attachment: storageobjects upsert err=%v", upsErr)
		}
	}
	att, err := board.CreateAttachment(r.Context(), d.Pool, courseCode, boardID, viewer,
		storageKey, in.FileName, mimeType, in.AltText, scanStatus, in.SizeBytes)
	if err != nil || att == nil {
		if d.StorageQuota != nil && in.SizeBytes > 0 {
			_ = d.StorageQuota.Release(r.Context(), tenantID, cid, viewer, in.SizeBytes)
		}
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record attachment.")
		return
	}
	out := attachmentJSON(*att, courseCode, boardID, cfg.AvScanningEnabled)
	out["storageKey"] = storageKey
	if putURL != "" {
		out["presignedPutUrl"] = putURL
		out["expiresAt"] = expiresAt
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

// handleBoardAttachmentContent is GET .../attachments/{attachment_id}/content
func (d Deps) handleBoardAttachmentContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		attachmentID := chi.URLParam(r, "attachment_id")
		att, err := board.GetAttachment(r.Context(), d.Pool, courseCode, boardID, attachmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load attachment.")
			return
		}
		if att == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attachment not found.")
			return
		}
		avOn := d.effectiveConfig().AvScanningEnabled
		if !board.AttachmentAccessible(*att, avOn) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Attachment is not available (scanning or blocked).")
			return
		}
		if d.Storage == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Storage unavailable.")
			return
		}
		// Prefer short-lived redirect when driver supports it.
		if url, err := d.Storage.GetPresignedURL(r.Context(), att.StorageKey, 15*time.Minute); err == nil && url != "" {
			http.Redirect(w, r, url, http.StatusFound)
			return
		} else if err != nil && !errors.Is(err, filestorage.ErrNoPresignedURL) {
			log.Printf("board-attachment-content: presign err=%v", err)
		}
		rc, err := d.Storage.GetObject(r.Context(), att.StorageKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found in storage.")
			return
		}
		defer func() { _ = rc.Close() }()
		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.FileName))
		_, _ = io.Copy(w, rc)
	}
}

// handleBoardLinkPreview is POST .../boards/{board_id}/link-preview
func (d Deps) handleBoardLinkPreview() http.HandlerFunc {
	type reqBody struct {
		URL string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil || b == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		preview, err := board.FetchLinkPreview(r.Context(), in.URL)
		if err != nil {
			if errors.Is(err, board.ErrUnfurlSSRF) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "URL blocked by SSRF policy.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		out := map[string]any{
			"url":         strings.TrimSpace(in.URL),
			"title":       preview.Title,
			"description": preview.Description,
			"image":       preview.Image,
			"siteName":    preview.SiteName,
			"fetchedAt":   preview.FetchedAt,
		}
		if yt := board.YouTubeVideoID(in.URL); yt != "" {
			out["provider"] = "youtube"
			out["embedId"] = yt
		} else if vm := board.VimeoVideoID(in.URL); vm != "" {
			out["provider"] = "vimeo"
			out["embedId"] = vm
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
