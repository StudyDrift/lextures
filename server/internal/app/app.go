// Package app wires configuration, the database, migrations, and the HTTP server.
package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/jwtblocklist"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/bannerevents"
	"github.com/lextures/lextures/server/internal/canvasimportevents"
	"github.com/lextures/lextures/server/internal/canvasimportqueue"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncevents"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncjobs"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
	"github.com/lextures/lextures/server/internal/commevents"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/feedevents"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	"github.com/lextures/lextures/server/internal/httpserver"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/lti"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/objectcache"
	"github.com/lextures/lextures/server/internal/platformstate"
	"github.com/lextures/lextures/server/internal/redisclient"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
	"github.com/lextures/lextures/server/internal/scheduler"
	botsservice "github.com/lextures/lextures/server/internal/service/bots"
	emailtemplatesvc "github.com/lextures/lextures/server/internal/service/emailtemplates"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/integrations"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
	learnerprofilederivers "github.com/lextures/lextures/server/internal/service/learnerprofile/derivers"
	marketplacecoursesservice "github.com/lextures/lextures/server/internal/service/marketplacecourses"
	"github.com/lextures/lextures/server/internal/service/oidcauth"
	"github.com/lextures/lextures/server/internal/service/storagequota"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Run starts the API. Pass the migration file tree (e.g. serverdata.Migrations from the module root).
func Run(ctx context.Context, fsys fs.FS) error {
	cfg := config.Load()
	// Observability stack (plan 17.7): Prometheus metrics, OTel traces, Sentry.
	// Built before logging.Configure so the Sentry ERROR-forwarding handler is
	// chained downstream of PII redaction (Sentry never sees unredacted PII).
	tel := telemetry.Init(ctx, telemetryConfig(cfg))
	defer tel.Shutdown(context.Background())
	logging.Configure(logging.Settings{
		DisableRedaction: cfg.DisablePIIRedaction,
		ExtraFields:      cfg.PIIRedactFields,
		HMACSecret:       []byte(cfg.JWTSecret),
		AppEnv:           cfg.AppEnv,
		WrapInner:        tel.WrapSlog,
	})
	if err := cfg.Validate(); err != nil {
		return err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("app: database: %w", err)
	}
	defer pool.Close()

	healthPool, err := db.NewHealthPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("app: health pool: %w", err)
	}
	defer healthPool.Close()

	// Shared Redis powers cross-instance state (JWT blocklist, rate limits,
	// caches) so the app tier can scale horizontally (plan 17.2). Unset REDIS_URL
	// keeps single-instance behaviour (redisClient is nil).
	redisClient, err := redisclient.New(ctx, redisclient.Config{
		URL:     cfg.RedisURL,
		PoolMin: cfg.RedisPoolMin,
		PoolMax: cfg.RedisPoolMax,
	})
	if err != nil {
		return fmt.Errorf("app: redis: %w", err)
	}
	if redisClient != nil {
		defer func() { _ = redisClient.Close() }()
		slog.Info("redis connected", "pool_min", cfg.RedisPoolMin, "pool_max", cfg.RedisPoolMax)
	}
	jwtBlocklist := jwtblocklist.New(redisClient)

	// Wire live DB/Redis/job-queue snapshots into the metrics collector so the
	// /metrics endpoint reflects current pool utilisation and queue depth at
	// scrape time (plan 17.7 FR-1). Closures keep telemetry decoupled from these
	// packages; nil sub-systems are simply omitted.
	if err := tel.RegisterSources(telemetrySources(pool, redisClient)); err != nil {
		slog.Warn("telemetry: collector registration failed", "err", err)
	}

	if cfg.RunMigrations {
		if err := migrate.RunWithFS(ctx, fsys, cfg.DatabaseURL); err != nil {
			return err
		}
	}

	dbPlatform, err := platformconfig.Get(ctx, pool)
	if err != nil {
		// Integration tests (and some local workflows) set RUN_MIGRATIONS=false against an
		// empty database, so migration 118 never creates settings.platform_app_settings.
		// Treat a missing table like "no DB overrides" instead of failing startup.
		if cfg.RunMigrations || !isUndefinedTable(err) {
			return fmt.Errorf("app: platform settings: %w", err)
		}
		dbPlatform = nil
	}
	merged := platformconfig.Merge(cfg, dbPlatform)
	if err := merged.Validate(); err != nil {
		return fmt.Errorf("app: effective configuration invalid (environment + database settings): %w", err)
	}
	// marketplace_flag_state on config load (plan MKT1 observability).
	telemetry.SetMarketplaceFlagState(merged.FFCourseMarketplace)
	platform := platformstate.New(merged)

	// ET-2: wire mail.Send* through org → system → code template resolution.
	// Delivery overrides follow the email template editor feature flag.
	emailtemplatesvc.WireMailSlotRenderer(pool, func() bool {
		return platform.Config().EmailTemplateEditorEnabled
	})

	storage, storageErr := filestorage.New(filestorage.BackendConfig{
		Backend:         cfg.StorageBackend,
		LocalRoot:       cfg.CourseFilesRoot,
		Endpoint:        cfg.StorageEndpoint,
		AccessKeyID:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Bucket:          cfg.StorageBucket,
		UseSSL:          cfg.StorageUseSSL,
		Region:          cfg.StorageRegion,
		CDNBaseURL:      merged.StorageCDNBaseURL,
	})
	if storageErr != nil {
		return fmt.Errorf("app: storage: %w", storageErr)
	}

	smsNotificationQueue, smsQueueErr := smsnotificationqueue.NewBus(
		merged.SmsNotificationQueueURL(),
		merged.SmsNotificationQueueName,
		merged.SmsNotificationConcurrency,
	)
	if smsQueueErr != nil {
		return fmt.Errorf("app: sms notification queue: %w", smsQueueErr)
	}
	defer func() { _ = smsNotificationQueue.Close() }()

	background.StartWithStorage(ctx, pool, merged, storage, smsNotificationQueue)
	// Generic durable background job queue (plan 17.3). No-op unless
	// BACKGROUND_JOBS_ENABLED; safe to start on every instance — workers claim
	// rows with SELECT ... FOR UPDATE SKIP LOCKED so they coordinate via Postgres.
	jobRegistry := background.StartJobQueueWorker(ctx, pool, platform)

	// Scheduled-jobs / cron layer (plan 17.4). The Scheduler is always
	// constructed so the admin API can list and manually trigger jobs, but the
	// tick loop only runs when enabled and the job queue is on (scheduled
	// triggers enqueue onto that queue). The distributed lock makes it safe to
	// run on every instance.
	sched := scheduler.New(pool, "")
	if pool != nil && merged.SchedulerEnabled && merged.BackgroundJobsEnabled {
		sched.Start(ctx)
	}

	ltiRT := lti.NewFromConfig(merged)
	brandingResolver := orgbranding.NewResolver(pool, merged.BrandingMultitenantHostSuffix, webHostFromOrigin(merged.PublicWebOrigin))

	var quotaSvc *storagequota.Service
	if merged.StorageQuotasEnabled {
		quotaSvc = &storagequota.Service{Pool: pool}
	}

	canvasImportHub := canvasimportevents.New()
	canvasImportQueue, queueErr := canvasimportqueue.NewBus(merged.CanvasImportQueueURL(), merged.CanvasImportQueueName, merged.CanvasImportConcurrency)
	if queueErr != nil {
		return fmt.Errorf("app: canvas import queue: %w", queueErr)
	}
	defer func() { _ = canvasImportQueue.Close() }()

	canvasSubmissionSyncHub := canvassubmissionsyncevents.New()
	canvasSubmissionSyncJobs := canvassubmissionsyncjobs.NewRegistry()
	canvasSubmissionSyncQueue, syncQueueErr := canvassubmissionsyncqueue.NewBus(
		merged.CanvasSubmissionSyncQueueURL(),
		merged.CanvasSubmissionSyncQueueName,
		merged.CanvasSubmissionSyncConcurrency,
	)
	if syncQueueErr != nil {
		return fmt.Errorf("app: canvas submission sync queue: %w", syncQueueErr)
	}
	defer func() { _ = canvasSubmissionSyncQueue.Close() }()

	gradingAgentQueue, gradingQueueErr := gradingagentqueue.NewBus(merged.GradingAgentQueueURL(), "grading.agent.run", 2)
	if gradingQueueErr != nil {
		return fmt.Errorf("app: grading agent queue: %w", gradingQueueErr)
	}
	defer func() { _ = gradingAgentQueue.Close() }()

	jwtSigner := auth.NewJWTSignerWithPool(cfg.JWTSecret, pool)
	if jwtBlocklist != nil {
		jwtSigner = jwtSigner.WithBlocklist(jwtBlocklist)
	}
	var learnerProfileSvc *learnerprofileservice.Service
	var introCourseSvc *introcourseservice.Service
	if pool != nil {
		learnerProfileSvc = learnerprofileservice.New(pool,
			learnerprofilederivers.StudyRhythmDeriver{Pool: pool},
			learnerprofilederivers.ContentModalityDeriver{Pool: pool},
			learnerprofilederivers.StrengthsGrowthDeriver{Pool: pool},
			learnerprofilederivers.InterestsDeriver{Pool: pool},
			learnerprofilederivers.LearningApproachDeriver{Pool: pool},
		)
		learnerprofileservice.RegisterMetrics(tel.Metrics.Registry())
		if redisClient != nil {
			learnerProfileSvc.SetRedis(redisClient)
		}
		background.RegisterLearnerProfileJobs(jobRegistry, learnerProfileSvc)

		introCourseSvc = introcourseservice.New(pool)
		introcourseservice.RegisterMetrics(tel.Metrics.Registry())
		marketplacecoursesservice.RegisterMetrics(tel.Metrics.Registry())
		background.RegisterIntroCourseJobs(jobRegistry, introCourseSvc, platform)
		if merged.IntroCourseEnabled {
			if _, err := introCourseSvc.EnsureProvisioned(ctx, merged); err != nil {
				slog.Warn("intro course startup provision failed", "err", err)
			}
			if _, err := introcourseservice.EnqueueBackfillIfNeeded(ctx, pool, merged); err != nil {
				slog.Warn("intro course backfill enqueue failed", "err", err)
			}
		}
		if merged.FFCourseMarketplace {
			mcSvc := marketplacecoursesservice.New(pool)
			if courses, err := mcSvc.EnsureDeployProvisioned(ctx, merged); err != nil {
				slog.Warn("marketplace courses startup provision failed", "err", err)
			} else if len(courses) > 0 {
				slog.Info("marketplace courses startup provision complete", "count", len(courses))
			}
		}
	}
	deps := httpserver.Deps{
		Pool:                      pool,
		Health:                    httpserver.NewHealthProbe(healthPool, redisClient, tel.Metrics),
		Redis:                     redisClient,
		ObjectCache:               objectcache.New(redisClient, func() bool { return platform.Config().FFRedisCache }),
		JWTSigner:                 jwtSigner,
		Config:                    cfg,
		Platform:                  platform,
		OIDC:                      oidcauth.NewService(merged),
		Comm:                      commevents.New(),
		Lti:                       ltiRT,
		BrandingResolver:          brandingResolver,
		NotifHub:                  notifevents.New(),
		BannerHub:                 bannerevents.New(),
		FeedHub:                   feedevents.New(),
		CanvasImportHub:           canvasImportHub,
		CanvasImportQueue:         canvasImportQueue,
		CanvasSubmissionSyncHub:   canvasSubmissionSyncHub,
		CanvasSubmissionSyncQueue: canvasSubmissionSyncQueue,
		CanvasSubmissionSyncJobs:  canvasSubmissionSyncJobs,
		GradingAgentQueue:         gradingAgentQueue,
		SmsNotificationQueue:      smsNotificationQueue,
		Scheduler:                 sched,
		Storage:                   storage,
		StorageQuota:              quotaSvc,
		Integrations:              integrations.NewService(pool, integrationsPublicBase(merged), []byte(cfg.JWTSecret)),
		Bots:                      botsservice.NewFromConfig(merged, pool, integrationsPublicBase(merged)),
		Telemetry:                 tel,
		LearnerProfileService:     learnerProfileSvc,
		IntroCourseService:        introCourseSvc,
	}
	// Re-wire T10 transcript notify hooks with the live NotifHub for SSE bell updates.
	background.RegisterTranscriptNotifyHooks(pool, platform.Config(), deps.NotifHub)
	background.StartCanvasImportConsumer(ctx, canvasImportQueue, deps)
	background.StartCanvasSubmissionSyncConsumer(ctx, canvasSubmissionSyncQueue, deps)
	background.StartGradingAgentConsumer(ctx, gradingAgentQueue, deps)
	background.StartSmsNotificationConsumer(ctx, smsNotificationQueue, pool, merged)
	// Internal metrics server (plan 17.7 FR-1 / NFR Security): /metrics is served
	// on a separate port that is firewalled to the VPC and never exposed via the
	// public load balancer (AC-6). It runs independently of the main API so a
	// scrape can succeed even while the API is saturated.
	var metricsSrv *http.Server
	if cfg.Observability.MetricsEnabled && cfg.Observability.MetricsAddr != "" {
		mux := http.NewServeMux()
		mux.Handle("/metrics", tel.MetricsHandler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		metricsSrv = &http.Server{Addr: cfg.Observability.MetricsAddr, Handler: mux}
		go func() {
			slog.Info("metrics server started", "addr", cfg.Observability.MetricsAddr)
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("metrics server failed", "err", err)
			}
		}()
	}

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpserver.NewHandler(deps),
	}
	slog.Info("http server started", "addr", cfg.HTTPAddr, "port_env", strings.TrimSpace(os.Getenv("PORT")))
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		// Drain in-flight requests on SIGTERM before exiting so rolling restarts
		// behind the load balancer do not drop requests (plan 17.2 FR-8 / AC-4).
		drain := time.Duration(cfg.ShutdownTimeoutSecs) * time.Second
		if drain <= 0 {
			drain = 30 * time.Second
		}
		slog.Info("http server shutting down", "drain", drain.String())
		shctx, cancel := context.WithTimeout(context.Background(), drain)
		defer cancel()
		_ = srv.Shutdown(shctx)
		if metricsSrv != nil {
			_ = metricsSrv.Shutdown(shctx)
		}
		<-errCh
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// integrationsPublicBase returns the externally reachable server base URL used
// to build OAuth redirect URIs for inbound integrations (plan 16.4).
func integrationsPublicBase(cfg config.Config) string {
	if cfg.OIDCPublicBaseURL != "" {
		return cfg.OIDCPublicBaseURL
	}
	return cfg.SAMLPublicBaseURL
}

// telemetryConfig maps the app configuration onto the telemetry package's
// decoupled config (plan 17.7).
func telemetryConfig(cfg config.Config) telemetry.Config {
	o := cfg.Observability
	return telemetry.Config{
		ServiceName: o.ServiceName,
		Version:     o.Version,
		Environment: cfg.AppEnv,
		DeployColor: o.DeployColor,
		OTel: telemetry.OTelConfig{
			Endpoint:    o.OTelEndpoint,
			Insecure:    o.OTelInsecure,
			SampleRatio: o.OTelSampleRatio,
		},
		Sentry: telemetry.SentryConfig{
			DSN:              o.SentryDSN,
			TracesSampleRate: o.SentryTracesSampleRate,
		},
	}
}

// telemetrySources builds the live snapshot closures the metrics collector reads
// at scrape time. Each closure tolerates a nil sub-system (plan 17.7 FR-1).
func telemetrySources(pool *pgxpool.Pool, rc *redisclient.Client) telemetry.Sources {
	var s telemetry.Sources
	if pool != nil {
		s.DBPool = func() telemetry.DBPoolSnapshot {
			st := pool.Stat()
			return telemetry.DBPoolSnapshot{
				Total:        st.TotalConns(),
				Acquired:     st.AcquiredConns(),
				Idle:         st.IdleConns(),
				Max:          st.MaxConns(),
				Constructing: st.ConstructingConns(),
			}
		}
		s.JobQueue = func() (telemetry.JobQueueSnapshot, bool) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			st, err := jobqueue.GetStats(ctx, pool)
			if err != nil {
				return telemetry.JobQueueSnapshot{}, false
			}
			return telemetry.JobQueueSnapshot{
				Pending:     st.Pending,
				Running:     st.Running,
				Failed:      st.Failed,
				DeadLetters: st.DeadLetters,
				Depth:       st.Depth,
				ByType:      st.ByType,
			}, true
		}
	}
	if rc != nil {
		s.Redis = func() telemetry.RedisPoolSnapshot {
			st := rc.Redis().PoolStats()
			return telemetry.RedisPoolSnapshot{
				Total:    st.TotalConns,
				Idle:     st.IdleConns,
				Stale:    st.StaleConns,
				Hits:     uint64(st.Hits),
				Misses:   uint64(st.Misses),
				Timeouts: uint64(st.Timeouts),
			}
		}
	}
	return s
}

func isUndefinedTable(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "42P01"
}

func webHostFromOrigin(origin string) string {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" {
		return ""
	}
	return orgbranding.NormalizeHost(u.Host)
}
