// Package background runs periodic jobs matching server/src/lib.rs (30s tickers).
package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/terms"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/itemanalysis"
	"github.com/lextures/lextures/server/internal/service/learningevents"
	"github.com/lextures/lextures/server/internal/service/openrouter"
	"github.com/lextures/lextures/server/internal/service/quizautosubmit"
	seattimesvc "github.com/lextures/lextures/server/internal/service/seattime"
	"github.com/lextures/lextures/server/internal/workers/avscan"
	"github.com/lextures/lextures/server/internal/workers/captioning"
	"github.com/lextures/lextures/server/internal/workers/h5pextract"
	"github.com/lextures/lextures/server/internal/workers/scormextract"
	"github.com/lextures/lextures/server/internal/workers/catalogsync"
	"github.com/lextures/lextures/server/internal/workers/sissync"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
	"github.com/lextures/lextures/server/internal/workers/transcode"
)

// Start launches quiz auto-submit and (when enabled) grade-posting sweeps on a 30s ticker
// (Rust `server/src/lib.rs`).
func Start(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) {
	StartWithStorage(ctx, pool, cfg, nil, nil)
}

// StartWithStorage is Start extended with an optional storage driver for transcode jobs.
func StartWithStorage(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, storage filestorage.Driver, smsQueue *smsnotificationqueue.Bus) {
	if pool == nil {
		return
	}
	go runEvery(ctx, 30*time.Second, func() {
		sweepExpiredQuizAttempts(context.Background(), pool, cfg, time.Now().UTC())
	})
	go runEvery(ctx, 30*time.Second, func() {
		if !cfg.GradePostingPoliciesEnabled {
			return
		}
		sweepScheduledReleases(context.Background(), pool, cfg, smsQueue, time.Now().UTC())
	})
	go runEvery(ctx, 30*time.Second, func() {
		n, err := terms.SweepStatuses(context.Background(), pool, time.Now().UTC())
		if err != nil {
			slog.Warn("term status sweep failed", "err", err)
			return
		}
		if n > 0 {
			slog.Info("term status sweep updated rows", "count", n)
		}
	})
	go runEvery(ctx, 30*time.Second, func() {
		if _, err := apitokens.FlushUsage(context.Background(), pool); err != nil {
			slog.Warn("api token usage flush failed", "err", err)
		}
	})
	go runEvery(ctx, 30*time.Second, func() {
		n, err := orgroles.SweepExpired(context.Background(), pool, time.Now().UTC(), 200)
		if err != nil {
			slog.Warn("org role grant sweep failed", "err", err)
			return
		}
		if n > 0 {
			slog.Info("org role grant sweep deleted rows", "count", n)
		}
	})
	go runEvery(ctx, 15*time.Second, func() {
		now := time.Now().UTC()
		sweepEmailJobs(context.Background(), pool, cfg, now)
	})
	go runEvery(ctx, 15*time.Second, func() {
		sweepPaymentWebhookJobs(context.Background(), pool, cfg, time.Now().UTC())
	})
	go runEvery(ctx, time.Minute, func() {
		sweepDailyDigests(context.Background(), pool, cfg, time.Now().UTC())
	})
	go runEvery(ctx, time.Hour, func() {
		sweepConferenceReminders(context.Background(), pool, cfg, time.Now().UTC())
	})
	go runEvery(ctx, 15*time.Second, func() {
		sweepPushJobs(context.Background(), pool, cfg, time.Now().UTC())
	})
	if cfg.FFCEUTracking {
		seattimesvc.InitGlobalBuffer(pool)
		go runEvery(ctx, 30*time.Second, func() {
			if seattimesvc.GlobalBuffer != nil {
				if err := seattimesvc.GlobalBuffer.Flush(context.Background()); err != nil {
					slog.Warn("seat-time buffer flush failed", "err", err)
				}
			}
		})
	}
	go runEvery(ctx, time.Minute, func() {
		sweepStuckGradingRuns(context.Background(), pool)
	})
	go runEvery(ctx, time.Hour, func() {
		n, err := SweepStalledTusUploads(context.Background(), pool, time.Now().UTC())
		if err != nil {
			slog.Warn("tus upload cleanup failed", "err", err)
			return
		}
		if n > 0 {
			slog.Info("tus upload cleanup deleted stalled uploads", "count", n)
		}
	})

	if cfg.VideoTranscodingEnabled && storage != nil {
		worker := transcode.New(pool, storage)
		if cfg.FFmpegPath != "" {
			worker.FFmpegPath = cfg.FFmpegPath
		}
		worker.AutoCaptionOnComplete = cfg.AutoCaptioningEnabled
		worker.CaptionBackend = cfg.WhisperBackend
		go runEvery(ctx, 30*time.Second, func() {
			sweepTranscodeJobs(context.Background(), worker)
		})
		slog.Info("video transcoding worker started")
	}

	if cfg.AutoCaptioningEnabled && storage != nil {
		backend := captioning.Backend(cfg.WhisperBackend)
		if backend == "" {
			backend = captioning.BackendWhisperAPI
		}
		capWorker := captioning.New(pool, storage, backend, cfg.OpenAIAPIKey)
		go runEvery(ctx, 30*time.Second, func() {
			sweepCaptionJobs(context.Background(), capWorker)
		})
		slog.Info("auto-captioning worker started", "backend", string(backend))
	}

	if cfg.ItemAnalysisEnabled {
		go runEvery(ctx, time.Minute, func() {
			sweepItemAnalysis(context.Background(), pool, time.Now().UTC())
		})
		slog.Info("item analysis background sweep started")
	}

	if cfg.XAPIEmissionEnabled {
		go runEvery(ctx, 15*time.Second, func() {
			learningevents.SweepForwardJobs(context.Background(), pool, cfg, time.Now().UTC())
		})
		slog.Info("xAPI LRS forwarding worker started")
	}

	if cfg.H5PEnabled && storage != nil {
		h5pWorker := h5pextract.New(pool, storage)
		go runEvery(ctx, 15*time.Second, func() {
			for {
				done, err := h5pWorker.ProcessNext(context.Background())
				if err != nil {
					slog.Warn("h5p extract sweep: error", "err", err)
					break
				}
				if !done {
					break
				}
			}
		})
		slog.Info("h5p extract worker started")
	}

	if cfg.ScormIngestionEnabled && storage != nil {
		scormWorker := scormextract.New(pool, storage)
		go runEvery(ctx, 15*time.Second, func() {
			for {
				done, err := scormWorker.ProcessNext(context.Background())
				if err != nil {
					slog.Warn("scorm extract sweep: error", "err", err)
					break
				}
				if !done {
					break
				}
			}
		})
		slog.Info("scorm extract worker started")
	}

	go runEvery(ctx, time.Hour, func() {
		sweepAtRiskScores(context.Background(), pool, cfg, time.Now().UTC())
	})

	go runEvery(ctx, 24*time.Hour, func() {
		now := time.Now().UTC()
		sweepIncompleteReminders(context.Background(), pool, cfg, now)
		sweepIncompleteLapse(context.Background(), pool, cfg, now)
	})

	if cfg.FFSISIntegration {
		go runEvery(ctx, time.Hour, func() {
			sissync.SweepScheduled(context.Background(), pool)
		})
		slog.Info("sis integration sync worker started")
	}

	if cfg.FFCatalogIntegration {
		go runEvery(ctx, time.Hour, func() {
			catalogsync.SweepScheduled(context.Background(), pool)
		})
		slog.Info("course catalog sync worker started")
	}

	if cfg.SelfReflectionEnabled {
		var orClient *openrouter.Client
		if cfg.OpenRouterAPIKey != "" {
			orClient = openrouter.NewClient(cfg.OpenRouterAPIKey)
		}
		go runEvery(ctx, 24*time.Hour, func() {
			sweepWeeklyCoachingTips(context.Background(), pool, cfg, orClient, time.Now().UTC())
		})
		slog.Info("weekly coaching tips sweep started")
	}

	if cfg.FFStudyReminders {
		go runEvery(ctx, time.Minute, func() {
			sweepStudyReminders(context.Background(), pool, cfg, time.Now().UTC())
		})
		slog.Info("study reminders sweep started")
	}

	if cfg.ReportExportEnabled {
		go runEvery(ctx, time.Minute, func() {
			sweepScheduledReports(context.Background(), pool, cfg, time.Now().UTC())
		})
		slog.Info("scheduled report delivery worker started")
	}

	if cfg.AvScanningEnabled && storage != nil {
		clam := clamav.NewClient(cfg.ClamAVAddr, cfg.ClamAVStub)
		avWorker := avscan.New(pool, storage, clam)
		avWorker.LocalRoot = cfg.CourseFilesRoot
		avWorker.EmailEnabled = cfg.EmailNotificationsEnabled
		go runEvery(ctx, 10*time.Second, func() {
			sweepAVScanJobs(context.Background(), avWorker)
		})
		slog.Info("av scanning worker started", "clamav_addr", cfg.ClamAVAddr, "stub", cfg.ClamAVStub)
	}

	go runEvery(ctx, 15*time.Second, func() {
		now := time.Now().UTC()
		sweepWebhookDeliveries(context.Background(), pool, cfg, now)
	})
	go runEvery(ctx, 24*time.Hour, func() {
		sweepWebhookRetention(context.Background(), pool, cfg, time.Now().UTC())
	})
	go runEvery(ctx, time.Minute, func() {
		sweepBotDueSoonReminders(context.Background(), pool, cfg, time.Now().UTC())
	})

	if cfg.FFPlagiarismChecks && cfg.OriginalityDetectionEnabled {
		go runEvery(ctx, 30*time.Second, func() {
			sweepOriginalityScans(context.Background(), pool, cfg)
		})
		slog.Info("originality scan worker started")
	}
}

func runEvery(ctx context.Context, d time.Duration, fn func()) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}

func sweepItemAnalysis(ctx context.Context, pool *pgxpool.Pool, now time.Time) {
	ids, err := repoitemanalysis.ListStaleQuizzes(ctx, pool, now, itemanalysis.MinResponses, 20, 6*time.Hour)
	if err != nil {
		slog.Warn("item analysis sweep: list stale quizzes failed", "err", err)
		return
	}
	for _, id := range ids {
		if _, err := itemanalysis.Compute(ctx, pool, id); err != nil {
			slog.Warn("item analysis sweep: compute failed", "quiz_id", id, "err", err)
		} else {
			slog.Info("item analysis sweep: computed stats", "quiz_id", id)
		}
	}
}

func sweepExpiredQuizAttempts(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	n, err := quizautosubmit.SweepExpiredAttempts(ctx, pool, cfg, now, 200)
	if err != nil {
		slog.Warn("auto-submit sweep failed", "err", err)
		return
	}
	if n > 0 {
		slog.Info("auto-submit sweep completed", "count", n)
	}
}
