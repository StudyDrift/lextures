package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const accountGenerateAvatarPromptMax = 4000

// account avatar URLs must fit the account PATCH limit (normalizeAvatarURL).
const accountGenerateAvatarURLMax = 2_000_000

// handlePostSettingsAccountGenerateAvatar is POST /api/v1/settings/account/generate-avatar.
// Body: {"prompt":"..."}. Response: {"imageUrl":"..."}.
// The avatar is not saved; the client applies it when the user saves the account.
func (d Deps) handlePostSettingsAccountGenerateAvatar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
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
		if len(prompt) > accountGenerateAvatarPromptMax {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is too long.")
			return
		}

		orgID := d.orgIDPtrForUser(r.Context(), userID)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		modelAlias := string(aiprovider.AliasImageGeneration)
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureAvatarGeneration, modelAlias, prompt) {
			return
		}

		got, callMeta, err := d.aiProviderResolver().CreateImage(r.Context(), orgID, prompt, aiprovider.ImageOptions{N: 1})
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
			dataURL, derr := avatarDataURL(got.B64JSON[0])
			if derr != nil {
				writeAIGenerationFailed(w, r, "Image generation failed.", derr)
				return
			}
			imageURL = dataURL
		}
		if !validAccountAvatarURL(imageURL) {
			writeAIGenerationFailed(w, r, "The image provider did not return an image.", nil)
			return
		}

		gwDec := aigateway.Decision{UserIDHash: aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID), OptInConfirmed: true}
		d.logAIInferenceAllowedWithProvider(r, userID, aigateway.FeatureAvatarGeneration, callMeta.ModelID, string(callMeta.Provider), prompt, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: userID, Feature: aigateway.FeatureAvatarGeneration, Model: callMeta.ModelID,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"imageUrl": imageURL})
	}
}

func validAccountAvatarURL(raw string) bool {
	t := strings.TrimSpace(raw)
	if t == "" || len(t) > accountGenerateAvatarURLMax {
		return false
	}
	if strings.HasPrefix(t, "https://") || strings.HasPrefix(t, "http://") {
		return true
	}
	return strings.HasPrefix(t, "data:image/")
}

func avatarDataURL(b64 string) (string, error) {
	raw, err := decodeImageB64(b64)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("generated image is empty")
	}
	mimeType := http.DetectContentType(raw)
	if !strings.HasPrefix(mimeType, "image/") {
		return "", fmt.Errorf("generated payload is not an image")
	}
	// Re-encode so the data URL matches the detected type and the account save limit.
	encoded := base64.StdEncoding.EncodeToString(raw)
	url := "data:" + mimeType + ";base64," + encoded
	if len(url) > accountGenerateAvatarURLMax {
		return "", fmt.Errorf("generated avatar is too large")
	}
	return url, nil
}
