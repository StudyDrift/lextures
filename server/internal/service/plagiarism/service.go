package plagiarism

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/originalityconfig"
	"github.com/lextures/lextures/server/internal/repos/originalityreports"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// Service orchestrates originality scan enqueue and processing.
type Service struct {
	Pool         *pgxpool.Pool
	Config       config.Config
	FilesRoot    string
	AI           aiprovider.ScopedCompleter
	StubExternal bool
}

// EnqueueForSubmission creates pending report rows for configured providers.
func (s *Service) EnqueueForSubmission(ctx context.Context, sub originalityreports.SubmissionNeedingScan, externalProvider string) error {
	mode := strings.TrimSpace(sub.OriginalityMode)
	if mode == "" || mode == "disabled" {
		return nil
	}
	providers := providersForMode(mode, externalProvider)
	for _, p := range providers {
		if _, err := originalityreports.InsertPending(ctx, s.Pool, sub.SubmissionID, p); err != nil {
			return err
		}
	}
	return nil
}

func providersForMode(mode, external string) []string {
	ext := strings.TrimSpace(strings.ToLower(external))
	if ext == "" || ext == "none" {
		ext = ""
	}
	switch mode {
	case "plagiarism":
		if ext != "" {
			return []string{ext}
		}
		return []string{ProviderTurnitin}
	case "ai":
		return []string{ProviderInternal}
	case "both":
		out := []string{ProviderInternal}
		if ext != "" {
			out = append(out, ext)
		} else {
			out = append(out, ProviderTurnitin)
		}
		return out
	default:
		return nil
	}
}

// SweepEnqueue finds submissions missing reports and enqueues them.
func (s *Service) SweepEnqueue(ctx context.Context) (int, error) {
	external, err := s.activeExternalProvider(ctx)
	if err != nil {
		return 0, err
	}
	subs, err := originalityreports.ListSubmissionsNeedingEnqueue(ctx, s.Pool, 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, sub := range subs {
		if err := s.EnqueueForSubmission(ctx, sub, external); err != nil {
			slog.Warn("plagiarism: enqueue failed", "submission_id", sub.SubmissionID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

// ProcessNext claims and completes one pending originality report.
func (s *Service) ProcessNext(ctx context.Context) (bool, error) {
	report, err := originalityreports.ClaimNextPending(ctx, s.Pool)
	if err != nil || report == nil {
		return false, err
	}
	sc, err := originalityreports.GetSubmissionContextByID(ctx, s.Pool, report.SubmissionID)
	if err != nil || sc == nil {
		_ = originalityreports.MarkFailed(ctx, s.Pool, report.ID, "submission not found")
		return true, err
	}
	text, err := s.loadSubmissionText(ctx, sc.CourseCode, sc.AttachmentFileID)
	if err != nil {
		_ = originalityreports.MarkFailed(ctx, s.Pool, report.ID, err.Error())
		return true, err
	}
	provider, err := s.providerFor(report.Provider)
	if err != nil {
		_ = originalityreports.MarkFailed(ctx, s.Pool, report.ID, err.Error())
		return true, err
	}
	result, err := provider.Scan(ctx, text)
	if err != nil {
		_ = originalityreports.MarkFailed(ctx, s.Pool, report.ID, err.Error())
		return true, err
	}
	if err := originalityreports.MarkDone(ctx, s.Pool, report.ID, result.SimilarityPct, result.AIProbability, result.ReportURL, result.ReportToken, result.ProviderReportID); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) providerFor(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ProviderInternal:
		return InternalAIProvider{AI: s.AI, Model: ""}, nil
	case ProviderTurnitin, ProviderCopyleaks, ProviderGPTZero:
		if s.StubExternal || s.Config.OriginalityStubExternal {
			return StubExternalProvider{Name_: name}, nil
		}
		return StubExternalProvider{Name_: name}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

func (s *Service) loadSubmissionText(ctx context.Context, courseCode string, fileID *uuid.UUID) (string, error) {
	if fileID == nil {
		return "", fmt.Errorf("no submission text available")
	}
	row, err := coursefiles.GetForCourse(ctx, s.Pool, courseCode, *fileID)
	if err != nil || row == nil {
		return "", fmt.Errorf("submission file not found")
	}
	path := coursefiles.BlobDiskPath(s.FilesRoot, courseCode, row.StorageKey)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read submission file: %w", err)
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return "", fmt.Errorf("empty submission text")
	}
	if len(text) > 512<<10 {
		text = text[:512<<10]
	}
	return text, nil
}

func (s *Service) activeExternalProvider(ctx context.Context) (string, error) {
	cfg, err := originalityconfig.GetFull(ctx, s.Pool)
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "none", nil
	}
	p := strings.TrimSpace(strings.ToLower(cfg.ActiveExternalProvider))
	if p == "" {
		return "none", nil
	}
	return p, nil
}

// RetryFailed resets failed reports for a submission to pending.
func (s *Service) RetryFailed(ctx context.Context, submissionID uuid.UUID) (int, error) {
	reports, err := originalityreports.ListBySubmission(ctx, s.Pool, submissionID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range reports {
		if r.Status != "failed" {
			continue
		}
		ok, err := originalityreports.ResetForRetry(ctx, s.Pool, submissionID, r.Provider)
		if err != nil {
			return n, err
		}
		if ok {
			n++
		}
	}
	return n, nil
}
