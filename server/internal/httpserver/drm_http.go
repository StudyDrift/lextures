package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	drmrepo "github.com/lextures/lextures/server/internal/repos/drm"
	drmservice "github.com/lextures/lextures/server/internal/service/drm"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/watermark"
	"github.com/jackc/pgx/v5/pgxpool"
)

// handlePostFileLicense is POST /api/v1/files/:object_id/license
// Grants (or denies) access to a DRM-protected file and returns a user-bound token.
// For watermark_only PDFs, streams the watermarked PDF bytes directly.
func (d Deps) handlePostFileLicense() http.HandlerFunc {
	type response struct {
		Granted  bool   `json:"granted"`
		DRMType  string `json:"drmType"`
		Token    string `json:"token,omitempty"`
		MediaURL string `json:"mediaUrl,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		if d.DRM == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "DRM feature is not enabled.")
			return
		}

		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object id.")
			return
		}

		ipAddr := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddr = strings.Split(forwarded, ",")[0]
		}
		ipAddr = strings.TrimSpace(strings.Split(ipAddr, ":")[0])

		result, err := d.DRM.RequestLicense(r.Context(), objectID, viewer, ipAddr)
		if err != nil {
			log.Printf("drm-license: request err object=%s user=%s: %v", objectID, viewer, err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "License request failed.")
			return
		}

		if result.AnomalyDetected {
			log.Printf("drm-anomaly: object=%s user=%s exceeded threshold", objectID, viewer)
			// TODO: emit admin alert email (FR-7 full alert delivery requires mail service)
		}

		if !result.Granted {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, result.DenialReason)
			return
		}

		// For watermark_only PDFs, apply the watermark and stream the result.
		if result.DRMType == drmrepo.DRMTypeWatermark {
			d.serveWatermarkedPDF(w, r, objectID, viewer, result.Token)
			return
		}

		// For all other types (none, widevine stub, fairplay stub): return a signed token.
		cfg := d.effectiveConfig()
		ttl := time.Duration(cfg.StoragePresignTTL) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		var mediaURL string
		if d.Storage != nil {
			obj, objErr := drmrepo.GetObjectDRM(r.Context(), d.Pool, objectID)
			if objErr == nil && obj != nil {
				mediaURL, _ = d.Storage.GetPresignedURL(r.Context(), obj.ObjectKey, ttl)
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response{
			Granted:  true,
			DRMType:  string(result.DRMType),
			Token:    result.Token,
			MediaURL: mediaURL,
		})
	}
}

// serveWatermarkedPDF fetches the PDF from storage, stamps it with the user's identity,
// and writes the watermarked bytes to w.
func (d Deps) serveWatermarkedPDF(w http.ResponseWriter, r *http.Request, objectID, userID uuid.UUID, token string) {
	obj, err := drmrepo.GetObjectDRM(r.Context(), d.Pool, objectID)
	if err != nil || obj == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
		return
	}

	// Resolve user display name for the watermark label.
	displayName, nameErr := resolveDisplayName(r.Context(), d.Pool, userID)
	if nameErr != nil || displayName == "" {
		displayName = userID.String() // fallback
	}

	var pdfBytes []byte

	if d.Storage != nil {
		if ld, ok := d.Storage.(*filestorage.LocalDriver); ok {
			pdfBytes, err = ld.ReadFile(obj.ObjectKey)
		} else {
			// For S3/R2: temporarily fall back to presigned redirect until GetObject is added.
			ttl := time.Hour
			if cfg := d.effectiveConfig(); cfg.StoragePresignTTL > 0 {
				ttl = time.Duration(cfg.StoragePresignTTL) * time.Second
			}
			presignURL, presignErr := d.Storage.GetPresignedURL(r.Context(), obj.ObjectKey, ttl)
			if presignErr == nil && presignURL != "" {
				http.Redirect(w, r, presignURL+"&wm_token="+token, http.StatusFound)
				return
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "File temporarily unavailable.")
			return
		}
	} else {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Storage not configured.")
		return
	}

	if err != nil {
		log.Printf("drm-watermark: read pdf key=%q err=%v", obj.ObjectKey, err)
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
		return
	}

	p := watermark.Params{
		UserDisplayName: displayName,
		AccessedAt:      time.Now().UTC(),
	}
	var stamped bytes.Buffer
	if wmErr := watermark.WatermarkPDF(bytes.NewReader(pdfBytes), &stamped, p); wmErr != nil {
		log.Printf("drm-watermark: stamp err key=%q user=%s err=%v", obj.ObjectKey, userID, wmErr)
		// FR-6 reliability: fall back to unwatermarked for non-hard-DRM resources.
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-DRM-Watermark-Error", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-DRM-Type", "watermark_only")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(stamped.Bytes())
}

// handlePutAdminFileDRM is PUT /api/v1/admin/files/:object_id/drm
// Allows admins to set the DRM type and configuration on a storage object.
func (d Deps) handlePutAdminFileDRM() http.HandlerFunc {
	type requestBody struct {
		DRMType    string  `json:"drmType"`
		DRMKeyID   *string `json:"drmKeyId,omitempty"`
		DRMProvider *string `json:"drmProvider,omitempty"`
	}
	type response struct {
		ObjectID    string  `json:"objectId"`
		DRMType     string  `json:"drmType"`
		DRMKeyID    *string `json:"drmKeyId,omitempty"`
		DRMProvider *string `json:"drmProvider,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object id.")
			return
		}
		var body requestBody
		if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		dt := drmrepo.DRMType(body.DRMType)
		switch dt {
		case drmrepo.DRMTypeNone, drmrepo.DRMTypeWatermark, drmrepo.DRMTypeWidevine, drmrepo.DRMTypeFairPlay:
			// valid
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid drmType.")
			return
		}
		if setErr := drmrepo.SetObjectDRM(r.Context(), d.Pool, objectID, dt, body.DRMKeyID, body.DRMProvider); setErr != nil {
			log.Printf("drm-admin-set: object=%s err=%v", objectID, setErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update DRM settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response{
			ObjectID:    objectID.String(),
			DRMType:     string(dt),
			DRMKeyID:    body.DRMKeyID,
			DRMProvider: body.DRMProvider,
		})
	}
}

// handleGetAdminDRMAnomalies is GET /api/v1/admin/drm/anomalies
// Returns (user, object) pairs where download count exceeded the threshold in the last hour.
func (d Deps) handleGetAdminDRMAnomalies() http.HandlerFunc {
	type response struct {
		Anomalies []drmrepo.Anomaly `json:"anomalies"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		anomalies, err := drmservice.ListAnomalies(r.Context(), d.Pool)
		if err != nil {
			log.Printf("drm-anomalies: list err=%v", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load anomalies.")
			return
		}
		if anomalies == nil {
			anomalies = []drmrepo.Anomaly{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response{Anomalies: anomalies})
	}
}

// resolveDisplayName returns the user's display name (or first+last fallback) from the DB.
func resolveDisplayName(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var displayName, firstName, lastName *string
	err := pool.QueryRow(ctx, `
		SELECT display_name, first_name, last_name
		FROM "user".users
		WHERE id = $1
	`, userID).Scan(&displayName, &firstName, &lastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if displayName != nil && *displayName != "" {
		return *displayName, nil
	}
	parts := []string{}
	if firstName != nil && *firstName != "" {
		parts = append(parts, *firstName)
	}
	if lastName != nil && *lastName != "" {
		parts = append(parts, *lastName)
	}
	return strings.Join(parts, " "), nil
}
