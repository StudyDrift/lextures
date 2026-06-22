// Package peerreview implements allocation, aggregation, and grade blending (plan 3.15).
package peerreview

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/peerreview"
	prrepo "github.com/lextures/lextures/server/internal/repos/peerreview"
)

var (
	ErrNoConfig       = errors.New("peerreview: no peer review config for assignment")
	ErrNotEnoughPeers = errors.New("peerreview: not enough submissions to allocate reviews")
)

type Service struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Service {
	return Service{Pool: pool}
}

type AllocateResult struct {
	AllocationsCreated int
}

// Allocate assigns peer reviews for an assignment config, idempotently filling gaps.
func (s Service) Allocate(ctx context.Context, courseID, assignmentID uuid.UUID) (*AllocateResult, error) {
	cfg, err := prrepo.GetConfigByAssignment(ctx, s.Pool, courseID, assignmentID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrNoConfig
	}
	submitters, err := prrepo.ListSubmittersForAssignment(ctx, s.Pool, courseID, assignmentID)
	if err != nil {
		return nil, err
	}
	if len(submitters) < 2 {
		return nil, ErrNotEnoughPeers
	}

	enrollmentToSubmission := make(map[uuid.UUID]uuid.UUID, len(submitters))
	submissionToEnrollment := make(map[uuid.UUID]uuid.UUID, len(submitters))
	for _, sub := range submitters {
		enrollmentToSubmission[sub.EnrollmentID] = sub.SubmissionID
		submissionToEnrollment[sub.SubmissionID] = sub.EnrollmentID
	}

	reviewCounts, err := prrepo.CountReviewsPerSubmission(ctx, s.Pool, cfg.ID)
	if err != nil {
		return nil, err
	}
	for _, sub := range submitters {
		if _, ok := reviewCounts[sub.SubmissionID]; !ok {
			reviewCounts[sub.SubmissionID] = 0
		}
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := 0
	n := cfg.ReviewsPerReviewer
	for _, reviewer := range submitters {
		current, err := prrepo.CountReviewerAllocations(ctx, s.Pool, cfg.ID, reviewer.EnrollmentID)
		if err != nil {
			return nil, err
		}
		needed := n - current
		if needed <= 0 {
			continue
		}

		type candidate struct {
			submissionID uuid.UUID
			enrollmentID uuid.UUID
			reviewCount  int
		}
		candidates := make([]candidate, 0, len(submitters))
		for _, target := range submitters {
			if target.EnrollmentID == reviewer.EnrollmentID {
				continue
			}
			if cfg.ExcludeSameGroup {
				shared, err := prrepo.ShareGroup(ctx, s.Pool, reviewer.EnrollmentID, target.EnrollmentID)
				if err != nil {
					return nil, err
				}
				if shared {
					continue
				}
			}
			candidates = append(candidates, candidate{
				submissionID: target.SubmissionID,
				enrollmentID: target.EnrollmentID,
				reviewCount:  reviewCounts[target.SubmissionID],
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].reviewCount != candidates[j].reviewCount {
				return candidates[i].reviewCount < candidates[j].reviewCount
			}
			return candidates[i].submissionID.String() < candidates[j].submissionID.String()
		})

		assigned := 0
		for _, c := range candidates {
			if assigned >= needed {
				break
			}
			if err := prrepo.InsertAllocation(ctx, tx, cfg.ID, reviewer.EnrollmentID, c.submissionID); err != nil {
				return nil, err
			}
			reviewCounts[c.submissionID]++
			assigned++
			created++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &AllocateResult{AllocationsCreated: created}, nil
}

// AggregateScores computes the configured aggregate for a slice of scores.
func AggregateScores(scores []float64, mode peerreview.Aggregation) *float64 {
	if len(scores) == 0 {
		return nil
	}
	sorted := append([]float64(nil), scores...)
	sort.Float64s(sorted)
	switch mode {
	case peerreview.AggregationMean:
		sum := 0.0
		for _, s := range sorted {
			sum += s
		}
		v := sum / float64(len(sorted))
		return &v
	case peerreview.AggregationMedian:
		mid := len(sorted) / 2
		if len(sorted)%2 == 1 {
			v := sorted[mid]
			return &v
		}
		v := (sorted[mid-1] + sorted[mid]) / 2
		return &v
	case peerreview.AggregationTrimmed:
		if len(sorted) <= 2 {
			return AggregateScores(sorted, peerreview.AggregationMean)
		}
		trimmed := sorted[1 : len(sorted)-1]
		return AggregateScores(trimmed, peerreview.AggregationMean)
	default:
		var _ = mode
		return AggregateScores(sorted, peerreview.AggregationMean)
	}
}

// BlendGrade combines instructor and peer aggregate scores per weighted_blend config.
func BlendGrade(instructorScore, peerAggregate float64, blendWeight float64) float64 {
	instructorWeight := 1.0 - blendWeight
	return instructorWeight*instructorScore + blendWeight*peerAggregate
}

type SubmissionSummary struct {
	SubmissionID   uuid.UUID
	StudentUserID  uuid.UUID
	PeerAggregate  *float64
	ReviewCount    int
	CompletedCount int
}

type InstructorSummary struct {
	ConfigID            uuid.UUID
	TotalAllocations    int
	CompletedReviews    int
	IncompleteReviewers []uuid.UUID
	SubmissionSummaries []SubmissionSummary
	OutlierReviewers    []uuid.UUID
}

func (s Service) BuildInstructorSummary(ctx context.Context, cfg *prrepo.ConfigRow) (*InstructorSummary, error) {
	allocs, err := prrepo.ListAllocationsForAssignment(ctx, s.Pool, cfg.ID)
	if err != nil {
		return nil, err
	}
	reviews, err := prrepo.ListReviewsForConfig(ctx, s.Pool, cfg.ID)
	if err != nil {
		return nil, err
	}
	reviewByAlloc := make(map[uuid.UUID]prrepo.ReviewRow, len(reviews))
	for _, r := range reviews {
		reviewByAlloc[r.AllocationID] = r
	}

	reviewerComplete := make(map[uuid.UUID]int)
	reviewerAssigned := make(map[uuid.UUID]int)
	submissionScores := make(map[uuid.UUID][]float64)
	submissionUsers := make(map[uuid.UUID]uuid.UUID)

	for _, a := range allocs {
		reviewerAssigned[a.ReviewerEnrollmentID]++
		submissionUsers[a.TargetSubmissionID] = a.TargetUserID
		if _, ok := reviewByAlloc[a.ID]; ok {
			reviewerComplete[a.ReviewerEnrollmentID]++
			if rev, ok := reviewByAlloc[a.ID]; ok && rev.Score != nil {
				submissionScores[a.TargetSubmissionID] = append(submissionScores[a.TargetSubmissionID], *rev.Score)
			}
		}
	}

	incomplete := make([]uuid.UUID, 0)
	for rid, assigned := range reviewerAssigned {
		if reviewerComplete[rid] < assigned {
			incomplete = append(incomplete, rid)
		}
	}

	summaries := make([]SubmissionSummary, 0, len(submissionScores))
	for sid, scores := range submissionScores {
		agg := AggregateScores(scores, cfg.Aggregation)
		summaries = append(summaries, SubmissionSummary{
			SubmissionID:   sid,
			StudentUserID:  submissionUsers[sid],
			PeerAggregate:  agg,
			ReviewCount:    len(scores),
			CompletedCount: len(scores),
		})
	}

	outliers := detectOutlierReviewers(allocs, reviewByAlloc, cfg.Aggregation)

	return &InstructorSummary{
		ConfigID:            cfg.ID,
		TotalAllocations:    len(allocs),
		CompletedReviews:    len(reviews),
		IncompleteReviewers: incomplete,
		SubmissionSummaries: summaries,
		OutlierReviewers:    outliers,
	}, nil
}

func detectOutlierReviewers(allocs []prrepo.AllocationRow, reviewByAlloc map[uuid.UUID]prrepo.ReviewRow, mode peerreview.Aggregation) []uuid.UUID {
	bySubmission := make(map[uuid.UUID][]float64)
	reviewerScores := make(map[uuid.UUID][]float64)
	for _, a := range allocs {
		rev, ok := reviewByAlloc[a.ID]
		if !ok || rev.Score == nil {
			continue
		}
		bySubmission[a.TargetSubmissionID] = append(bySubmission[a.TargetSubmissionID], *rev.Score)
		reviewerScores[a.ReviewerEnrollmentID] = append(reviewerScores[a.ReviewerEnrollmentID], *rev.Score)
	}
	medians := make(map[uuid.UUID]float64)
	for sid, scores := range bySubmission {
		if agg := AggregateScores(scores, mode); agg != nil {
			medians[sid] = *agg
		}
	}
	outliers := make(map[uuid.UUID]struct{})
	for _, a := range allocs {
		rev, ok := reviewByAlloc[a.ID]
		if !ok || rev.Score == nil {
			continue
		}
		median, ok := medians[a.TargetSubmissionID]
		if !ok {
			continue
		}
		if math.Abs(*rev.Score-median) > 20 {
			outliers[a.ReviewerEnrollmentID] = struct{}{}
		}
	}
	out := make([]uuid.UUID, 0, len(outliers))
	for id := range outliers {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

// PeerAggregateForSubmission returns the aggregated peer score for one submission.
func (s Service) PeerAggregateForSubmission(ctx context.Context, cfg *prrepo.ConfigRow, submissionID uuid.UUID) (*float64, error) {
	reviews, err := prrepo.ListReviewsForSubmission(ctx, s.Pool, cfg.ID, submissionID)
	if err != nil {
		return nil, err
	}
	scores := make([]float64, 0, len(reviews))
	for _, r := range reviews {
		if r.Score != nil {
			scores = append(scores, *r.Score)
		}
	}
	return AggregateScores(scores, cfg.Aggregation), nil
}

func (s Service) UpsertConfig(ctx context.Context, in prrepo.UpsertConfigInput) (*prrepo.ConfigRow, error) {
	return prrepo.UpsertConfig(ctx, s.Pool, in)
}

func (s Service) GetConfig(ctx context.Context, courseID, assignmentID uuid.UUID) (*prrepo.ConfigRow, error) {
	return prrepo.GetConfigByAssignment(ctx, s.Pool, courseID, assignmentID)
}

func (s Service) SubmitReview(ctx context.Context, allocationID uuid.UUID, score *float64, rubricScores map[string]float64, comments *string) (*prrepo.ReviewRow, error) {
	if err := prrepo.UpdateAllocationStatus(ctx, s.Pool, allocationID, peerreview.AllocationSubmitted); err != nil {
		return nil, err
	}
	return prrepo.UpsertReview(ctx, s.Pool, allocationID, score, rubricScores, comments)
}

func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return "peerreview:ok", nil
}
