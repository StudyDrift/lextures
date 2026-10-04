package httpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const courseGenerateImagePromptMax = 4000
const courseGenerateImageMaxBytes = 10 * 1024 * 1024

const courseImageGenerationUnavailableMsg = "Image generation is not available for the configured AI provider."

// handlePostCourseGenerateImage is POST /api/v1/courses/{course_code}/generate-image
// Body: {"prompt":"..."}. Response: {"imageUrl":"https://..."}.
// The image is not saved as the course hero; the client applies it with PUT hero-image.
func (d Deps) handlePostCourseGenerateImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		hasAccess, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return
		}
		if !hasAccess {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		hasPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		var body struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		if len(prompt) > courseGenerateImagePromptMax {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is too long.")
			return
		}

		orgID := d.orgIDPtrForUser(r.Context(), userID)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		modelAlias := string(aiprovider.AliasImageGeneration)
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureHeroImageGeneration, modelAlias, prompt) {
			return
		}

		resolver := d.aiProviderResolver()
		got, callMeta, err := resolver.CreateImage(r.Context(), orgID, prompt, aiprovider.ImageOptions{N: 1})
		if err != nil {
			if imageGenNotConfigured(err) {
				apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
				return
			}
			if imageGenUnavailable(err) {
				writeAIGenerationFailed(w, r, courseImageGenerationUnavailableMsg, err)
				return
			}
			writeAIGenerationFailed(w, r, "Image generation failed.", err)
			return
		}

		imageURL := firstGeneratedImageURL(got)
		if imageURL == "" && len(got.B64JSON) > 0 {
			stored, serr := d.persistGeneratedCourseImage(r.Context(), courseCode, userID, got.B64JSON[0])
			if serr != nil {
				writeAIGenerationFailed(w, r, "Image generation failed.", serr)
				return
			}
			imageURL = stored
		}
		if imageURL == "" || !validHeroImageURL(imageURL) {
			writeAIGenerationFailed(w, r, "The image provider did not return an image.", nil)
			return
		}

		courseID, _ := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		gwDec := aigateway.Decision{UserIDHash: aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID), OptInConfirmed: true}
		d.logAIInferenceAllowedWithProvider(r, userID, aigateway.FeatureHeroImageGeneration, callMeta.ModelID, string(callMeta.Provider), prompt, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: userID, CourseID: courseID, CourseCode: courseCode, Feature: aigateway.FeatureHeroImageGeneration, Model: callMeta.ModelID,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"imageUrl": imageURL})
	}
}

func imageGenNotConfigured(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not configured")
}

func imageGenUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not available") || strings.Contains(msg, "not supported")
}

func firstGeneratedImageURL(got aiprovider.ImageResult) string {
	for _, u := range got.URLs {
		u = strings.TrimSpace(u)
		if u != "" {
			return u
		}
	}
	return ""
}

func (d Deps) persistGeneratedCourseImage(ctx context.Context, courseCode string, userID uuid.UUID, b64 string) (string, error) {
	raw, err := decodeImageB64(b64)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw) > courseGenerateImageMaxBytes {
		return "", fmt.Errorf("generated image is empty or too large")
	}
	mimeType := http.DetectContentType(raw)
	if !strings.HasPrefix(mimeType, "image/") {
		return "", fmt.Errorf("generated payload is not an image")
	}
	ext := imageExt(mimeType)
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || cid == nil {
		return "", fmt.Errorf("failed to load course")
	}
	fileUUID := uuid.New().String()
	storageKey := fmt.Sprintf("files/%s/%s%s", courseCode, fileUUID, ext)
	reader := bytes.NewReader(raw)
	cfg := d.effectiveConfig()
	if d.Storage != nil {
		if perr := d.Storage.PutObject(ctx, storageKey, reader, int64(len(raw)), mimeType); perr != nil {
			log.Printf("course-generate-image: PutObject key=%s err=%v", storageKey, perr)
			return "", fmt.Errorf("failed to store image")
		}
	} else {
		root := strings.TrimSpace(cfg.CourseFilesRoot)
		if root == "" {
			root = "data/course-files"
		}
		p := coursefiles.BlobDiskPath(root, courseCode, storageKey)
		if werr := writeLocalFile(p, reader); werr != nil {
			log.Printf("course-generate-image: local write key=%s err=%v", storageKey, werr)
			return "", fmt.Errorf("failed to store image")
		}
	}
	fileID, err := coursefiles.Create(ctx, d.Pool, *cid, userID, storageKey, "hero-generated"+ext, mimeType, int64(len(raw)))
	if err != nil {
		log.Printf("course-generate-image: db insert err=%v", err)
		return "", fmt.Errorf("failed to record image")
	}
	return fmt.Sprintf("/api/v1/courses/%s/course-files/%s/content", courseCode, fileID.String()), nil
}

func decodeImageB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 && strings.HasPrefix(strings.ToLower(s[:i]), "data:") {
		s = s[i+1:]
	}
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil {
		return raw, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid generated image")
	}
	return raw, nil
}

func imageExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}
