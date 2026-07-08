// Package learnerprofile implements the cross-course learner profile store and derivation engine (LP01).
package learnerprofile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/redisclient"
	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
)

// FacetDeriver derives one facet from source signals.
type FacetDeriver interface {
	Key() string
	Derive(ctx context.Context, userID uuid.UUID) (FacetResult, error)
	MinSignals() int
	Version() int
}

// Service is the LearnerProfileService boundary.
type Service struct {
	Pool     *pgxpool.Pool
	redis    *redisclient.Client
	derivers map[string]FacetDeriver
	locale   string
}

// New returns a Service with the given derivers registered by facet key.
func New(pool *pgxpool.Pool, derivers ...FacetDeriver) *Service {
	m := make(map[string]FacetDeriver, len(derivers))
	for _, d := range derivers {
		if d != nil {
			m[d.Key()] = d
		}
	}
	return &Service{Pool: pool, derivers: m, locale: "en"}
}

// RegisterDeriver adds or replaces a facet deriver (used in tests and LP02+ wiring).
func (s *Service) RegisterDeriver(d FacetDeriver) {
	if s.derivers == nil {
		s.derivers = map[string]FacetDeriver{}
	}
	s.derivers[d.Key()] = d
}

// Health returns a stable heartbeat for wiring tests.
func (s *Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return "learnerprofile:ok", nil
}

// Get returns the caller's profile read model, lazily creating an empty profile shell.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (ProfileView, error) {
	profileID, err := lprepo.EnsureProfile(ctx, s.Pool, userID)
	if err != nil {
		return ProfileView{}, err
	}
	p, err := lprepo.GetProfileByUserID(ctx, s.Pool, userID)
	if err != nil {
		return ProfileView{}, err
	}
	facets, err := lprepo.ListFacets(ctx, s.Pool, profileID)
	if err != nil {
		return ProfileView{}, err
	}
	summaries := make([]FacetSummary, 0, len(facets))
	for _, f := range facets {
		summaries = append(summaries, facetToSummary(f))
	}
	status := "active"
	var lastComputed *time.Time
	if p != nil {
		status = p.Status
		lastComputed = p.LastComputedAt
	}
	if len(summaries) == 0 {
		status = "insufficient_data"
	}
	return ProfileView{
		Status:         status,
		LastComputedAt: lastComputed,
		Facets:         summaries,
	}, nil
}

// GetFacet returns one facet with insights and top evidence.
func (s *Service) GetFacet(ctx context.Context, userID uuid.UUID, facetKey string) (*FacetDetail, error) {
	if _, ok := lprepo.ValidFacetKeys[facetKey]; !ok {
		return nil, lprepo.ErrUnknownFacet
	}
	profileID, err := lprepo.EnsureProfile(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	f, err := lprepo.GetFacet(ctx, s.Pool, profileID, facetKey)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, nil
	}
	insights, err := s.loadInsightViews(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	summary := facetToSummary(*f)
	return &FacetDetail{Facet: summary, Insights: insights}, nil
}

// GetFacetEvidence returns evidence grouped by insight key for drill-down.
func (s *Service) GetFacetEvidence(ctx context.Context, userID uuid.UUID, facetKey string) (map[string][]EvidenceView, error) {
	detail, err := s.GetFacet(ctx, userID, facetKey)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return map[string][]EvidenceView{}, nil
	}
	out := make(map[string][]EvidenceView, len(detail.Insights))
	for _, ins := range detail.Insights {
		out[ins.InsightKey] = ins.Evidence
	}
	return out, nil
}

// RecomputeIncremental recomputes the given facets for one user.
func (s *Service) RecomputeIncremental(ctx context.Context, userID uuid.UUID, changedFacets ...string) error {
	keys := changedFacets
	if len(keys) == 0 {
		keys = s.deriverKeys()
	}
	return s.recompute(ctx, userID, "incremental", keys)
}

// RecomputeAll runs a full recompute for every registered deriver across active learners.
func (s *Service) RecomputeAll(ctx context.Context) error {
	userIDs, err := lprepo.ListActiveUserIDs(ctx, s.Pool, 100000)
	if err != nil {
		return err
	}
	keys := s.deriverKeys()
	for _, userID := range userIDs {
		if err := s.recompute(ctx, userID, "full", keys); err != nil {
			return err
		}
	}
	if err := s.refreshFacetsPopulatedGauge(ctx); err != nil {
		slog.Warn("learner_profile.facets_populated_gauge", "err", err)
	}
	return nil
}

// Pause sets profile status to paused (LP08).
func (s *Service) Pause(ctx context.Context, userID uuid.UUID) error {
	return lprepo.SetProfileStatus(ctx, s.Pool, userID, "paused")
}

// Resume sets profile status to active and schedules a recompute (LP08).
func (s *Service) Resume(ctx context.Context, userID uuid.UUID) error {
	if err := lprepo.SetProfileStatus(ctx, s.Pool, userID, "active"); err != nil {
		return err
	}
	_, err := EnqueueIncremental(ctx, s.Pool, userID)
	return err
}

// Reset deletes all learner profile rows for a user (LP08).
func (s *Service) Reset(ctx context.Context, userID uuid.UUID) error {
	return lprepo.EraseUser(ctx, s.Pool, userID)
}

// Erase removes all learner profile rows for a user (LP01/LP08 erasure hook).
func (s *Service) Erase(ctx context.Context, userID uuid.UUID) error {
	return lprepo.EraseUser(ctx, s.Pool, userID)
}

func (s *Service) deriverKeys() []string {
	out := make([]string, 0, len(s.derivers))
	for k := range s.derivers {
		out = append(out, k)
	}
	return out
}

func (s *Service) recompute(ctx context.Context, userID uuid.UUID, mode string, facetKeys []string) error {
	profileID, err := lprepo.EnsureProfile(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	p, err := lprepo.GetProfileByUserID(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	if p != nil && p.Status == "paused" {
		return nil
	}
	userHash := hashUserID(userID)
	for _, key := range facetKeys {
		d, ok := s.derivers[key]
		if !ok {
			continue
		}
		started := time.Now().UTC()
		result, deriveErr := safeDerive(ctx, d, userID)
		if deriveErr != nil {
			recordRecompute(key, mode, "error", started)
			slog.Warn("learner_profile.recompute_failed",
				"user_hash", userHash, "facet", key, "mode", mode, "err", deriveErr)
			continue
		}
		write := facetResultToWrite(result)
		if err := lprepo.WriteFacet(ctx, s.Pool, profileID, key, write); err != nil {
			recordRecompute(key, mode, "error", started)
			slog.Warn("learner_profile.write_failed",
				"user_hash", userHash, "facet", key, "mode", mode, "err", err)
			continue
		}
		recordRecompute(key, mode, "ok", started)
		slog.Debug("learner_profile.recomputed",
			"user_hash", userHash, "facet", key, "mode", mode,
			"confidence", result.Confidence, "state", result.State)
	}
	s.InvalidateAdaptiveContext(ctx, userID)
	return nil
}

func safeDerive(ctx context.Context, d FacetDeriver, userID uuid.UUID) (result FacetResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("deriver panic: %v", r)
		}
	}()
	return d.Derive(ctx, userID)
}

func facetResultToWrite(r FacetResult) lprepo.FacetWrite {
	summary := r.Summary
	if summary == nil {
		summary = json.RawMessage(`{}`)
	}
	w := lprepo.FacetWrite{
		State:           r.State,
		Summary:         summary,
		Confidence:      r.Confidence,
		ComputedVersion: r.ComputedVersion,
	}
	for _, ins := range r.Insights {
		iw := lprepo.InsightWrite{
			InsightKey:   ins.InsightKey,
			LabelI18nKey: ins.LabelI18nKey,
			Value:        ins.Value,
			Confidence:   ins.Confidence,
			Salience:     ins.Salience,
		}
		for _, ev := range ins.Evidence {
			iw.Evidence = append(iw.Evidence, lprepo.EvidenceWrite{
				SourceKind:       ev.SourceKind,
				SourceTable:      ev.SourceTable,
				CourseID:         ev.CourseID,
				ObservationCount: ev.ObservationCount,
				WindowStart:      ev.WindowStart,
				WindowEnd:        ev.WindowEnd,
				Contribution:     ev.Contribution,
				SampleRefs:       ev.SampleRefs,
			})
		}
		w.Insights = append(w.Insights, iw)
	}
	return w
}

func (s *Service) loadInsightViews(ctx context.Context, facetID uuid.UUID) ([]InsightView, error) {
	insights, err := lprepo.ListInsights(ctx, s.Pool, facetID)
	if err != nil {
		return nil, err
	}
	if len(insights) == 0 {
		return []InsightView{}, nil
	}
	ids := make([]uuid.UUID, len(insights))
	for i, ins := range insights {
		ids[i] = ins.ID
	}
	evMap, err := lprepo.ListEvidenceForInsights(ctx, s.Pool, ids)
	if err != nil {
		return nil, err
	}
	out := make([]InsightView, 0, len(insights))
	for _, ins := range insights {
		evRows := evMap[ins.ID]
		if len(evRows) == 0 {
			continue
		}
		view := InsightView{
			InsightKey: ins.InsightKey,
			Label:      ResolveLabel(s.locale, ins.LabelI18nKey),
			Value:      ins.Value,
			Confidence: ins.Confidence,
			Salience:   ins.Salience,
			Evidence:   evidenceToViews(evRows),
		}
		out = append(out, view)
	}
	return out, nil
}

func facetToSummary(f lprepo.Facet) FacetSummary {
	return FacetSummary{
		FacetKey:        f.FacetKey,
		State:           f.State,
		Summary:         f.Summary,
		Confidence:      f.Confidence,
		ComputedVersion: f.ComputedVersion,
		UpdatedAt:       f.UpdatedAt,
	}
}

func evidenceToViews(rows []lprepo.Evidence) []EvidenceView {
	out := make([]EvidenceView, 0, len(rows))
	for _, ev := range rows {
		view := EvidenceView{
			SourceKind:       ev.SourceKind,
			SourceTable:      ev.SourceTable,
			ObservationCount: ev.ObservationCount,
			Contribution:     ev.Contribution,
		}
		if ev.CourseID != nil {
			s := ev.CourseID.String()
			view.CourseID = &s
		}
		if ev.WindowStart != nil {
			s := ev.WindowStart.UTC().Format(time.RFC3339)
			view.WindowStart = &s
		}
		if ev.WindowEnd != nil {
			s := ev.WindowEnd.UTC().Format(time.RFC3339)
			view.WindowEnd = &s
		}
		out = append(out, view)
	}
	return out
}

func hashUserID(id uuid.UUID) string {
	sum := sha256.Sum256([]byte(id.String()))
	return hex.EncodeToString(sum[:8])
}

func (s *Service) refreshFacetsPopulatedGauge(ctx context.Context) error {
	var n float64
	err := s.Pool.QueryRow(ctx, `
SELECT count(*)::float8 FROM learner.profile_facets WHERE state = 'ok'
`).Scan(&n)
	if err != nil {
		return err
	}
	setFacetsPopulated(n)
	return nil
}