package coursechecklist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/course"
	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
)

// Service orchestrates checklist evaluation, snapshot caching, and dismissals (CC.2).
type Service struct {
	Pool *pgxpool.Pool
	TTL  time.Duration
	Now  func() time.Time
}

// NewService builds a Service with the given snapshot TTL (0 → default).
func NewService(pool *pgxpool.Pool, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = SnapshotTTLDefault
	}
	return &Service{
		Pool: pool,
		TTL:  ttl,
		Now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

// GetChecklist returns the full checklist, recomputing when the snapshot is stale.
func (s *Service) GetChecklist(ctx context.Context, courseCode string, includeNotApplicable bool) (ChecklistResponse, error) {
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistResponse{}, err
	}
	res, computedAt, truncated, staleFlag, err := s.loadOrEvaluate(ctx, courseID, courseCode, false)
	if err != nil {
		return ChecklistResponse{}, err
	}
	dismissed, names, err := s.loadDismissals(ctx, courseID)
	if err != nil {
		return ChecklistResponse{}, err
	}
	return AssembleChecklist(res, AssembleOptions{
		CourseCode:           courseCode,
		ComputedAt:           computedAt,
		Stale:                staleFlag,
		EvidenceTruncated:    truncated,
		IncludeNotApplicable: includeNotApplicable,
		Dismissed:            dismissed,
		DisplayNames:         names,
	}), nil
}

// GetSummary returns badge counts from a warm snapshot without evaluating (FR-14).
// When the snapshot is missing/stale it evaluates once (same as GetChecklist).
func (s *Service) GetSummary(ctx context.Context, courseCode string) (ChecklistSummary, error) {
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistSummary{}, err
	}
	snap, freshness, err := s.loadSnapshotAndFreshness(ctx, courseID)
	if err != nil {
		return ChecklistSummary{}, err
	}
	engineV := EngineVersion()
	catalogV := CatalogVersion()
	now := s.now()
	if !IsSnapshotStale(snap, engineV, catalogV, s.TTL, freshness, now) {
		observeSnapshotHit("hit")
		dismissedCount := snap.DismissedCount
		if n, err := ccrepo.CountDismissed(ctx, s.Pool, courseID); err == nil {
			dismissedCount = n
		}
		return ChecklistSummary{
			OutstandingEssential: snap.OutstandingEssential,
			OutstandingTotal:     snap.OutstandingTotal,
			Done:                 snap.DoneCount,
			Total:                snap.TotalCount,
			Dismissed:            dismissedCount,
			ComputedAt:           snap.ComputedAt.UTC(),
			Stale:                false,
		}, nil
	}
	// Stale/miss: recompute via loadOrEvaluate (metrics recorded there).
	res, computedAt, truncated, staleFlag, err := s.loadOrEvaluate(ctx, courseID, courseCode, false)
	if err != nil {
		return ChecklistSummary{}, err
	}
	dismissed, _, err := s.loadDismissals(ctx, courseID)
	if err != nil {
		return ChecklistSummary{}, err
	}
	dismissedByID := make(map[string]ccrepo.ItemState, len(dismissed))
	for _, st := range dismissed {
		dismissedByID[st.ItemID] = st
	}
	_ = truncated
	_ = staleFlag
	return DeriveSummary(res, dismissedByID, computedAt, false), nil
}

// Refresh forces recomputation bypassing TTL (FR-17).
func (s *Service) Refresh(ctx context.Context, courseCode string) (ChecklistResponse, error) {
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistResponse{}, err
	}
	res, computedAt, truncated, _, err := s.loadOrEvaluate(ctx, courseID, courseCode, true)
	if err != nil {
		return ChecklistResponse{}, err
	}
	dismissed, names, err := s.loadDismissals(ctx, courseID)
	if err != nil {
		return ChecklistResponse{}, err
	}
	return AssembleChecklist(res, AssembleOptions{
		CourseCode:        courseCode,
		ComputedAt:        computedAt,
		Stale:             false,
		EvidenceTruncated: truncated,
		Dismissed:         dismissed,
		DisplayNames:      names,
	}), nil
}

// Dismiss marks an item dismissed for the course (FR-15).
func (s *Service) Dismiss(ctx context.Context, courseCode, rawItemID string, actor uuid.UUID, req DismissRequest) (ChecklistItem, error) {
	itemID, ok := ResolveItemID(rawItemID)
	if !ok {
		return ChecklistItem{}, ErrItemNotFound
	}
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistItem{}, err
	}
	st, changed, err := ccrepo.Dismiss(ctx, s.Pool, ccrepo.DismissInput{
		CourseID: courseID,
		ItemID:   string(itemID),
		ActorID:  actor,
		Reason:   req.Reason,
		Note:     req.Note,
	})
	if err != nil {
		return ChecklistItem{}, err
	}
	if changed {
		observeDismissal(st.DismissReason)
		slog.Info("coursechecklist.dismiss",
			"course_id", courseID.String(),
			"item_id", string(itemID),
			"actor_user_id", actor.String(),
			"action", "dismiss",
		)
	}
	_ = s.refreshSnapshotCounters(ctx, courseID, courseCode)
	return s.itemForID(ctx, courseID, courseCode, itemID, &st)
}

// Restore undoes a dismissal (FR-16).
func (s *Service) Restore(ctx context.Context, courseCode, rawItemID string, actor uuid.UUID) (ChecklistItem, error) {
	itemID, ok := ResolveItemID(rawItemID)
	if !ok {
		return ChecklistItem{}, ErrItemNotFound
	}
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistItem{}, err
	}
	st, changed, err := ccrepo.Restore(ctx, s.Pool, courseID, string(itemID), actor)
	if err != nil {
		return ChecklistItem{}, err
	}
	if changed {
		slog.Info("coursechecklist.restore",
			"course_id", courseID.String(),
			"item_id", string(itemID),
			"actor_user_id", actor.String(),
			"action", "restore",
		)
	}
	_ = s.refreshSnapshotCounters(ctx, courseID, courseCode)
	return s.itemForID(ctx, courseID, courseCode, itemID, &st)
}

// Recheck re-evaluates one item and patches the stored snapshot (FR-18).
func (s *Service) Recheck(ctx context.Context, courseCode, rawItemID string) (ChecklistItem, error) {
	itemID, ok := ResolveItemID(rawItemID)
	if !ok {
		return ChecklistItem{}, ErrItemNotFound
	}
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return ChecklistItem{}, err
	}
	res, computedAt, truncated, err := s.evaluateOnly(ctx, courseID, courseCode, itemID)
	if err != nil {
		return ChecklistItem{}, err
	}
	// Merge into existing snapshot payload when present.
	if err := s.mergeRecheckIntoSnapshot(ctx, courseID, res, computedAt, truncated); err != nil {
		slog.Warn("coursechecklist.recheck.snapshot_patch_failed",
			"course_id", courseID.String(), "item_id", string(itemID), "err", err.Error())
	}
	dismissed, names, err := s.loadDismissals(ctx, courseID)
	if err != nil {
		return ChecklistItem{}, err
	}
	dismissedByID := map[string]ccrepo.ItemState{}
	for _, st := range dismissed {
		dismissedByID[st.ItemID] = st
	}
	for _, fr := range res.Findings {
		if fr.ID == itemID {
			return itemFromFinding(fr, dismissedByID[string(itemID)], names), nil
		}
	}
	return ChecklistItem{}, ErrItemNotFound
}

// History returns recent checklist audit events (≤ 100).
func (s *Service) History(ctx context.Context, courseCode string) (HistoryResponse, error) {
	courseID, err := s.resolveCourseID(ctx, courseCode)
	if err != nil {
		return HistoryResponse{}, err
	}
	events, err := ccrepo.ListEvents(ctx, s.Pool, courseID, 100)
	if err != nil {
		return HistoryResponse{}, err
	}
	out := make([]HistoryEvent, 0, len(events))
	for _, e := range events {
		he := HistoryEvent{
			ID:         e.ID.String(),
			ItemID:     e.ItemID,
			Action:     e.Action,
			Reason:     e.Reason,
			OccurredAt: e.OccurredAt.UTC(),
		}
		if e.ActorUserID != nil {
			s := e.ActorUserID.String()
			he.ActorUserID = &s
		}
		out = append(out, he)
	}
	return HistoryResponse{
		CourseCode:     courseCode,
		EngineVersion:  EngineVersion(),
		CatalogVersion: CatalogVersion(),
		Events:         out,
	}, nil
}

// ErrItemNotFound is returned when item_id is unknown or retired.
var ErrItemNotFound = errors.New("coursechecklist: item not found")

// ErrCourseNotFound is returned when the course code does not resolve.
var ErrCourseNotFound = errors.New("coursechecklist: course not found")

func (s *Service) resolveCourseID(ctx context.Context, courseCode string) (uuid.UUID, error) {
	if s == nil || s.Pool == nil {
		return uuid.Nil, errors.New("coursechecklist: nil service pool")
	}
	id, err := course.GetIDByCourseCode(ctx, s.Pool, courseCode)
	if err != nil {
		return uuid.Nil, err
	}
	if id == nil {
		return uuid.Nil, ErrCourseNotFound
	}
	return *id, nil
}

func (s *Service) loadDismissals(ctx context.Context, courseID uuid.UUID) ([]ccrepo.ItemState, map[uuid.UUID]string, error) {
	dismissed, err := ccrepo.ListDismissed(ctx, s.Pool, courseID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]uuid.UUID, 0, len(dismissed))
	for _, st := range dismissed {
		if st.DismissedByUserID != nil {
			ids = append(ids, *st.DismissedByUserID)
		}
	}
	names, err := userrepo.DisplayLabelsByIDs(ctx, s.Pool, ids)
	if err != nil {
		return dismissed, map[uuid.UUID]string{}, nil
	}
	return dismissed, names, nil
}

func (s *Service) loadSnapshotAndFreshness(ctx context.Context, courseID uuid.UUID) (*ccrepo.Snapshot, ccrepo.MutationFreshness, error) {
	snap, err := ccrepo.GetSnapshot(ctx, s.Pool, courseID)
	if err != nil {
		return nil, ccrepo.MutationFreshness{}, err
	}
	freshness, err := ccrepo.MutationFreshnessForCourse(ctx, s.Pool, courseID)
	if err != nil {
		return nil, ccrepo.MutationFreshness{}, err
	}
	return snap, freshness, nil
}

func (s *Service) loadOrEvaluate(ctx context.Context, courseID uuid.UUID, courseCode string, force bool) (Result, time.Time, bool, bool, error) {
	snap, freshness, err := s.loadSnapshotAndFreshness(ctx, courseID)
	if err != nil {
		return Result{}, time.Time{}, false, false, err
	}
	engineV := EngineVersion()
	catalogV := CatalogVersion()
	now := s.now()
	stale := force || IsSnapshotStale(snap, engineV, catalogV, s.TTL, freshness, now)
	if !stale {
		observeSnapshotHit("hit")
		res, truncated, err := decodeSnapshotPayload(snap.Payload)
		if err != nil {
			// Corrupt payload — recompute.
			observeSnapshotHit("stale")
		} else {
			return res, snap.ComputedAt.UTC(), truncated, false, nil
		}
	} else if snap == nil {
		observeSnapshotHit("miss")
	} else {
		observeSnapshotHit("stale")
	}

	res, computedAt, truncated, err := s.evaluateFull(ctx, courseID, courseCode)
	if err != nil {
		return Result{}, time.Time{}, false, false, err
	}
	if err := s.writeSnapshotBestEffort(ctx, courseID, res, computedAt, truncated); err != nil {
		slog.Warn("coursechecklist.snapshot_write_failed",
			"course_id", courseID.String(), "err", err.Error())
	}
	return res, computedAt, truncated, false, nil
}

func (s *Service) evaluateFull(ctx context.Context, courseID uuid.UUID, courseCode string) (Result, time.Time, bool, error) {
	key := courseID.String()
	type evalOut struct {
		res       Result
		at        time.Time
		truncated bool
	}
	v, err, _ := evalFlight.Do(key, func() (any, error) {
		acquireEvalSlot()
		defer releaseEvalSlot()
		needs := DataNeedsForEvaluate(MustDefault(), EvaluateOptions{})
		snap, err := LoadSnapshot(ctx, s.Pool, courseCode, needs)
		if err != nil {
			return nil, err
		}
		res := Evaluate(ctx, snap, EvaluateOptions{})
		truncated := false
		res, truncated = fitPayload(res)
		return evalOut{res: res, at: s.now(), truncated: truncated}, nil
	})
	if err != nil {
		return Result{}, time.Time{}, false, err
	}
	out := v.(evalOut)
	return out.res, out.at, out.truncated, nil
}

func (s *Service) evaluateOnly(ctx context.Context, courseID uuid.UUID, courseCode string, itemID ItemID) (Result, time.Time, bool, error) {
	key := courseID.String() + ":only:" + string(itemID)
	type evalOut struct {
		res       Result
		at        time.Time
		truncated bool
	}
	v, err, _ := evalFlight.Do(key, func() (any, error) {
		acquireEvalSlot()
		defer releaseEvalSlot()
		opt := EvaluateOptions{Only: []ItemID{itemID}}
		needs := DataNeedsForEvaluate(MustDefault(), opt)
		snap, err := LoadSnapshot(ctx, s.Pool, courseCode, needs)
		if err != nil {
			return nil, err
		}
		res := Evaluate(ctx, snap, opt)
		truncated := false
		res, truncated = fitPayload(res)
		return evalOut{res: res, at: s.now(), truncated: truncated}, nil
	})
	if err != nil {
		return Result{}, time.Time{}, false, err
	}
	out := v.(evalOut)
	return out.res, out.at, out.truncated, nil
}

func (s *Service) writeSnapshotBestEffort(ctx context.Context, courseID uuid.UUID, res Result, computedAt time.Time, truncated bool) error {
	dismissedCount, err := ccrepo.CountDismissed(ctx, s.Pool, courseID)
	if err != nil {
		dismissedCount = 0
	}
	dismissed, err := ccrepo.ListDismissed(ctx, s.Pool, courseID)
	if err != nil {
		dismissed = nil
	}
	dismissedByID := make(map[string]ccrepo.ItemState, len(dismissed))
	for _, st := range dismissed {
		dismissedByID[st.ItemID] = st
	}
	summary := DeriveSummary(res, dismissedByID, computedAt, false)
	payload, err := encodeSnapshotPayload(res, truncated)
	if err != nil {
		return err
	}
	return ccrepo.UpsertSnapshot(ctx, s.Pool, ccrepo.UpsertSnapshotInput{
		CourseID:             courseID,
		ComputedAt:           computedAt,
		EngineVersion:        res.EngineVersion,
		CatalogVersion:       res.CatalogVersion,
		Payload:              payload,
		TotalCount:           summary.Total,
		DoneCount:            summary.Done,
		OutstandingEssential: summary.OutstandingEssential,
		OutstandingTotal:     summary.OutstandingTotal,
		DismissedCount:       dismissedCount,
	})
}

func (s *Service) refreshSnapshotCounters(ctx context.Context, courseID uuid.UUID, courseCode string) error {
	snap, err := ccrepo.GetSnapshot(ctx, s.Pool, courseID)
	if err != nil || snap == nil {
		return err
	}
	res, _, err := decodeSnapshotPayload(snap.Payload)
	if err != nil {
		return err
	}
	dismissed, err := ccrepo.ListDismissed(ctx, s.Pool, courseID)
	if err != nil {
		return err
	}
	dismissedByID := make(map[string]ccrepo.ItemState, len(dismissed))
	for _, st := range dismissed {
		dismissedByID[st.ItemID] = st
	}
	summary := DeriveSummary(res, dismissedByID, snap.ComputedAt.UTC(), false)
	return ccrepo.UpdateSnapshotCounters(ctx, s.Pool, courseID,
		summary.OutstandingEssential, summary.OutstandingTotal, summary.Done, summary.Total, summary.Dismissed)
}

func (s *Service) mergeRecheckIntoSnapshot(ctx context.Context, courseID uuid.UUID, partial Result, computedAt time.Time, truncated bool) error {
	snap, err := ccrepo.GetSnapshot(ctx, s.Pool, courseID)
	if err != nil {
		return err
	}
	var base Result
	baseTruncated := truncated
	if snap != nil {
		decoded, tr, err := decodeSnapshotPayload(snap.Payload)
		if err == nil {
			base = decoded
			baseTruncated = baseTruncated || tr
		}
	}
	if len(base.Findings) == 0 {
		// No base — store partial as full (rare cold recheck).
		return s.writeSnapshotBestEffort(ctx, courseID, partial, computedAt, truncated)
	}
	byID := make(map[ItemID]ItemResult, len(partial.Findings))
	for _, fr := range partial.Findings {
		byID[fr.ID] = fr
	}
	for i, fr := range base.Findings {
		if repl, ok := byID[fr.ID]; ok {
			base.Findings[i] = repl
		}
	}
	base.Counts = aggregateCounts(base.Findings)
	base.ByCategory = aggregateByCategory(base.Findings)
	base.EngineVersion = EngineVersion()
	base.CatalogVersion = CatalogVersion()
	base, fitTrunc := fitPayload(base)
	return s.writeSnapshotBestEffort(ctx, courseID, base, computedAt, baseTruncated || fitTrunc)
}

func (s *Service) itemForID(ctx context.Context, courseID uuid.UUID, courseCode string, itemID ItemID, st *ccrepo.ItemState) (ChecklistItem, error) {
	// Prefer warm snapshot finding; fall back to single-item evaluate.
	var fr *ItemResult
	if snap, err := ccrepo.GetSnapshot(ctx, s.Pool, courseID); err == nil && snap != nil {
		if res, _, err := decodeSnapshotPayload(snap.Payload); err == nil {
			for i := range res.Findings {
				if res.Findings[i].ID == itemID {
					fr = &res.Findings[i]
					break
				}
			}
		}
	}
	if fr == nil {
		res, _, _, err := s.evaluateOnly(ctx, courseID, courseCode, itemID)
		if err != nil {
			return ChecklistItem{}, err
		}
		for i := range res.Findings {
			if res.Findings[i].ID == itemID {
				fr = &res.Findings[i]
				break
			}
		}
	}
	if fr == nil {
		return ChecklistItem{}, ErrItemNotFound
	}
	names := map[uuid.UUID]string{}
	state := ccrepo.ItemState{}
	if st != nil {
		state = *st
		if st.DismissedByUserID != nil {
			if m, err := userrepo.DisplayLabelsByIDs(ctx, s.Pool, []uuid.UUID{*st.DismissedByUserID}); err == nil {
				names = m
			}
		}
	}
	return itemFromFinding(*fr, state, names), nil
}

func encodeSnapshotPayload(res Result, truncated bool) (json.RawMessage, error) {
	b, err := json.Marshal(snapshotPayload{Result: res, EvidenceTruncated: truncated})
	if err != nil {
		return nil, err
	}
	if len(b) > ccrepo.MaxPayloadBytes {
		return nil, ccrepo.ErrPayloadTooLarge
	}
	return b, nil
}

func decodeSnapshotPayload(raw json.RawMessage) (Result, bool, error) {
	var p snapshotPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Result{}, false, err
	}
	return p.Result, p.EvidenceTruncated, nil
}

// fitPayload drops evidence when the serialized snapshot would exceed 256 KiB (AC-13).
func fitPayload(res Result) (Result, bool) {
	if _, err := encodeSnapshotPayload(res, false); err == nil {
		return res, false
	}
	stripped := DropEvidence(res)
	if _, err := encodeSnapshotPayload(stripped, true); err != nil {
		// Still too large — return stripped anyway; write will fail best-effort.
		slog.Warn("coursechecklist.payload_still_too_large", "err", err.Error())
	}
	return stripped, true
}

// SweepRetention deletes aged snapshots and audit events (nightly sweeper).
func SweepRetention(ctx context.Context, pool *pgxpool.Pool, now time.Time) (snapshotsDeleted, eventsDeleted int64, err error) {
	if pool == nil {
		return 0, 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	snapCutoff := now.AddDate(0, 0, -90)
	eventCutoff := now.AddDate(0, 0, -400)
	snapshotsDeleted, err = ccrepo.DeleteSnapshotsUntouchedSince(ctx, pool, snapCutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("sweep snapshots: %w", err)
	}
	eventsDeleted, err = ccrepo.DeleteEventsOlderThan(ctx, pool, eventCutoff)
	if err != nil {
		return snapshotsDeleted, 0, fmt.Errorf("sweep events: %w", err)
	}
	return snapshotsDeleted, eventsDeleted, nil
}
