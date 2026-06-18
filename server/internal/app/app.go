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

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/canvasimportevents"
	"github.com/lextures/lextures/server/internal/canvasimportqueue"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncevents"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncjobs"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
	"github.com/lextures/lextures/server/internal/commevents"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/feedevents"
	"github.com/lextures/lextures/server/internal/httpserver"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/lti"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/platformstate"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/oidcauth"
	"github.com/lextures/lextures/server/internal/service/storagequota"
)

// Run starts the API. Pass the migration file tree (e.g. serverdata.Migrations from the module root).
func Run(ctx context.Context, fsys fs.FS) error {
	cfg := config.Load()
	logging.Configure(logging.Settings{
		DisableRedaction: cfg.DisablePIIRedaction,
		ExtraFields:      cfg.PIIRedactFields,
		HMACSecret:       []byte(cfg.JWTSecret),
		AppEnv:           cfg.AppEnv,
	})
	if err := cfg.Validate(); err != nil {
		return err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("app: database: %w", err)
	}
	defer pool.Close()
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

	storage, storageErr := filestorage.New(filestorage.BackendConfig{
		Backend:         cfg.StorageBackend,
		LocalRoot:       cfg.CourseFilesRoot,
		Endpoint:        cfg.StorageEndpoint,
		AccessKeyID:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Bucket:          cfg.StorageBucket,
		UseSSL:          cfg.StorageUseSSL,
		Region:          cfg.StorageRegion,
	})
	if storageErr != nil {
		return fmt.Errorf("app: storage: %w", storageErr)
	}

	smsNotificationQueue, smsQueueErr := smsnotificationqueue.NewBus(
		merged.RabbitMQURL,
		merged.SmsNotificationQueueName,
		merged.SmsNotificationConcurrency,
	)
	if smsQueueErr != nil {
		return fmt.Errorf("app: sms notification queue: %w", smsQueueErr)
	}
	defer func() { _ = smsNotificationQueue.Close() }()

	background.StartWithStorage(ctx, pool, merged, storage, smsNotificationQueue)

	ltiRT := lti.NewFromConfig(merged)
	brandingResolver := orgbranding.NewResolver(pool, merged.BrandingMultitenantHostSuffix, webHostFromOrigin(merged.PublicWebOrigin))

	var quotaSvc *storagequota.Service
	if merged.StorageQuotasEnabled {
		quotaSvc = &storagequota.Service{Pool: pool}
	}

	canvasImportHub := canvasimportevents.New()
	canvasImportQueue, queueErr := canvasimportqueue.NewBus(merged.RabbitMQURL, merged.CanvasImportQueueName, merged.CanvasImportConcurrency)
	if queueErr != nil {
		return fmt.Errorf("app: canvas import queue: %w", queueErr)
	}
	defer func() { _ = canvasImportQueue.Close() }()

	canvasSubmissionSyncHub := canvassubmissionsyncevents.New()
	canvasSubmissionSyncJobs := canvassubmissionsyncjobs.NewRegistry()
	canvasSubmissionSyncQueue, syncQueueErr := canvassubmissionsyncqueue.NewBus(
		merged.RabbitMQURL,
		merged.CanvasSubmissionSyncQueueName,
		merged.CanvasSubmissionSyncConcurrency,
	)
	if syncQueueErr != nil {
		return fmt.Errorf("app: canvas submission sync queue: %w", syncQueueErr)
	}
	defer func() { _ = canvasSubmissionSyncQueue.Close() }()

	deps := httpserver.Deps{
		Pool:              pool,
		JWTSigner:         auth.NewJWTSignerWithPool(cfg.JWTSecret, pool),
		Config:            cfg,
		Platform:          platformstate.New(merged),
		OIDC:              oidcauth.NewService(merged),
		Comm:              commevents.New(),
		Lti:               ltiRT,
		BrandingResolver:  brandingResolver,
		NotifHub:          notifevents.New(),
		FeedHub:           feedevents.New(),
		CanvasImportHub:           canvasImportHub,
		CanvasImportQueue:         canvasImportQueue,
		CanvasSubmissionSyncHub:   canvasSubmissionSyncHub,
		CanvasSubmissionSyncQueue: canvasSubmissionSyncQueue,
		CanvasSubmissionSyncJobs:  canvasSubmissionSyncJobs,
		SmsNotificationQueue:      smsNotificationQueue,
		Storage:                   storage,
		StorageQuota:      quotaSvc,
	}
	background.StartCanvasImportConsumer(ctx, canvasImportQueue, deps)
	background.StartCanvasSubmissionSyncConsumer(ctx, canvasSubmissionSyncQueue, deps)
	background.StartSmsNotificationConsumer(ctx, smsNotificationQueue, pool, merged)
	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpserver.NewHandler(deps),
	}
	slog.Info("http server started", "addr", cfg.HTTPAddr, "port_env", strings.TrimSpace(os.Getenv("PORT")))
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(shctx)
		<-errCh
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
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
