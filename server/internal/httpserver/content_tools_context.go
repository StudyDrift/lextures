package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	ctctx "github.com/lextures/lextures/server/internal/service/contenttools/context"
)

func (d Deps) registerContentToolsContextRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/context/sources", d.handleContentToolsContextSourcesList())
	r.Post("/api/v1/courses/{course_code}/content-tools/context/sources/{source_id}/reingest", d.handleContentToolsContextSourceReingest())
	r.Patch("/api/v1/courses/{course_code}/content-tools/context/sources/{source_id}", d.handleContentToolsContextSourcePatch())
	r.Post("/api/v1/courses/{course_code}/content-tools/context/preview", d.handleContentToolsContextPreview())
}

func (d Deps) handleContentToolsContextSourcesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		itemRaw := strings.TrimSpace(r.URL.Query().Get("itemId"))
		itemID, err := uuid.Parse(itemRaw)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "itemId is required.")
			return
		}
		rows, err := ctrepo.ListActivitySources(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list sources.")
			return
		}
		// Discover from current page body so the panel shows links even before first pack build.
		instIDRaw := strings.TrimSpace(r.URL.Query().Get("instanceId"))
		if instIDRaw != "" {
			if instID, perr := uuid.Parse(instIDRaw); perr == nil {
				_, _ = ctctx.Build(r.Context(), d.Pool, instID, ctctx.BuildOpts{
					EnqueueIngest: true,
					TokenBudget:   ctctx.DefaultRequestContextTokens,
				})
				rows, _ = ctrepo.ListActivitySources(r.Context(), d.Pool, courseID, itemID)
			}
		}
		items := make([]ctmodel.ContextSource, 0, len(rows))
		totalTokens := 0
		for _, row := range rows {
			title := ""
			if row.Title != nil {
				title = *row.Title
			}
			errMsg := ""
			if row.Error != nil {
				errMsg = *row.Error
			}
			text := ""
			if row.ExtractedText != nil {
				text = *row.ExtractedText
				totalTokens += ctctx.EstimateTokens(text)
			}
			quality := ""
			if text != "" {
				n := len([]rune(text))
				switch {
				case n >= 400:
					quality = "high"
				case n >= 80:
					quality = "medium"
				default:
					quality = "low"
				}
			}
			host := ""
			if u, uerr := url.Parse(row.URL); uerr == nil {
				host = u.Hostname()
			}
			items = append(items, ctmodel.ContextSource{
				ID:                row.ID,
				SourceID:          row.SourceID,
				URL:               row.URL,
				Title:             title,
				Host:              host,
				Origin:            row.Origin,
				Status:            row.Status,
				Error:             errMsg,
				FetchedAt:         row.FetchedAt,
				ByteSize:          row.ByteSize,
				Excluded:          row.Excluded,
				ExtractedText:     text,
				ExtractionQuality: quality,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ctmodel.ContextSourcesResponse{Items: items, TotalTokens: totalTokens})
	}
}

func (d Deps) handleContentToolsContextSourceReingest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if !d.contentToolsRateLimit(w, r, viewer, "context_reingest", 10) {
			return
		}
		sourceRowID, err := uuid.Parse(chi.URLParam(r, "source_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid source id.")
			return
		}
		row, err := ctrepo.GetActivitySource(r.Context(), d.Pool, courseID, sourceRowID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load source.")
			return
		}
		if row == nil || row.SourceID == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Source not found.")
			return
		}
		_ = ctrepo.MarkLinkSourcePending(r.Context(), d.Pool, *row.SourceID)
		settings, _ := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		policy := ctctx.IngestPolicy{Mode: ctctx.IngestPublic}
		if settings != nil {
			policy.Mode = settings.LinkIngestionMode
			policy.Allowlist = settings.LinkHostAllowlist
		}
		src, err := ctctx.IngestSource(r.Context(), d.Pool, *row.SourceID, policy)
		if err != nil && src == nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Re-ingest failed.")
			return
		}
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, nil, nil, &actor, "_context", "link_reingest", map[string]any{
			"activitySourceId": sourceRowID.String(),
			"sourceId":         row.SourceID.String(),
		})
		updated, _ := ctrepo.GetActivitySource(r.Context(), d.Pool, courseID, sourceRowID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(activitySourceToAPI(updated))
	}
}

func (d Deps) handleContentToolsContextSourcePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		sourceRowID, err := uuid.Parse(chi.URLParam(r, "source_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid source id.")
			return
		}
		var body ctmodel.ContextSourcePatch
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Excluded == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "excluded is required.")
			return
		}
		if err := ctrepo.SetActivitySourceExcluded(r.Context(), d.Pool, courseID, sourceRowID, *body.Excluded, viewer); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Source not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update source.")
			return
		}
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, nil, nil, &actor, "_context", "link_exclude", map[string]any{
			"activitySourceId": sourceRowID.String(),
			"excluded":         *body.Excluded,
		})
		updated, _ := ctrepo.GetActivitySource(r.Context(), d.Pool, courseID, sourceRowID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(activitySourceToAPI(updated))
	}
}

func (d Deps) handleContentToolsContextPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if !d.contentToolsRateLimit(w, r, viewer, "context_preview", 20) {
			return
		}
		var body ctmodel.ContextPreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InstanceID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "instanceId is required.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, body.InstanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		pack, err := ctctx.Build(r.Context(), d.Pool, body.InstanceID, ctctx.BuildOpts{
			Query:         body.Query,
			EnqueueIngest: true,
			TokenBudget:   ctctx.DefaultRequestContextTokens,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build context pack.")
			return
		}
		out := contextPackToAPI(pack)
		// Exercise CT.6 AI rails (gateway + budget + PII + tool-calling/fallback) on dry-run.
		if strings.TrimSpace(body.Query) != "" {
			orgID, _ := ctrepo.CourseOrgID(r.Context(), d.Pool, courseID)
			call, cerr := ctctx.RunMediatedCall(r.Context(), d.Pool, ctctx.CallOpts{
				InstanceID:  body.InstanceID,
				CourseID:    courseID,
				OrgID:       orgID,
				UserID:      viewer,
				ToolID:      inst.ToolID,
				FeatureID:   aigateway.FeatureContentTool,
				TaskPrompt:  "Answer briefly using only the grounded sources. Cite by id.",
				LearnerText: body.Query,
				Model:       "dry-run",
				Completer:   aiprovider.DryRunToolCallingCompleter{},
				GatewayCfg:  aigateway.Config{DisclosureEnabled: false},
				BuildOpts: ctctx.BuildOpts{
					Query:         body.Query,
					EnqueueIngest: false,
					TokenBudget:   ctctx.DefaultRequestContextTokens,
				},
			})
			if cerr == nil && call != nil {
				out.DryRunAnswer = call.Text
				out.RedactedQuery = call.RedactedIn
				for _, c := range call.Citations {
					out.DryRunCitations = append(out.DryRunCitations, ctmodel.CitationPublic{
						Kind: string(c.Kind), ID: c.ID, Title: c.Title, URL: c.URL, Loc: c.Loc,
					})
				}
			} else if cerr != nil {
				// Surface typed denials without failing the pack preview.
				var ge *ctctx.GatewayError
				var be *ctctx.BudgetError
				switch {
				case errors.As(cerr, &ge):
					out.DryRunAnswer = ge.Error()
				case errors.As(cerr, &be):
					out.DryRunAnswer = be.Error()
				default:
					out.DryRunAnswer = cerr.Error()
				}
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func activitySourceToAPI(row *ctrepo.ActivitySourceRow) ctmodel.ContextSource {
	if row == nil {
		return ctmodel.ContextSource{}
	}
	title := ""
	if row.Title != nil {
		title = *row.Title
	}
	errMsg := ""
	if row.Error != nil {
		errMsg = *row.Error
	}
	text := ""
	if row.ExtractedText != nil {
		text = *row.ExtractedText
	}
	host := ""
	if u, err := url.Parse(row.URL); err == nil {
		host = u.Hostname()
	}
	return ctmodel.ContextSource{
		ID:            row.ID,
		SourceID:      row.SourceID,
		URL:           row.URL,
		Title:         title,
		Host:          host,
		Origin:        row.Origin,
		Status:        row.Status,
		Error:         errMsg,
		FetchedAt:     row.FetchedAt,
		ByteSize:      row.ByteSize,
		Excluded:      row.Excluded,
		ExtractedText: text,
	}
}

func contextPackToAPI(pack *ctctx.ContextPack) ctmodel.ContextPack {
	if pack == nil {
		return ctmodel.ContextPack{Segments: []ctmodel.ContextSegment{}, PendingSources: []ctmodel.ContextPendingSource{}}
	}
	segs := make([]ctmodel.ContextSegment, 0, len(pack.Segments))
	for _, s := range pack.Segments {
		segs = append(segs, ctmodel.ContextSegment{
			Kind: string(s.Kind), ID: s.ID, Title: s.Title, URL: s.URL, Lang: s.Lang, Text: s.Text, Tokens: s.Tokens,
		})
	}
	pending := make([]ctmodel.ContextPendingSource, 0, len(pack.PendingSources))
	for _, p := range pack.PendingSources {
		pending = append(pending, ctmodel.ContextPendingSource{URL: p.URL, Status: p.Status, Reason: p.Reason})
	}
	return ctmodel.ContextPack{
		InstanceID:     pack.InstanceID,
		Segments:       segs,
		PendingSources: pending,
		TotalTokens:    pack.TotalTokens,
		VariantID:      pack.VariantID,
	}
}
