package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// Matches course.course_files.byte_size CHECK (migration 294).
const canvasMaxImportedSubmissionFileBytes = 524288000 // 500 MB

// Only prefetch small attachments in parallel; larger files stream during per-submission import.
const canvasPrefetchSubmissionAttachmentBytes = 10 << 20

// canvasAssignmentSubmissionImportDeps carries blob + DB context for importing submission bodies/attachments.
type canvasAssignmentSubmissionImportDeps struct {
	CourseCode     string
	ImporterUserID uuid.UUID
	FilesRoot      string
	Storage        filestorage.Driver
}

func canvasAssignmentSubmissionImportable(sub map[string]any) bool {
	if sub == nil {
		return false
	}
	if canvasSubmissionHasContent(sub) {
		return true
	}
	state := strings.ToLower(strings.TrimSpace(strAt(sub, "workflow_state", "")))
	switch state {
	case "submitted", "pending_review", "graded":
		return true
	default:
		return false
	}
}

func canvasSubmissionPayloadHasContent(sub map[string]any) bool {
	if sub == nil {
		return false
	}
	if strings.TrimSpace(strAt(sub, "body", "")) != "" {
		return true
	}
	if strings.TrimSpace(strAt(sub, "url", "")) != "" {
		return true
	}
	return len(arrAt(sub, "attachments")) > 0
}

// canvasEffectiveSubmissionPayload prefers top-level Canvas fields, then the latest history row with content.
func canvasEffectiveSubmissionPayload(sub map[string]any) map[string]any {
	if sub == nil {
		return nil
	}
	if canvasSubmissionPayloadHasContent(sub) {
		return sub
	}
	hist := arrAt(sub, "submission_history")
	for i := len(hist) - 1; i >= 0; i-- {
		if hm := hist[i]; canvasSubmissionPayloadHasContent(hm) {
			return hm
		}
	}
	return sub
}

func canvasSubmissionHasContent(sub map[string]any) bool {
	return canvasSubmissionPayloadHasContent(canvasEffectiveSubmissionPayload(sub))
}

func canvasSubmissionTextForImport(sub map[string]any) (string, bool) {
	sub = canvasEffectiveSubmissionPayload(sub)
	if sub == nil {
		return "", false
	}
	body := strings.TrimSpace(strAt(sub, "body", ""))
	url := strings.TrimSpace(strAt(sub, "url", ""))
	if body == "" && url == "" {
		return "", false
	}
	var b strings.Builder
	if body != "" {
		text := markdownFromHTML(body)
		if text == "" {
			text = body
		}
		b.WriteString(text)
	}
	if url != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("URL: ")
		b.WriteString(url)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", false
	}
	return out, true
}

func canvasFirstSubmissionAttachment(sub map[string]any) map[string]any {
	atts := arrAt(canvasEffectiveSubmissionPayload(sub), "attachments")
	if len(atts) == 0 {
		return nil
	}
	return atts[0]
}

func canvasAttachmentByteSize(att map[string]any) int64 {
	if att == nil {
		return 0
	}
	return int64At(att, "size")
}

func canvasSubmissionSubmittedAt(sub map[string]any) time.Time {
	if t := canvasTimeAt(sub, "submitted_at"); t != nil {
		return *t
	}
	if t := canvasTimeAt(sub, "graded_at"); t != nil {
		return *t
	}
	return time.Now().UTC()
}

func canvasDownloadCanvasURL(
	ctx context.Context,
	client *http.Client,
	downloadURL, accessToken string,
) ([]byte, string, error) {
	if client == nil || strings.TrimSpace(downloadURL) == "" {
		return nil, "", fmt.Errorf("missing download client or url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, canvasMaxImportedSubmissionFileBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > canvasMaxImportedSubmissionFileBytes {
		return nil, "", fmt.Errorf("attachment exceeds %d byte limit", canvasMaxImportedSubmissionFileBytes)
	}
	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct == "" {
		ct = "application/octet-stream"
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return data, ct, nil
}

func canvasSubmissionAttachmentFilename(att map[string]any) string {
	return strAt(att, "filename", strAt(att, "display_name", "submission"))
}

func canvasSubmissionAttachmentMimeType(att map[string]any, responseContentType string) string {
	mimeType := strAt(att, "content-type", "application/octet-stream")
	ct := strings.TrimSpace(responseContentType)
	if ct != "" {
		if i := strings.Index(ct, ";"); i >= 0 {
			ct = strings.TrimSpace(ct[:i])
		}
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = ct
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return mimeType
}

func canvasSubmissionAttachmentStorageKey(courseCode, filename, mimeType string) (storageKey, resolvedFilename string) {
	ext := filepath.Ext(filename)
	if ext == "" {
		switch mimeType {
		case "text/plain":
			ext = ".txt"
		case "text/markdown":
			ext = ".md"
		default:
			ext = ".bin"
		}
		filename += ext
	}
	return fmt.Sprintf("submissions/import/%s/%s%s", courseCode, uuid.New().String(), ext), filename
}

func canvasStreamAndStoreSubmissionAttachment(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	accessToken string,
	deps canvasAssignmentSubmissionImportDeps,
	courseID uuid.UUID,
	att map[string]any,
) (*uuid.UUID, error) {
	downloadURL := strAt(att, "url", "")
	if downloadURL == "" {
		return nil, nil
	}
	filename := canvasSubmissionAttachmentFilename(att)
	mimeType := strAt(att, "content-type", "application/octet-stream")
	declaredSize := canvasAttachmentByteSize(att)
	if declaredSize > canvasMaxImportedSubmissionFileBytes {
		log.Printf("canvas-import: skip submission attachment %q: declared size %d exceeds %d byte limit",
			filename, declaredSize, canvasMaxImportedSubmissionFileBytes)
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}

	contentLen := resp.ContentLength
	if contentLen > canvasMaxImportedSubmissionFileBytes {
		log.Printf("canvas-import: skip submission attachment %q: content-length %d exceeds %d byte limit",
			filename, contentLen, canvasMaxImportedSubmissionFileBytes)
		return nil, nil
	}
	mimeType = canvasSubmissionAttachmentMimeType(att, resp.Header.Get("Content-Type"))

	storageKey, filename := canvasSubmissionAttachmentStorageKey(deps.CourseCode, filename, mimeType)
	root := strings.TrimSpace(deps.FilesRoot)
	if root == "" {
		root = "data/course-files"
	}

	byteSize := contentLen
	if byteSize <= 0 {
		byteSize = declaredSize
	}

	limitedBody := io.LimitReader(resp.Body, canvasMaxImportedSubmissionFileBytes+1)
	if deps.Storage != nil {
		if byteSize <= 0 {
			data, readErr := io.ReadAll(limitedBody)
			if readErr != nil {
				return nil, readErr
			}
			if int64(len(data)) > canvasMaxImportedSubmissionFileBytes {
				log.Printf("canvas-import: skip submission attachment %q: downloaded %d bytes exceeds limit", filename, len(data))
				return nil, nil
			}
			return canvasStoreImportedSubmissionBlob(ctx, tx, deps, courseID, filename, mimeType, data)
		}
		if err := deps.Storage.PutObject(ctx, storageKey, limitedBody, byteSize, mimeType); err != nil {
			return nil, err
		}
	} else {
		p := coursefiles.BlobDiskPath(root, deps.CourseCode, storageKey)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		n, copyErr := io.Copy(f, limitedBody)
		closeErr := f.Close()
		if copyErr != nil {
			_ = os.Remove(p)
			return nil, copyErr
		}
		if closeErr != nil {
			_ = os.Remove(p)
			return nil, closeErr
		}
		if n > canvasMaxImportedSubmissionFileBytes {
			_ = os.Remove(p)
			log.Printf("canvas-import: skip submission attachment %q: downloaded %d bytes exceeds limit", filename, n)
			return nil, nil
		}
		byteSize = n
	}

	fileID, err := coursefiles.CreateInTransaction(
		ctx, tx, courseID, deps.ImporterUserID,
		storageKey, filename, mimeType, byteSize,
	)
	if err != nil {
		return nil, err
	}
	return &fileID, nil
}

func canvasStoreImportedSubmissionBlob(
	ctx context.Context,
	tx pgx.Tx,
	deps canvasAssignmentSubmissionImportDeps,
	courseID uuid.UUID,
	filename, mimeType string,
	data []byte,
) (*uuid.UUID, error) {
	if len(data) == 0 || len(data) > canvasMaxImportedSubmissionFileBytes {
		return nil, nil
	}
	storageKey, filename := canvasSubmissionAttachmentStorageKey(deps.CourseCode, filename, mimeType)
	root := strings.TrimSpace(deps.FilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	if deps.Storage != nil {
		if err := deps.Storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
			return nil, err
		}
	} else {
		p := coursefiles.BlobDiskPath(root, deps.CourseCode, storageKey)
		if err := writeLocalFile(p, bytes.NewReader(data)); err != nil {
			return nil, err
		}
	}
	fileID, err := coursefiles.CreateInTransaction(
		ctx, tx, courseID, deps.ImporterUserID,
		storageKey, filename, mimeType, int64(len(data)),
	)
	if err != nil {
		return nil, err
	}
	return &fileID, nil
}

type canvasPrefetchedSubmissionAttachment struct {
	filename string
	mimeType string
	data     []byte
}

func canvasPrefetchSubmissionAttachmentsParallel(
	ctx context.Context,
	client *http.Client,
	accessToken string,
	subs []map[string]any,
) map[int64]canvasPrefetchedSubmissionAttachment {
	out := make(map[int64]canvasPrefetchedSubmissionAttachment)
	type job struct {
		canvasUserID int64
		sub          map[string]any
	}
	jobs := make([]job, 0, len(subs))
	for _, sub := range subs {
		canvasUserID := int64At(sub, "user_id")
		if canvasUserID <= 0 || canvasFirstSubmissionAttachment(sub) == nil {
			continue
		}
		jobs = append(jobs, job{canvasUserID: canvasUserID, sub: sub})
	}
	if len(jobs) == 0 {
		return out
	}
	var mu sync.Mutex
	g, gctx := canvasImportParallelGroup(ctx, len(jobs))
	for _, j := range jobs {
		j := j
		g.Go(func() error {
			att := canvasFirstSubmissionAttachment(j.sub)
			if att == nil {
				return nil
			}
			size := canvasAttachmentByteSize(att)
			if size <= 0 || size > canvasPrefetchSubmissionAttachmentBytes {
				return nil
			}
			downloadURL := strAt(att, "url", "")
			if downloadURL == "" {
				return nil
			}
			filename := strAt(att, "filename", strAt(att, "display_name", "submission"))
			mimeType := strAt(att, "content-type", "application/octet-stream")
			data, ct, err := canvasDownloadCanvasURL(gctx, client, downloadURL, accessToken)
			if err != nil || len(data) == 0 {
				return nil
			}
			if mimeType == "" || mimeType == "application/octet-stream" {
				mimeType = ct
			}
			mu.Lock()
			out[j.canvasUserID] = canvasPrefetchedSubmissionAttachment{
				filename: filename,
				mimeType: mimeType,
				data:     data,
			}
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}

func canvasImportOneAssignmentSubmission(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	accessToken string,
	deps canvasAssignmentSubmissionImportDeps,
	courseID, moduleItemID, studentID uuid.UUID,
	sub map[string]any,
	prefetched *canvasPrefetchedSubmissionAttachment,
) error {
	if !canvasAssignmentSubmissionImportable(sub) {
		return nil
	}
	submittedAt := canvasSubmissionSubmittedAt(sub)
	var attachmentFileID *uuid.UUID

	if prefetched != nil && len(prefetched.data) > 0 {
		if id, storeErr := canvasStoreImportedSubmissionBlob(ctx, tx, deps, courseID, prefetched.filename, prefetched.mimeType, prefetched.data); storeErr != nil {
			return storeErr
		} else if id != nil {
			attachmentFileID = id
		}
	} else if att := canvasFirstSubmissionAttachment(sub); att != nil {
		if id, storeErr := canvasStreamAndStoreSubmissionAttachment(ctx, tx, client, accessToken, deps, courseID, att); storeErr != nil {
			return storeErr
		} else if id != nil {
			attachmentFileID = id
		}
	}

	if attachmentFileID == nil {
		if text, ok := canvasSubmissionTextForImport(sub); ok {
			filename := fmt.Sprintf("submission-%s.txt", studentID.String())
			id, err := canvasStoreImportedSubmissionBlob(ctx, tx, deps, courseID, filename, "text/plain", []byte(text))
			if err != nil {
				return err
			}
			attachmentFileID = id
		}
	}

	return moduleassignmentsubmissions.UpsertImportedInTransaction(
		ctx, tx, courseID, moduleItemID, studentID, attachmentFileID, submittedAt,
	)
}
