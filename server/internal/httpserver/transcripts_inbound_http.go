package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/transcriptinbound"
)

func (d Deps) registerTranscriptInboundRoutes(r chi.Router) {
	r.Post("/api/v1/integrations/transcripts/inbound", d.handleIntegrationsTranscriptInbound())
	r.Get("/api/v1/admin/transcripts/inbound", d.handleAdminListTranscriptInbound())
	r.Get("/api/v1/admin/transcripts/inbound/{id}", d.handleAdminGetTranscriptInbound())
	r.Get("/api/v1/admin/transcripts/inbound/{id}/courses", d.handleAdminTranscriptInboundCourses())
	r.Get("/api/v1/admin/transcripts/inbound/{id}/original", d.handleAdminTranscriptInboundOriginal())
	r.Post("/api/v1/admin/transcripts/inbound/{id}/match", d.handleAdminMatchTranscriptInbound())
	r.Post("/api/v1/admin/transcripts/inbound/{id}/unmatch", d.handleAdminUnmatchTranscriptInbound())
	r.Post("/api/v1/admin/transcripts/inbound/{id}/accept", d.handleAdminAcceptTranscriptInbound())
	r.Post("/api/v1/admin/transcripts/inbound/{id}/reject", d.handleAdminRejectTranscriptInbound())
	r.Get("/api/v1/me/transcripts/inbound", d.handleMeListTranscriptInbound())
}

func (d Deps) transcriptInboundFeatureOff(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFTranscripts || !cfg.FFTranscriptInbound {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Transcript inbound is not enabled.")
		return true
	}
	return false
}

type inboundDocJSON struct {
	ID                string          `json:"id"`
	OrgID             string          `json:"orgId"`
	Channel           string          `json:"channel"`
	SourceName        *string         `json:"sourceName,omitempty"`
	ExternalRef       *string         `json:"externalRef,omitempty"`
	Format            string          `json:"format"`
	ContentHash       string          `json:"contentHash"`
	ContentType       *string         `json:"contentType,omitempty"`
	ByteSize          int             `json:"byteSize"`
	Parsed            json.RawMessage `json:"parsed,omitempty"`
	StudentName       *string         `json:"studentName,omitempty"`
	StudentDOB        *string         `json:"studentDob,omitempty"`
	StudentRef        *string         `json:"studentRef,omitempty"`
	MatchedUserID     *string         `json:"matchedUserId,omitempty"`
	MatchConfidence   *float64        `json:"matchConfidence,omitempty"`
	MatchDetail       json.RawMessage `json:"matchDetail,omitempty"`
	Status            string          `json:"status"`
	NeedsManualReview bool            `json:"needsManualReview"`
	ReviewerID        *string         `json:"reviewerId,omitempty"`
	RejectReason      *string         `json:"rejectReason,omitempty"`
	QuarantineReason  *string         `json:"quarantineReason,omitempty"`
	ReceivedAt        string          `json:"receivedAt"`
	ProcessedAt       *string         `json:"processedAt,omitempty"`
}

func inboundToJSON(d *transcriptsrepo.InboundDocument) inboundDocJSON {
	out := inboundDocJSON{
		ID:                d.ID.String(),
		OrgID:             d.OrgID.String(),
		Channel:           d.Channel,
		SourceName:        d.SourceName,
		ExternalRef:       d.ExternalRef,
		Format:            d.Format,
		ContentHash:       d.ContentHash,
		ContentType:       d.ContentType,
		ByteSize:          d.ByteSize,
		Parsed:            d.Parsed,
		StudentName:       d.StudentName,
		StudentDOB:        d.StudentDOB,
		StudentRef:        d.StudentRef,
		MatchConfidence:   d.MatchConfidence,
		MatchDetail:       d.MatchDetail,
		Status:            d.Status,
		NeedsManualReview: d.NeedsManualReview,
		RejectReason:      d.RejectReason,
		QuarantineReason:  d.QuarantineReason,
		ReceivedAt:        d.ReceivedAt.UTC().Format(time.RFC3339),
	}
	if d.MatchedUserID != nil {
		s := d.MatchedUserID.String()
		out.MatchedUserID = &s
	}
	if d.ReviewerID != nil {
		s := d.ReviewerID.String()
		out.ReviewerID = &s
	}
	if d.ProcessedAt != nil {
		s := d.ProcessedAt.UTC().Format(time.RFC3339)
		out.ProcessedAt = &s
	}
	return out
}

type inboundEventJSON struct {
	ID        string          `json:"id"`
	EventType string          `json:"eventType"`
	ActorID   *string         `json:"actorId,omitempty"`
	Detail    json.RawMessage `json:"detail,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

// POST /api/v1/integrations/transcripts/inbound — HMAC peer intake.
func (d Deps) handleIntegrationsTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, transcriptinbound.MaxInboundBytes+1024))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read request body.")
			return
		}
		secret := ""
		if cfg.WebhookSecret != nil {
			secret = strings.TrimSpace(*cfg.WebhookSecret)
		}
		if !verifyTranscriptsHoldHMAC(r, secret, body) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid webhook signature.")
			return
		}

		in, err := parseInboundReceiveBody(r, body)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		res, err := transcriptinbound.Receive(r.Context(), d.Pool, in)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		status := http.StatusCreated
		if res.Duplicate {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document":  inboundToJSON(res.Document),
			"duplicate": res.Duplicate,
		})
	}
}

type inboundJSONBody struct {
	OrgID          string `json:"orgId"`
	Channel        string `json:"channel"`
	SourceName     string `json:"sourceName"`
	ExternalRef    string `json:"externalRef"`
	Format         string `json:"format"`
	ContentType    string `json:"contentType"`
	ContentBase64  string `json:"contentBase64"`
	StudentName    string `json:"studentName"`
	StudentDOB     string `json:"studentDob"`
	StudentRef     string `json:"studentRef"`
}

func parseInboundReceiveBody(r *http.Request, body []byte) (transcriptinbound.ReceiveInput, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	// Multipart is uncommon with HMAC-over-raw-body; prefer JSON envelope or raw XML/PDF.
	if strings.HasPrefix(ct, "application/json") {
		var payload inboundJSONBody
		if err := json.Unmarshal(body, &payload); err != nil {
			return transcriptinbound.ReceiveInput{}, errors.New("invalid JSON body")
		}
		orgID, err := uuid.Parse(strings.TrimSpace(payload.OrgID))
		if err != nil {
			return transcriptinbound.ReceiveInput{}, errors.New("orgId is required")
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.ContentBase64))
		if err != nil || len(raw) == 0 {
			return transcriptinbound.ReceiveInput{}, errors.New("contentBase64 is required")
		}
		channel := strings.TrimSpace(payload.Channel)
		if channel == "" {
			channel = transcriptsrepo.InboundChannelAPIPeer
		}
		return transcriptinbound.ReceiveInput{
			OrgID:       orgID,
			Channel:     channel,
			SourceName:  payload.SourceName,
			ExternalRef: payload.ExternalRef,
			Format:      payload.Format,
			ContentType: firstNonEmptyStr(payload.ContentType, ct),
			RawBytes:    raw,
			StudentName: payload.StudentName,
			StudentDOB:  payload.StudentDOB,
			StudentRef:  payload.StudentRef,
		}, nil
	}

	orgRaw := strings.TrimSpace(r.URL.Query().Get("orgId"))
	if orgRaw == "" {
		orgRaw = strings.TrimSpace(r.Header.Get("X-Lextures-Org-Id"))
	}
	orgID, err := uuid.Parse(orgRaw)
	if err != nil {
		return transcriptinbound.ReceiveInput{}, errors.New("orgId query/header is required for raw body intake")
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		channel = transcriptsrepo.InboundChannelAPIPeer
	}
	return transcriptinbound.ReceiveInput{
		OrgID:       orgID,
		Channel:     channel,
		SourceName:  r.URL.Query().Get("sourceName"),
		ExternalRef: r.URL.Query().Get("externalRef"),
		Format:      r.URL.Query().Get("format"),
		ContentType: ct,
		RawBytes:    body,
		StudentName: r.URL.Query().Get("studentName"),
		StudentDOB:  r.URL.Query().Get("studentDob"),
		StudentRef:  r.URL.Query().Get("studentRef"),
	}, nil
}

func (d Deps) handleAdminListTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		f := transcriptsrepo.InboundListFilter{
			Status: strings.TrimSpace(r.URL.Query().Get("status")),
			Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		}
		if orgRaw := strings.TrimSpace(r.URL.Query().Get("orgId")); orgRaw != "" {
			if id, err := uuid.Parse(orgRaw); err == nil {
				f.OrgID = &id
			}
		}
		if lim := strings.TrimSpace(r.URL.Query().Get("limit")); lim != "" {
			if n, err := strconv.Atoi(lim); err == nil {
				f.Limit = n
			}
		}
		list, err := transcriptsrepo.ListInboundDocuments(r.Context(), d.Pool, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load inbound queue.")
			return
		}
		out := make([]inboundDocJSON, 0, len(list))
		for i := range list {
			out = append(out, inboundToJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"documents": out})
	}
}

func (d Deps) handleAdminGetTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		doc, err := transcriptsrepo.GetInboundDocument(r.Context(), d.Pool, id)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		events, _ := transcriptsrepo.ListInboundEvents(r.Context(), d.Pool, id)
		evJSON := make([]inboundEventJSON, 0, len(events))
		for _, e := range events {
			row := inboundEventJSON{
				ID:        e.ID.String(),
				EventType: e.EventType,
				Detail:    e.Detail,
				CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
			}
			if e.ActorID != nil {
				s := e.ActorID.String()
				row.ActorID = &s
			}
			evJSON = append(evJSON, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": inboundToJSON(doc),
			"events":   evJSON,
		})
	}
}

func (d Deps) handleAdminTranscriptInboundCourses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		doc, err := transcriptsrepo.GetInboundDocument(r.Context(), d.Pool, id)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		courses, err := transcriptinbound.CoursesFromParsed(doc.Parsed)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Parsed course data unavailable.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"documentId": doc.ID.String(),
			"status":     doc.Status,
			"courses":    courses,
			"parsed":     doc.Parsed,
		})
	}
}

func (d Deps) handleAdminTranscriptInboundOriginal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		doc, err := transcriptsrepo.GetInboundDocument(r.Context(), d.Pool, id)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		ct := transcriptinbound.SniffContentType(doc.Format, doc.RawBytes)
		if doc.ContentType != nil && strings.TrimSpace(*doc.ContentType) != "" {
			ct = *doc.ContentType
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Disposition", "inline; filename=\"inbound-"+doc.ID.String()+"\"")
		w.Header().Set("X-Content-Hash", doc.ContentHash)
		_, _ = w.Write(doc.RawBytes)
	}
}

func (d Deps) handleAdminMatchTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		var body struct {
			UserID string `json:"userId"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(body.UserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "userId is required.")
			return
		}
		doc, err := transcriptsrepo.MatchInboundDocument(r.Context(), d.Pool, id, userID, &actorID, 1.0, map[string]any{
			"manual": true,
		})
		if err != nil {
			writeInboundError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": inboundToJSON(doc)})
	}
}

func (d Deps) handleAdminUnmatchTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
		doc, err := transcriptsrepo.ClearInboundMatch(r.Context(), d.Pool, id, &actorID, body.Reason)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": inboundToJSON(doc)})
	}
}

func (d Deps) handleAdminAcceptTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		doc, err := transcriptinbound.Accept(r.Context(), d.Pool, id, &actorID)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": inboundToJSON(doc)})
	}
}

func (d Deps) handleAdminRejectTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		doc, err := transcriptsrepo.RejectInboundDocument(r.Context(), d.Pool, id, &actorID, body.Reason)
		if err != nil {
			writeInboundError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": inboundToJSON(doc)})
	}
}

// GET /api/v1/me/transcripts/inbound — student view of matched inbound docs.
func (d Deps) handleMeListTranscriptInbound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptInboundFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		list, err := transcriptsrepo.ListInboundForUser(r.Context(), d.Pool, userID, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load inbound transcripts.")
			return
		}
		out := make([]inboundDocJSON, 0, len(list))
		for i := range list {
			row := inboundToJSON(&list[i])
			row.Parsed = nil
			row.MatchDetail = nil
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"documents": out})
	}
}

func writeInboundError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, transcriptsrepo.ErrInboundNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Inbound document not found.")
	case errors.Is(err, transcriptsrepo.ErrInboundDuplicate):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Inbound document already received.")
	case errors.Is(err, transcriptsrepo.ErrInboundInvalidStatus):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Inbound document status does not allow this action.")
	case errors.Is(err, transcriptinbound.ErrTooLarge):
		apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, "Document exceeds size limit.")
	case errors.Is(err, transcriptinbound.ErrUnsupportedType), errors.Is(err, transcriptinbound.ErrUnsafePayload):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Document failed validation and was refused.")
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Inbound operation failed.")
	}
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
