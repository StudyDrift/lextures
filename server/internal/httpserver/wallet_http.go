package httpserver

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/logging"
	walletrepo "github.com/lextures/lextures/server/internal/repos/wallet"
	"github.com/lextures/lextures/server/internal/service/credentialwallet"
)

func (d Deps) walletFeatureOff(w http.ResponseWriter) bool {
	if !credentialwallet.Enabled(d.effectiveConfig()) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential wallet is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerWalletRoutes(r chi.Router) {
	r.Get("/api/v1/me/wallet", d.handleWalletList())
	r.Get("/api/v1/me/wallet/collections", d.handleWalletListCollections())
	r.Post("/api/v1/me/wallet/collections", d.handleWalletCreateCollection())
	r.Get("/api/v1/me/wallet/collections/{id}", d.handleWalletGetCollection())
	r.Put("/api/v1/me/wallet/collections/{id}", d.handleWalletUpdateCollection())
	r.Delete("/api/v1/me/wallet/collections/{id}", d.handleWalletDeleteCollection())
	r.Post("/api/v1/me/wallet/collections/{id}/revoke", d.handleWalletRevokeCollection())
	r.Get("/api/v1/me/wallet/collections/{id}/access", d.handleWalletCollectionAccess())
	r.Post("/api/v1/me/wallet/export", d.handleWalletStartExport())
	r.Get("/api/v1/me/wallet/export/{id}", d.handleWalletGetExport())
	r.Get("/api/v1/me/wallet/export/{id}/download", d.handleWalletDownloadExport())
	r.Get("/api/v1/me/wallet/{itemId}", d.handleWalletGetItem())
	r.Get("/api/v1/wallet/s/{token}", d.handleWalletPublicShare())
}

func walletItemJSON(it walletrepo.Item, webOrigin string) map[string]any {
	out := map[string]any{
		"id":           it.ID.String(),
		"kind":         string(it.Kind),
		"sourceId":     it.SourceID.String(),
		"title":        it.Title,
		"revoked":      it.Revoked,
		"verifyStatus": credentialwallet.VerifyStatus(it),
		"createdAt":    it.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":    it.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if it.Issuer != nil {
		out["issuer"] = *it.Issuer
	}
	if it.IssuedAt != nil {
		out["issuedAt"] = it.IssuedAt.UTC().Format(time.RFC3339)
	}
	if url := credentialwallet.VerifyURL(webOrigin, it); url != "" {
		out["verifyUrl"] = url
	}
	if len(it.Metadata) > 0 {
		var meta map[string]any
		if json.Unmarshal(it.Metadata, &meta) == nil {
			out["metadata"] = meta
		}
	}
	return out
}

func walletCollectionJSON(c walletrepo.Collection, webOrigin string) map[string]any {
	out := map[string]any{
		"id":         c.ID.String(),
		"name":       c.Name,
		"disclosure": string(c.Disclosure),
		"itemIds":    uuidStrings(c.ItemIDs),
		"createdAt":  c.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":  c.UpdatedAt.UTC().Format(time.RFC3339),
		"revoked":    c.RevokedAt != nil,
	}
	if c.ShareToken != nil && c.RevokedAt == nil {
		out["shareToken"] = *c.ShareToken
		origin := strings.TrimRight(strings.TrimSpace(webOrigin), "/")
		out["shareUrl"] = origin + "/wallet/s/" + *c.ShareToken
	}
	if c.ExpiresAt != nil {
		out["expiresAt"] = c.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if c.RevokedAt != nil {
		out["revokedAt"] = c.RevokedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func (d Deps) handleWalletList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		cfg := d.effectiveConfig()
		items, err := credentialwallet.Refresh(r.Context(), d.Pool, cfg, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load wallet.")
			return
		}
		logging.GlobalWalletMetrics.IncView()
		out := make([]map[string]any, 0, len(items))
		for i := range items {
			out = append(out, walletItemJSON(items[i], cfg.PublicWebOrigin))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":      out,
			"alumniNote": "Your credential wallet remains available after enrollment ends.",
		})
	}
}

func (d Deps) handleWalletGetItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		cfg := d.effectiveConfig()
		_, _ = credentialwallet.Refresh(r.Context(), d.Pool, cfg, userID)
		it, err := walletrepo.GetItem(r.Context(), d.Pool, userID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load wallet item.")
			return
		}
		if it == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Wallet item not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(walletItemJSON(*it, cfg.PublicWebOrigin))
	}
}

type createCollectionBody struct {
	Name       string   `json:"name"`
	Disclosure string   `json:"disclosure"`
	ItemIDs    []string `json:"itemIds"`
	Share      *bool    `json:"share"`
	ExpiresAt  *string  `json:"expiresAt"`
}

func parseItemIDs(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*s))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d Deps) handleWalletListCollections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		rows, err := walletrepo.ListCollections(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load collections.")
			return
		}
		cfg := d.effectiveConfig()
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			out = append(out, walletCollectionJSON(rows[i], cfg.PublicWebOrigin))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"collections": out})
	}
}

func (d Deps) handleWalletCreateCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		var body createCollectionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ids, err := parseItemIDs(body.ItemIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid itemIds.")
			return
		}
		if len(ids) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Select at least one credential.")
			return
		}
		expiresAt, err := parseOptionalTime(body.ExpiresAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid expiresAt.")
			return
		}
		cfg := d.effectiveConfig()
		_, _ = credentialwallet.Refresh(r.Context(), d.Pool, cfg, userID)
		share := true
		if body.Share != nil {
			share = *body.Share
		}
		c, err := walletrepo.CreateCollection(r.Context(), d.Pool, walletrepo.CreateCollectionInput{
			UserID:     userID,
			Name:       body.Name,
			Disclosure: credentialwallet.NormalizeDisclosure(body.Disclosure),
			ItemIDs:    ids,
			ExpiresAt:  expiresAt,
			Share:      share,
		})
		if err != nil {
			if errors.Is(err, walletrepo.ErrItemNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "One or more wallet items were not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create collection.")
			return
		}
		logging.GlobalWalletMetrics.IncShareCreated()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(walletCollectionJSON(*c, cfg.PublicWebOrigin))
	}
}

func (d Deps) handleWalletGetCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collection id.")
			return
		}
		c, err := walletrepo.GetCollection(r.Context(), d.Pool, userID, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load collection.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Collection not found.")
			return
		}
		cfg := d.effectiveConfig()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(walletCollectionJSON(*c, cfg.PublicWebOrigin))
	}
}

type updateCollectionBody struct {
	Name         *string  `json:"name"`
	Disclosure   *string  `json:"disclosure"`
	ItemIDs      []string `json:"itemIds"`
	ExpiresAt    *string  `json:"expiresAt"`
	ClearExpiry  *bool    `json:"clearExpiry"`
	EnableShare  *bool    `json:"enableShare"`
}

func (d Deps) handleWalletUpdateCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collection id.")
			return
		}
		var body updateCollectionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := walletrepo.UpdateCollectionInput{
			UserID:      userID,
			CollectionID: id,
			Name:        body.Name,
			EnableShare: body.EnableShare,
		}
		if body.Disclosure != nil {
			d := credentialwallet.NormalizeDisclosure(*body.Disclosure)
			in.Disclosure = &d
		}
		if body.ItemIDs != nil {
			ids, err := parseItemIDs(body.ItemIDs)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid itemIds.")
				return
			}
			in.ItemIDs = &ids
		}
		if body.ClearExpiry != nil && *body.ClearExpiry {
			in.ClearExpiry = true
		} else {
			expiresAt, err := parseOptionalTime(body.ExpiresAt)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid expiresAt.")
				return
			}
			in.ExpiresAt = expiresAt
		}
		c, err := walletrepo.UpdateCollection(r.Context(), d.Pool, in)
		if err != nil {
			if errors.Is(err, walletrepo.ErrItemNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "One or more wallet items were not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update collection.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Collection not found.")
			return
		}
		cfg := d.effectiveConfig()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(walletCollectionJSON(*c, cfg.PublicWebOrigin))
	}
}

func (d Deps) handleWalletDeleteCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collection id.")
			return
		}
		if err := walletrepo.DeleteCollection(r.Context(), d.Pool, userID, id); err != nil {
			if errors.Is(err, walletrepo.ErrCollectionNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Collection not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete collection.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleWalletRevokeCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collection id.")
			return
		}
		c, err := walletrepo.RevokeCollectionShare(r.Context(), d.Pool, userID, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke share link.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Collection or share link not found.")
			return
		}
		logging.GlobalWalletMetrics.IncShareRevoked()
		cfg := d.effectiveConfig()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(walletCollectionJSON(*c, cfg.PublicWebOrigin))
	}
}

func (d Deps) handleWalletCollectionAccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collection id.")
			return
		}
		events, err := walletrepo.ListCollectionAccess(r.Context(), d.Pool, userID, id, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load access history.")
			return
		}
		out := make([]map[string]any, 0, len(events))
		for _, e := range events {
			row := map[string]any{
				"id":        e.ID.String(),
				"result":    e.Result,
				"createdAt": e.CreatedAt.UTC().Format(time.RFC3339),
			}
			if e.RequesterIP != nil {
				row["requesterIp"] = *e.RequesterIP
			}
			if e.RequesterUA != nil {
				row["requesterUa"] = *e.RequesterUA
			}
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"access": out})
	}
}

func (d Deps) handleWalletStartExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		exp, err := walletrepo.CreateExport(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start export.")
			return
		}
		if _, err := background.EnqueueWalletExport(r.Context(), d.Pool, exp.ID); err != nil {
			// Fall back to synchronous build when the queue is unavailable.
			cfg := d.effectiveConfig()
			if procErr := credentialwallet.ProcessExport(r.Context(), d.Pool, cfg, exp.ID); procErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue export.")
				return
			}
			logging.GlobalWalletMetrics.IncExport()
			exp, err = walletrepo.GetExport(r.Context(), d.Pool, userID, exp.ID)
			if err != nil || exp == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export.")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        exp.ID.String(),
			"status":    string(exp.Status),
			"createdAt": exp.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleWalletGetExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid export id.")
			return
		}
		exp, err := walletrepo.GetExport(r.Context(), d.Pool, userID, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export.")
			return
		}
		if exp == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Export not found.")
			return
		}
		out := map[string]any{
			"id":        exp.ID.String(),
			"status":    string(exp.Status),
			"createdAt": exp.CreatedAt.UTC().Format(time.RFC3339),
		}
		if exp.CompletedAt != nil {
			out["completedAt"] = exp.CompletedAt.UTC().Format(time.RFC3339)
		}
		if exp.ErrorMessage != nil {
			out["error"] = *exp.ErrorMessage
		}
		if exp.Status == walletrepo.ExportReady {
			out["downloadPath"] = "/api/v1/me/wallet/export/" + exp.ID.String() + "/download"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleWalletDownloadExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.walletFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid export id.")
			return
		}
		exp, err := walletrepo.GetExport(r.Context(), d.Pool, userID, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export.")
			return
		}
		if exp == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Export not found.")
			return
		}
		if exp.Status != walletrepo.ExportReady || len(exp.ZipBytes) == 0 {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Export is not ready.")
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="credential-wallet.zip"`)
		_, _ = w.Write(exp.ZipBytes)
	}
}

func (d Deps) handleWalletPublicShare() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.walletFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "token"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share link not found.")
			return
		}
		c, err := walletrepo.GetCollectionByShareToken(r.Context(), d.Pool, token)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load share link.")
			return
		}
		ip, ua := clientIPUA(r)
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share link not found.")
			return
		}
		if c.RevokedAt != nil {
			_ = walletrepo.RecordCollectionAccess(r.Context(), d.Pool, c.ID, "revoked", ip, ua)
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "This share link has been revoked.")
			return
		}
		if c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt) {
			_ = walletrepo.RecordCollectionAccess(r.Context(), d.Pool, c.ID, "expired", ip, ua)
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "This share link has expired.")
			return
		}
		items, err := walletrepo.GetItemsByIDs(r.Context(), d.Pool, c.UserID, c.ItemIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credentials.")
			return
		}
		// Preserve collection order.
		byID := make(map[uuid.UUID]walletrepo.Item, len(items))
		for _, it := range items {
			byID[it.ID] = it
		}
		cfg := d.effectiveConfig()
		publicItems := make([]credentialwallet.PublicItem, 0, len(c.ItemIDs))
		for _, id := range c.ItemIDs {
			it, ok := byID[id]
			if !ok {
				continue
			}
			publicItems = append(publicItems, credentialwallet.FilterItem(it, c.Disclosure, cfg.PublicWebOrigin))
		}
		_ = walletrepo.RecordCollectionAccess(r.Context(), d.Pool, c.ID, "ok", ip, ua)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":       c.Name,
			"disclosure": string(c.Disclosure),
			"items":      publicItems,
		})
	}
}

func clientIPUA(r *http.Request) (ip, ua string) {
	ua = r.UserAgent()
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		ip = host
	} else {
		ip = r.RemoteAddr
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])
		}
	}
	return ip, ua
}
