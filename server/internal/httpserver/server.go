package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/hibp"
	"github.com/lextures/lextures/server/internal/canvasimportevents"
	"github.com/lextures/lextures/server/internal/canvasimportqueue"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncevents"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncjobs"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
	"github.com/lextures/lextures/server/internal/commevents"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/feedevents"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/lti"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/objectcache"
	"github.com/lextures/lextures/server/internal/openapi"
	"github.com/lextures/lextures/server/internal/platformstate"
	"github.com/lextures/lextures/server/internal/redisclient"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/scheduler"
	botsservice "github.com/lextures/lextures/server/internal/service/bots"
	"github.com/lextures/lextures/server/internal/service/cleverauth"
	drmservice "github.com/lextures/lextures/server/internal/service/drm"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	integrationsservice "github.com/lextures/lextures/server/internal/service/integrations"
	"github.com/lextures/lextures/server/internal/service/oidcauth"
	"github.com/lextures/lextures/server/internal/service/openrouter"
	statuspageservice "github.com/lextures/lextures/server/internal/service/statuspage"
	"github.com/lextures/lextures/server/internal/service/storagequota"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Deps is the minimal set of server dependencies. Expand with auth, LTI, etc. during the migration.
type Deps struct {
	Pool      *pgxpool.Pool
	Ready     ReadyChecker
	JWTSigner *auth.JWTSigner
	// Config is the environment-only configuration (used with DB overrides in Platform).
	Config   config.Config
	Platform *platformstate.Platform
	OIDC     *oidcauth.Service
	Clever   *cleverauth.Service
	Comm     *commevents.Hub
	Lti      *lti.Runtime
	// BrandingResolver caches hostname→org branding (plan 5.7). Optional; nil builds an ephemeral resolver per request group via brandingResolver().
	BrandingResolver *orgbranding.Resolver
	// PasswordChecker overrides HIBP / password breach checks (tests). When nil, a production checker is built from Pool.
	PasswordChecker hibp.Checker
	// NotifHub broadcasts SSE signals for real-time in-app notification bell updates (plan 6.3). Optional.
	NotifHub *notifevents.Hub
	// FeedHub fans out course-feed change signals to WebSocket subscribers so the
	// SPA refreshes in real time when channels/messages change (including via CLI). Optional.
	FeedHub *feedevents.Hub
	// CanvasImportHub fans out queued Canvas import progress to WebSocket subscribers.
	CanvasImportHub *canvasimportevents.Hub
	// CanvasImportQueue publishes Canvas import jobs to RabbitMQ (or in-memory fallback).
	CanvasImportQueue *canvasimportqueue.Bus
	// CanvasSubmissionSyncHub fans out queued Canvas grade-push results to WebSocket subscribers.
	CanvasSubmissionSyncHub *canvassubmissionsyncevents.Hub
	// CanvasSubmissionSyncQueue publishes Canvas grade-push jobs to RabbitMQ (or in-memory fallback).
	CanvasSubmissionSyncQueue *canvassubmissionsyncqueue.Bus
	// CanvasSubmissionSyncJobs tracks in-flight Canvas grade-push jobs for auth and reconnect.
	CanvasSubmissionSyncJobs *canvassubmissionsyncjobs.Registry
	// SmsNotificationQueue publishes SMS notification jobs to RabbitMQ (or in-memory fallback).
	SmsNotificationQueue *smsnotificationqueue.Bus
	// GradingAgentQueue publishes grading-agent batch jobs to RabbitMQ (or in-memory fallback).
	GradingAgentQueue *gradingagentqueue.Bus
	// Scheduler exposes the configured scheduled jobs for the admin scheduler API
	// (plan 17.4). When nil, scheduler admin endpoints return 501.
	Scheduler *scheduler.Scheduler
	// Storage is the object-storage driver (plan 8.1). When nil, falls back to local disk reads.
	Storage filestorage.Driver
	// DRM is the DRM / watermarking service (plan 8.10). When nil, DRM endpoints return 501.
	DRM *drmservice.Service
	// StorageQuota enforces per-tenant/course/user storage limits (plan 8.5).
	// When nil, quota endpoints return 501 and upload enforcement is skipped.
	StorageQuota *storagequota.Service
	// Integrations powers inbound third-party connectors (Google Classroom,
	// Teams, Canva, LTI 1.1 embeds) — plan 16.4. When nil, endpoints return 501.
	Integrations *integrationsservice.Service
	// Bots powers Slack/Teams/Discord classroom bots (plan 16.6). When nil, endpoints return 501.
	Bots *botsservice.Service
	// StatusPageClient overrides the default Statuspage.io client (tests). When nil, built from Config.
	StatusPageClient *statuspageservice.Client
	// Redis is the shared Redis client for cross-instance state (plan 17.2). When
	// nil, the server runs single-instance and the readiness probe skips Redis.
	Redis *redisclient.Client
	// ObjectCache is the Redis-backed object cache for hot read paths (plan 17.5).
	ObjectCache *objectcache.Service
	// Telemetry installs the observability middleware chain (Prometheus metrics,
	// OpenTelemetry spans, X-Trace-Id header, Sentry panic capture) — plan 17.7.
	// When nil, the server runs without instrumentation.
	Telemetry *telemetry.Telemetry
}

func (d Deps) effectiveConfig() config.Config {
	if d.Platform != nil {
		return d.Platform.Config()
	}
	return d.Config
}

func (d Deps) openRouterClient() *openrouter.Client {
	if d.Platform != nil {
		return d.Platform.OpenRouter()
	}
	return nil
}

// NewHandler builds the HTTP API (routes only; does not start listening).
func NewHandler(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(corsAll)
	r.Use(middleware.RequestID)
	// Observability runs outermost (after RequestID) so traces, the X-Trace-Id
	// header, and HTTP metrics cover every request — including those later
	// rejected by rate limiting (plan 17.7 FR-1/FR-2/FR-8).
	if d.Telemetry != nil {
		for _, mw := range d.Telemetry.ObserveMiddlewares() {
			r.Use(mw)
		}
	}
	// Rate limiting runs before RealIP so it sees the genuine TCP peer and can
	// reject forged X-Forwarded-For headers from untrusted clients (plan 17.6).
	r.Use(d.rateLimitMiddleware(d.buildRateLimiter()))
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// Sentry panic capture sits just inside chi's Recoverer so it reports the
	// panic (with PII scrubbed) before Recoverer converts it to a 500
	// (plan 17.7 FR-3, AC-3). No-op when Sentry is disabled.
	if d.Telemetry != nil {
		r.Use(d.Telemetry.SentryRecoverMiddleware)
	}
	r.Use(middleware.Compress(5))
	r.Use(logging.AccessLog)
	r.Use(authenticatedNoStoreMiddleware)
	r.Use(d.publicAPIMiddleware)
	ready := d.Ready
	if ready == nil {
		ready = defaultReady(d.Pool, d.Redis)
	}
	r.Get("/api/openapi.json", openapi.ServeOpenAPI)
	r.Get("/api/docs", openapi.ServeDocs)
	r.Get("/health", handleHealth())
	r.Get("/health/ready", handleReady(ready))
	r.Post("/api/v1/public/onboarding/track", d.handlePublicOnboardingTrack())
	r.Get("/api/v1/public/branding/resolve", d.handlePublicBrandingResolve())
	r.Get("/api/v1/public/orgs/by-slug/{slug}", d.handlePublicOrgBySlug())
	r.Get("/api/v1/public/locale-defaults", d.handleGetPublicLocaleDefaults())
	r.Get("/api/v1/public/org-branding/{orgId}/{asset}", d.handlePublicOrgBrandAsset())
	d.registerPublicCatalogRoutes(r)
	r.Get("/api/v1/search", d.handleSearchIndex())
	r.Get("/api/v1/search/query", d.handleSearchQuery())
	r.Get("/api/v1/reports/learning-activity", d.handleLearningActivityReport())
	d.registerReportExportRoutes(r)
	r.Post("/api/v1/recommendations/event", d.handleRecommendationEvent())
	r.Post("/api/v1/webhooks/originality/{provider}", d.handleOriginalityWebhook())
	r.Post("/api/v1/webhooks/proctoring-callback/{vendor}", d.handleProctoringCallback())
	r.Get("/oneroster/v1p2/*", d.handleOneRosterV1P2())
	d.registerSAMLBrowserRoutes(r)
	d.registerLTIHTTPRoutes(r)
	d.registerAuthRoutes(r)
	d.registerMeRoutes(r)
	d.registerParentRoutes(r)
	d.registerUserRoutes(r)
	d.registerOrgRoutes(r)
	d.registerCourseRoutes(r)
	d.registerMeetingRoutes(r)
	d.registerOfficeHoursRoutes(r)
	d.registerTutorRoutes(r)
	d.registerStudyBuddyRoutes(r)
	d.registerSurveyRoutes(r)
	d.registerLearnerRoutes(r)
	d.registerConceptRoutes(r)
	d.registerDiagnosticRoutes(r)
	d.registerCommunicationRoutes(r)
	d.registerImportRoutes(r)
	d.registerStandardsRoutes(r)
	d.registerSettingsRoutes(r)
	d.registerAdminRoutes(r)
	d.registerAdminJobRoutes(r)
	d.registerAdminSchedulerRoutes(r)
	d.registerSCIMRoutes(r)
	r.Route("/api/v1", func(s chi.Router) { d.registerAccommodationRoutes(s) })
	d.registerAttendanceRoutes(r)
	d.registerBehaviorRoutes(r)
	d.registerReportCardRoutes(r)
	d.registerSBGReportRoutes(r)
	d.registerSISRoutes(r)
	d.registerWebhookRoutes(r)
	d.registerCalendarFeedRoutes(r)
	d.registerFinalGradeRoutes(r)
	d.registerCatalogRoutes(r)
	d.registerLibraryRoutes(r)
	d.registerHELibraryRoutes(r)
	d.registerBookstoreRoutes(r)
	d.registerEportfolioRoutes(r)
	d.registerTranscriptsRoutes(r)
	d.registerAdvisingRoutes(r)
	d.registerResearchConsentRoutes(r)
	d.registerConsortiumRoutes(r)
	d.registerBillingRoutes(r)
	d.registerPaymentsRoutes(r)
	d.registerTaxRoutes(r)
	d.registerRevenueShareRoutes(r)
	d.registerLearningPathRoutes(r)
	d.registerAccessibilityRoutes(r)
	d.registerBroadcastRoutes(r)
	d.registerClassroomSignalsRoutes(r)
	d.registerConferenceRoutes(r)
	d.registerDemographicsRoutes(r)
	d.registerContentFilterRoutes(r)
	d.registerUIModeRoutes(r)
	r.Get("/api/v1/help/contextual-articles", d.handleHelpContextualArticles())
	d.registerTranslationRoutes(r)
	d.registerReadingLevelRoutes(r)
	d.registerAltTextRoutes(r)
	d.registerCourseTranslationRoutes(r)
	d.registerTusRoutes(r)
	d.registerTranscodeRoutes(r)
	d.registerCanvasImportRoutes(r)
	d.registerCanvasSubmissionSyncRoutes(r)
	d.registerCaptionRoutes(r)
	d.registerStorageQuotaRoutes(r)
	d.registerAVScanRoutes(r)
	r.Post("/api/v1/xapi/statements", d.handlePostXAPIStatements())
	r.Post("/api/v1/scorm/rte/{registration_id}/commit", d.handlePostScormRTECommit())
	d.registerEngagementRoutes(r)
	d.registerSeatTimeRoutes(r)
	d.registerInsightsRoutes(r)
	d.registerOERRoutes(r)
	d.registerCloudProviderRoutes(r)
	d.registerIntegrationRoutes(r)
	d.registerBotRoutes(r)
	d.registerMarketplaceRoutes(r)
	d.registerLegalRoutes(r)
	d.registerTrustRoutes(r)
	d.registerFERPARoutes(r)
	d.registerCoppaRoutes(r)
	d.registerGDPRRoutes(r)
	d.registerCCPARoutes(r)
	d.registerDPARoutes(r)
	d.registerStatePrivacyRoutes(r)
	d.registerSOC2Routes(r)
	d.registerISORoutes(r)
	d.registerAdminAuditLogRoutes(r)
	d.registerSecurityReportsRoutes(r)
	d.registerDataResidencyRoutes(r)
	d.registerAIDisclosureRoutes(r)
	d.registerAIProviderSettingsRoutes(r)
	d.registerBackupOpsRoutes(r)
	d.registerStatusRoutes(r)
	d.registerPIIRedactionRoutes(r)
	d.registerUnimplementedV1(r)
	d.mountRouterErrorHandlers(r)
	return r
}

// defaultReady builds a readiness probe that verifies the database and, when
// configured, the shared Redis instance (plan 17.2 FR-1: GET /health/ready feeds
// the load balancer, which must include Redis connectivity). Redis is only
// checked when present so single-instance deployments stay healthy without it.
func defaultReady(p *pgxpool.Pool, rc *redisclient.Client) ReadyChecker {
	if p == nil {
		return func() error { return errNoDBPool }
	}
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			return err
		}
		if rc != nil {
			if err := rc.Ping(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

var errNoDBPool = &degradedErr{s: "database pool is not configured"}

// degradedErr is a lightweight error type for readiness.
type degradedErr struct{ s string }

func (e *degradedErr) Error() string { return e.s }
