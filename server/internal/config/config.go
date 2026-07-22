// Package config loads process configuration from the environment.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lextures/lextures/server/internal/redisclient"
)

const (
	// JWTSecretMinLen matches the legacy Rust server's minimum accepted JWT secret length.
	JWTSecretMinLen = 32

	insecureJWTFallback = "dev-secret-do-not-use-in-production"

	// defaultShutdownTimeoutSecs is the graceful-shutdown drain window (plan 17.2 FR-8).
	defaultShutdownTimeoutSecs = 30
)

var defaultCanvasAllowedHostSuffixes = []string{"instructure.com"}

// Config holds API server, database, and integration settings.
type Config struct {
	HTTPAddr string

	DatabaseURL      string
	JWTSecret        string
	AllowInsecureJWT bool
	RunMigrations    bool

	// BootstrapAdminEmail, when non-empty, is the only email that may receive Global Admin
	// on the first human password signup (empty DB besides system users). Loaded from BOOTSTRAP_ADMIN_EMAIL.
	BootstrapAdminEmail string

	OpenRouterAPIKey string
	CourseFilesRoot  string

	CanvasAllowedHostSuffixes []string
	PublicWebOrigin           string

	// BrandingMultitenantHostSuffix matches "{slug}.<suffix>" to tenant.organizations.slug (plan 5.7).
	// Example: "lextures.io" maps greenvalley.lextures.io → slug greenvalley. Empty disables subdomain mapping.
	BrandingMultitenantHostSuffix string

	// PlatformSecretsKey is a 32-byte AES-256 key (base64 in PLATFORM_SECRETS_KEY) used to encrypt
	// SMTP passwords and similar values stored in settings.platform_app_settings.
	PlatformSecretsKey []byte

	SMTPHost     string
	SMTPPort     uint16
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// EmailProvider selects the transactional email backend: "smtp" (default) or "ses".
	// Additional providers can be added without changing call sites (mail.SelectProvider).
	// Env: EMAIL_PROVIDER; DB override: email_provider.
	EmailProvider string
	// FFEmailSES enables the Amazon SES email provider (default off).
	// When false, EmailProvider=ses is ignored and SMTP is used.
	// Managed in Settings → Global platform (not process env).
	FFEmailSES bool
	// SESRegion is the AWS region for SES API calls (e.g. us-east-1).
	// Falls back to STORAGE_REGION / AWS_REGION, then us-east-1.
	SESRegion string
	// SESFrom is the verified SES From address. Falls back to SMTPFrom.
	SESFrom string
	// SESConfigurationSet is an optional SES configuration set name.
	SESConfigurationSet string
	// SESAccessKeyID / SESSecretAccessKey are optional static credentials.
	// When empty, the default AWS credential chain is used (env, shared config, IAM role).
	SESAccessKeyID     string
	SESSecretAccessKey string

	LTIEnabled          bool
	LTIAPIBaseURL       string
	LTIRSAPrivateKeyPEM string
	LTIRSAKeyID         string

	AnnotationEnabled           bool
	FeedbackMediaEnabled        bool
	BlindGradingEnabled         bool
	ModeratedGradingEnabled     bool
	OriginalityDetectionEnabled bool
	// OriginalityStubExternal is an env-only test/dev seam (ORIGINALITY_STUB_EXTERNAL).
	// Not a platform settings toggle — see docs/completed/flags.md.
	OriginalityStubExternal     bool
	GradePostingPoliciesEnabled bool
	GradebookCSVEnabled         bool
	ResubmissionWorkflowEnabled bool

	SAMLSSOEnabled      bool
	SAMLPublicBaseURL   string
	SAMLSPEntityID      string
	SAMLSPX509PEM       string
	SAMLSPPrivateKeyPEM string

	OIDCSSOEnabled            bool
	OIDCPublicBaseURL         string
	OIDCGoogleClientID        string
	OIDCGoogleClientSecret    string
	OIDCGoogleHostedDomain    string
	OIDCMicrosoftTenant       string
	OIDCMicrosoftClientID     string
	OIDCMicrosoftClientSecret string
	OIDCAppleClientID         string
	OIDCAppleTeamID           string
	OIDCAppleKeyID            string
	OIDCApplePrivateKeyPEM    string
	// OIDCAppleNativeAudience is the comma-separated allow-list of audiences accepted
	// by POST /api/v1/auth/oidc/apple/native (iOS bundle IDs). Default: com.lextures.ios.
	OIDCAppleNativeAudience string
	// OIDCGoogleNativeAudience is the audience (OAuth web/server client ID) accepted by
	// POST /api/v1/auth/oidc/google/native. When empty, OIDCGoogleClientID is used.
	OIDCGoogleNativeAudience string

	CleverSSOEnabled   bool
	CleverClientID     string
	CleverClientSecret string
	CleverDistrictID   string // optional; skips Clever school picker when set

	ClassLinkSSOEnabled       bool
	ClassLinkOIDCIssuer       string // e.g. https://launchpad.classlink.com/v2_0/sis/{tenant}
	ClassLinkOIDCClientID     string
	ClassLinkOIDCClientSecret string

	OneRosterEnabled             bool
	OneRosterBearerFallbackToken string
	OneRosterBearerFallbackInst  string // UUID string; used with fallback token when DB has no match

	ScimEnabled bool

	// MFAEnabled gates TOTP/WebAuthn MFA (env MFA_ENABLED or DB override).
	MFAEnabled bool
	// MFAEnforcement is none | all | staff (platform setting; staff = Teacher/TA/Global Admin).
	MFAEnforcement string

	// MagicLinkEnabled allows email one-time sign-in links (plan 4.7).
	MagicLinkEnabled bool
	// MagicLinkEnrolledOnly when true: only users with an active course enrollment receive a link.
	MagicLinkEnrolledOnly bool

	// TurnstileSecretKey is the Cloudflare Turnstile secret for signup CAPTCHA server-side verification.
	TurnstileSecretKey string

	// SessionManagementUIEnabled gates /api/v1/me/sessions and related UI (plan 4.9).
	SessionManagementUIEnabled bool

	// EmailNotificationsEnabled gates event-driven transactional email (plan 6.2).
	EmailNotificationsEnabled bool

	// BackgroundJobsEnabled gates the generic Postgres-backed background job
	// queue worker (plan 17.3, rollout flag background_jobs_enabled). Defaults on
	// when APP_ENV=local so manual triggers work in dev without extra env vars.
	BackgroundJobsEnabled bool
	// BackgroundJobsConcurrency is the number of jobs processed in parallel per
	// app instance (default 4 when unset).
	BackgroundJobsConcurrency int

	// SchedulerEnabled gates the cron-like scheduled-job layer (plan 17.4,
	// rollout flag scheduler_enabled). It requires BackgroundJobsEnabled because
	// scheduled triggers enqueue onto the generic job queue.
	SchedulerEnabled bool

	// PushNotificationsEnabled gates Web Push (VAPID) notifications (plan 6.3).
	PushNotificationsEnabled bool
	// SmsNotificationsEnabled gates SMS notification enqueueing (Twilio delivery).
	SmsNotificationsEnabled bool
	// VAPIDPublicKey is the base64url-encoded P-256 public key for VAPID.
	VAPIDPublicKey string
	// VAPIDPrivateKey is the base64url-encoded P-256 private key for VAPID signing.
	VAPIDPrivateKey string
	// VAPIDSubject is the mailto: or https: URI sent in the VAPID JWT sub claim.
	VAPIDSubject string

	// Native push (APNs / FCM) credentials for mobile apps (plans 21.5, M0.1).
	APNSP8Key             string
	APNSKeyID             string
	APNSTeamID            string
	APNSBundleID          string
	APNSEnvironment       string // "development" or "production"
	FCMServiceAccountJSON string

	// VirtualClassroomEnabled gates the virtual meeting / live classroom feature (plan 6.4).
	VirtualClassroomEnabled bool

	// JitsiBaseURL is the Jitsi Meet server base URL (e.g. "https://meet.jit.si"). Defaults to meet.jit.si.
	JitsiBaseURL string
	// JitsiAppID is the Jitsi app_id for JWT signing. Empty disables JWT room tokens.
	JitsiAppID string
	// JitsiAppSecret is the HMAC-SHA256 secret for Jitsi JWT signing. Empty uses unsigned public rooms.
	JitsiAppSecret string

	// BBBBaseURL is the BigBlueButton API base URL (e.g. "https://bbb.example.com/bigbluebutton").
	BBBBaseURL string
	// BBBSecret is the BigBlueButton shared secret for request signing.
	BBBSecret string

	// StorageBackend selects the file storage driver: "local" (default), "s3", "r2", "minio".
	StorageBackend string
	// StorageBucket is the S3-compatible bucket name.
	StorageBucket string
	// StorageEndpoint is the S3-compatible endpoint host (e.g. "play.min.io:9000"). Empty defaults per backend.
	StorageEndpoint string
	// StorageAccessKeyID is the S3 access key ID.
	StorageAccessKeyID string
	// StorageSecretAccessKey is the S3 secret access key.
	StorageSecretAccessKey string
	// StorageRegion is the AWS/R2 region (e.g. "us-east-1"). Optional for MinIO.
	StorageRegion string
	// StorageCDNBaseURL is the CDN origin for public media URLs (plan 17.5 FR-5).
	// When set, presigned download URLs are rewritten to use this host (e.g. https://cdn.example.com).
	StorageCDNBaseURL string
	// StorageUseSSL controls TLS for S3 connections (default true for s3/r2, false for minio in dev).
	StorageUseSSL bool
	// StoragePresignTTL is the presigned URL TTL in seconds (default 3600).
	StoragePresignTTL int
	// StorageMigrateLocal, when true, copies existing local files to the object store on startup.
	StorageMigrateLocal bool

	// TusUploadTTLHours is how long stalled (incomplete) tus uploads are retained before cleanup (default 48).
	TusUploadTTLHours int

	// DRMEnabled gates watermarking and DRM features (plan 8.10). Defaults to false; enterprise-only.
	DRMEnabled bool
	// DRMHMACSecret is the base64-encoded 32-byte secret used to sign user-bound DRM tokens.
	// If empty, token signing falls back to a derived value (not suitable for production).
	DRMHMACSecret string

	// VideoTranscodingEnabled gates HLS transcoding (plan 8.3). Defaults to false.
	VideoTranscodingEnabled bool
	// TranscodeRetainSourceDays is how long the original raw upload is kept after a successful
	// transcode before background deletion. 0 means keep indefinitely (default 30).
	TranscodeRetainSourceDays int
	// FFmpegPath is the path to the ffmpeg binary. Defaults to "ffmpeg" (from PATH).
	FFmpegPath string

	// AutoCaptioningEnabled gates auto-captioning via Whisper (plan 8.4). Defaults to false.
	AutoCaptioningEnabled bool
	// VideoCaptionsEnabled gates caption editor, player controls, and compliance UI (plan 12.4).
	VideoCaptionsEnabled bool
	// WhisperBackend selects the ASR backend: whisper-api (default), whisper-local, azure-speech, google-speech, stub.
	WhisperBackend string
	// OpenAIAPIKey is the OpenAI API key used when WhisperBackend=whisper-api.
	OpenAIAPIKey string

	// StorageQuotasEnabled gates per-course/user storage quota enforcement (plan 8.5).
	StorageQuotasEnabled bool
	// StorageDefaultTenantQuotaGB, when > 0, sets a default tenant-level quota in gigabytes
	// applied at startup. 0 means unlimited unless an admin sets an explicit limit.
	StorageDefaultTenantQuotaGB int64

	// AtRiskAlertsEnabled gates at-risk scoring, alerts, and instructor UI (plan 9.2).
	AtRiskAlertsEnabled bool

	// AvScanningEnabled gates ClamAV malware scanning on uploads (plan 8.6).
	AvScanningEnabled bool

	// H5PEnabled gates interactive H5P module items and xAPI completion (plan 8.12).
	H5PEnabled bool
	// ScormIngestionEnabled gates SCORM/cmi5 package upload and player (plan 2.14).
	ScormIngestionEnabled bool
	// ClamAVAddr is the clamd TCP address (default localhost:3310).
	ClamAVAddr string
	// ClamAVStub when true uses in-process EICAR detection (tests/dev without clamd).
	// Env-only (CLAMAV_STUB); not a platform settings toggle.
	ClamAVStub bool

	// OERLibraryEnabled gates the OER search and import UI (plan 8.9).
	OERLibraryEnabled bool
	// OERStub uses embedded catalog data instead of live OER provider APIs (dev/e2e).
	// Env-only (OER_STUB); not a platform settings toggle.
	OERStub bool

	// ItemAnalysisEnabled gates CTT item analysis statistics for quizzes (plan 9.4).
	ItemAnalysisEnabled bool
	// StudentProgressEnabled gates per-student progress dashboards (plan 9.1).
	StudentProgressEnabled bool
	// OutcomesReportEnabled gates course-level outcomes achievement reporting (plan 9.5).
	OutcomesReportEnabled bool

	// EngagementTrackingEnabled gates engagement metrics collection and reporting (plan 9.7).
	EngagementTrackingEnabled bool
	// SelfReflectionEnabled gates learner study stats, journal, and coaching tips (plan 9.9).
	SelfReflectionEnabled bool
	// LearnerProfileEnabled gates the autonomous cross-course learner profile (LP01).
	LearnerProfileEnabled bool
	// LpAdaptRecommendationsEnabled gates profile-driven recommendation ranking (LP09).
	LpAdaptRecommendationsEnabled bool
	// LpAdaptReviewEnabled gates profile-driven SRS review prioritisation (LP09).
	LpAdaptReviewEnabled bool
	// LpAdaptModalityEnabled gates profile-driven content-format preference (LP09).
	LpAdaptModalityEnabled bool
	// LpAdaptTutorEnabled gates profile-driven tutor scaffolding (LP09).
	LpAdaptTutorEnabled bool
	// IntroCourseEnabled gates the canonical "Welcome to Lextures" intro course (IC01).
	IntroCourseEnabled bool
	// InstructorInsightsEnabled gates the "What's Working" instructor signals dashboard (plan 9.10).
	InstructorInsightsEnabled bool

	// EquationEditorEnabled gates the visual equation editor in the web client (plan 8.11).
	EquationEditorEnabled bool
	// ReadingLevelEnabled gates Flesch-Kincaid scoring and AI content simplification (plan 11.6).
	ReadingLevelEnabled bool
	// GraderAgentEnabled enables the instructor-authored grading agent in SpeedGrader (plan 19.16).
	GraderAgentEnabled bool
	// GraderAgentReviewInboxEnabled enables the persistent review inbox and run history (GA-M1).
	GraderAgentReviewInboxEnabled bool
	// GraderAgentSuggestModeEnabled enables suggest-only runs and bulk review actions (GA-M3).
	GraderAgentSuggestModeEnabled bool
	// GraderAgentTextEntryGradingEnabled grades typed online text-entry submissions (GA-M2).
	GraderAgentTextEntryGradingEnabled bool
	// GraderAgentVisionGradingEnabled grades image/scanned submissions via vision models (GA-M2).
	GraderAgentVisionGradingEnabled bool
	// GraderAgentRunFiltersEnabled allows section/group/selection filters on batch runs (GA-M5).
	GraderAgentRunFiltersEnabled bool
	// GraderAgentCostEstimateEnabled shows pre-run cost estimates and optional per-run budgets (GA-M7).
	GraderAgentCostEstimateEnabled bool
	// GraderAgentCancelRunEnabled allows cancelling in-progress batch runs (GA-M6).
	GraderAgentCancelRunEnabled bool
	// CodeExecutionEnabled enables sandboxed code execution for quiz and grader agent nodes (plan 2.4 / 19.17.7).
	CodeExecutionEnabled bool
	// AltTextEnforcementEnabled gates alt-text prompts, AI suggestions, and coverage reporting (plan 12.5).
	AltTextEnforcementEnabled bool
	// FFAltTextEnforcement when true hard-blocks content save until alt text is resolved (plan 12.5).
	FFAltTextEnforcement bool
	// FFHighContrastReducedMotion enables the high-contrast/reduced-motion preference panel and API (plan 12.7).
	FFHighContrastReducedMotion bool
	// FFMotionNavigation enables splash handoff and route/screen/tab transitions (AN.2). Default ON; kill-switch.
	FFMotionNavigation bool
	// FFMotionReveal enables skeleton→content crossfade and staggered entrances (AN.3). Default ON; kill-switch.
	FFMotionReveal bool
	// FFMotionLists enables list insert/remove/reorder and drag-lift motion (AN.4). Default ON; kill-switch.
	FFMotionLists bool
	// FFMotionOverlays enables dialog/sheet/menu/toast/tooltip enter-exit motion (AN.5). Default ON; kill-switch.
	FFMotionOverlays bool
	// FFMotionControls enables control micro-interactions (press, toggle/tabs, validation, haptics) (AN.6). Default ON; kill-switch.
	// FFMotionDelight enables delight & progress moments (progress fills, quiz feedback, achievement bursts) (AN.7). Default ON; kill-switch.
	FFMotionDelight  bool
	FFMotionControls bool
	// FFMobileCreateCourse is always on (platform master removed). Mobile New course / create wizard.
	FFMobileCreateCourse bool
	// FFMobileCourseCreateV2 is always on (collapsed into create-course; platform master removed).
	FFMobileCourseCreateV2 bool
	// FFMobileCanvasImport is always on (platform master removed). Mobile Canvas import wizard.
	FFMobileCanvasImport bool
	// FFMobileAdminConsole is always on (platform master removed). Mobile Settings/Admin hub.
	FFMobileAdminConsole bool
	// FFMobileEnrollmentAdd is always on (platform master removed). Mobile People roster add.
	FFMobileEnrollmentAdd bool
	// FFMobileLiveQuiz is always on (platform master removed). Interactive live quizzes on mobile.
	FFMobileLiveQuiz bool
	// FFMobileWhiteboardEdit is always on (platform master removed). Course whiteboard authoring on mobile.
	FFMobileWhiteboardEdit bool
	// FFMobileMarketplacePurchase is always on (platform master removed). Mobile marketplace claim/buy.
	FFMobileMarketplacePurchase bool
	// FFMobileBoardsAdvanced is always on (platform master removed). Board templates/export/present/governance.
	FFMobileBoardsAdvanced bool
	// SpeechToTextEnabled gates browser dictation in block editor and quiz fields (plan 12.9).
	SpeechToTextEnabled bool
	// AccommodationsEngineEnabled gates the K-12 accommodations engine (plan 12.10).
	AccommodationsEngineEnabled bool
	// FFAccommodationsEngine when true writes accommodation audit log entries (plan 12.10).
	FFAccommodationsEngine bool
	// ReadAloudEnabled gates in-context read-aloud on course content pages (plan 12.8).
	ReadAloudEnabled bool
	// FFReadAloud when true exposes read-aloud controls to learners (plan 12.8).
	FFReadAloud bool
	// TranslationMemoryEnabled gates course content translation workflow and TM (plan 11.5).
	TranslationMemoryEnabled bool

	// ReportExportEnabled gates PDF export and scheduled report delivery (plan 9.8).
	ReportExportEnabled bool
	// XAPIEmissionEnabled gates Caliper/xAPI learning event storage and LRS forwarding (plan 9.6).
	XAPIEmissionEnabled bool
	// LRSAnonymizeActors hashes actor mbox emails in emitted xAPI statements (plan 9.6 AC-4).
	LRSAnonymizeActors bool

	// FERPAWorkflowEnabled gates FERPA directory opt-out, record-access requests, and disclosure log (plan 10.1).
	FERPAWorkflowEnabled bool
	// CoppaWorkflowEnabled gates COPPA verifiable parental consent for K-12 deployments (plan 10.2).
	CoppaWorkflowEnabled bool
	// GDPRModuleEnabled gates GDPR/UK GDPR DSAR workflow, consent management, and RoPA (plan 10.3).
	GDPRModuleEnabled bool
	// CCPAModuleEnabled gates CCPA/CPRA Do Not Sell opt-out, GPC processing, and privacy rights handler (plan 10.4).
	CCPAModuleEnabled bool
	// DPAPortalEnabled gates the SDPC/NDPA DPA portal: district acceptance, data inventory, and SDPC CSV export (plan 10.5).
	DPAPortalEnabled bool
	// StatePrivacyEnabled gates CA SOPIPA, NY Ed Law 2-d, and IL SOPPA state-specific student data privacy controls (plan 10.6).
	StatePrivacyEnabled bool
	// SOC2ModuleEnabled gates the SOC 2 Type II compliance admin UI: access reviews, incident log, vendor risk register (plan 10.9).
	SOC2ModuleEnabled bool
	// IsoIsmsEnabled gates ISO 27001/27701 ISMS admin APIs: audit findings, risk register, SoA (plan 10.10).
	IsoIsmsEnabled bool
	// AdminAuditLogEnabled gates the admin audit log viewer and export API (plan 10.11). Defaults to true.
	AdminAuditLogEnabled bool
	// AdminConsoleEnabled gates the org admin console UI and /api/v1/admin-console/* APIs (plan 18.1).
	AdminConsoleEnabled bool
	// ImpersonationEnabled gates admin "view as student" impersonation (plan 18.3).
	ImpersonationEnabled bool
	// BulkCsvImportEnabled gates org-admin bulk user CSV import (plan 18.2).
	BulkCsvImportEnabled bool
	// AdminSearchEnabled gates org-wide admin search UI and /api/v1/admin/search APIs (plan 18.4).
	AdminSearchEnabled bool
	// EmailTemplateEditorEnabled gates org admin email template editor APIs and UI (plan 18.5).
	EmailTemplateEditorEnabled bool
	// MaintenanceBannerEnabled gates site-wide / org maintenance banners and banner admin APIs (plan 18.6).
	MaintenanceBannerEnabled bool
	// CustomFieldsEnabled gates org-admin custom field schemas and value APIs (plan 18.7).
	CustomFieldsEnabled bool
	// SeatManagementEnabled gates org seat license enforcement and license admin APIs (plan 18.8).
	SeatManagementEnabled bool
	// DataResidencyEnabled gates per-tenant region pinning enforcement and the data residency compliance admin API (plan 10.12).
	DataResidencyEnabled bool
	// AiDisclosureEnabled gates AI opt-out, gateway enforcement, inference logging, and disclosure APIs (plan 10.17). Defaults to true.
	AiDisclosureEnabled bool
	// AiProviderAbstractionEnabled gates per-tenant AI provider selection and BYOK (plan 16.7 / AP.9).
	// Defaults to true (GA). Set AI_PROVIDER_ABSTRACTION_ENABLED=0 to roll back to legacy OpenRouter-only admin paths.
	AiProviderAbstractionEnabled bool
	// SecurityDisclosureModuleEnabled gates responsible-disclosure report triage APIs (plan 10.16).
	SecurityDisclosureModuleEnabled bool
	// BackupModuleEnabled gates backup/restore ops: backup status and restore drill APIs (plan 10.15).
	BackupModuleEnabled bool
	// RTLEnabled gates mirrored RTL layout for RTL locales (plan 11.2). Defaults to false until audit complete.
	RTLEnabled bool
	// FFReadingPreferences gates the reading preferences panel UI (plan 12.6). Default false; flip after QA sign-off.
	FFReadingPreferences bool
	// FFParentPortal enables the K-12 parent/guardian portal: parent-student linking, read-only grade access, and notification prefs (plan 13.1).
	FFParentPortal bool
	// FFParentPortalV2 enables expanded parent portal sections (attendance, behavior, report cards, message teacher) — plan W02.
	FFParentPortalV2 bool
	// FFReportCards enables district-formatted report cards with comment banks and PDF generation (plan 13.4).
	FFReportCards bool
	// FFSISIntegration enables SIS vendor connections (PowerSchool, Infinite Campus, Skyward, Aeries) and nightly sync (plan 13.7).
	FFSISIntegration bool
	// FFCatalogIntegration enables HE course catalog browse, registration status, and SIS catalog sync (plan 14.2).
	FFCatalogIntegration bool
	// FFEnrollmentStateMachine enables HE enrollment lifecycle states (W/AU/I/NC) and state history (plan 14.3).
	FFEnrollmentStateMachine bool
	// FFIncompleteGradeWorkflow enables Incomplete grade grant/resolve, reminders, and registrar report (plan 14.4).
	FFIncompleteGradeWorkflow bool
	// FFLibrary enables the school library catalog, student reading log, and reading dashboard (plan 13.8).
	FFLibrary bool
	// FFBroadcasts enables district/school broadcast messages and emergency acknowledgement (plan 13.10).
	FFBroadcasts bool
	// FFClassroomSignals enables the digital hall pass and anonymous question queue (plan 13.9).
	FFClassroomSignals bool
	// FFConferenceScheduling enables parent-teacher conference scheduling (plan 13.12).
	FFConferenceScheduling bool
	// FFDemographics enables student demographic flags and Title I reporting (plan 13.13).
	FFDemographics bool
	// FFContentFilterIntegration enables GoGuardian/Securly content-filter hooks and allowlist (plan 13.14).
	FFContentFilterIntegration bool
	// FFUiMode enables age-appropriate UI modes: K-2, elementary, and secondary (plan 13.11).
	FFUiMode bool
	// FFGradeSubmission enables the final grade roll-up, review-and-confirm step, and SIS export (plan 14.5).
	FFGradeSubmission bool
	// FFWhatifGrades enables student what-if grade projection on My Grades (plan 3.16).
	FFWhatifGrades bool
	// FFGradeCurving enables instructor grade curving/scaling on assignments (plan 3.17).
	FFGradeCurving bool
	// FFAcademicCalendar enables institutional academic calendar events, dashboard upcoming-dates panel, and iCal feed (plan 14.6).
	FFAcademicCalendar bool
	// FFPlagiarismChecks enables HE plagiarism workflow APIs and async scans (plan 14.8).
	FFPlagiarismChecks bool
	// FFCourseEvaluations enables anonymous end-of-term course evaluations (plan 14.7).
	FFCourseEvaluations bool
	// FFProctoringIntegration enables LTI 1.3 proctoring vendor integrations (plan 14.9).
	FFProctoringIntegration bool
	// FFCoCurricularTranscript enables CLR generation, download, and public verification (plan 14.13).
	FFCoCurricularTranscript bool
	// FFLibraryIntegration enables HE library / e-reserves integration: Leganto LTI, Alma search, EZproxy rewriting (plan 14.10).
	FFLibraryIntegration bool
	// FFBookstoreIntegration enables bookstore / textbook integration: VitalSource & RedShelf Inclusive Access LTI deep links (plan 14.11).
	FFBookstoreIntegration bool
	// FFEportfolio enables the ePortfolio / capstone artifact collection module (plan 14.12).
	// Managed in Settings → Global platform (not process env).
	FFEportfolio bool
	// FFTranscripts enables student transcript requests and institution webhook configuration.
	// Managed in Settings → Global platform (not process env).
	FFTranscripts bool
	// FFTranscriptInbound enables inbound transcript receiving and the intake queue (T07).
	// Managed in Settings → Global platform (not process env).
	FFTranscriptInbound bool
	// FFDiplomas enables diploma/certificate templates, issuance, wallet, and verification (T11).
	// Managed in Settings → Global platform (not process env).
	FFDiplomas bool
	// FFWebhooks enables outbound webhook subscriptions and delivery (plan 16.3).
	// Managed in Settings → Global platform (not process env).
	FFWebhooks bool
	// FFZapierConnector enables Zapier/Make REST-hook webhook subscriptions (plan 16.10).
	// Managed in Settings → Global platform (not process env).
	FFZapierConnector bool
	// FFAdvisingIntegration enables advising appointment links, degree progress, and advisor notes (plan 14.14).
	// Managed in Settings → Global platform (not process env).
	FFAdvisingIntegration bool
	// FFResearchConsent enables research / IRB consent studies, consent prompts, and gated data export (plan 14.15).
	// Managed in Settings → Global platform (not process env).
	FFResearchConsent bool
	// FFAccessibilityIntake enables the accessibility services intake workflow: coordinator
	// accommodation profiles propagated to assessment overrides (plan 14.16).
	// Managed in Settings → Global platform (not process env).
	FFAccessibilityIntake bool
	// FFCEUTracking enables CEU seat-time tracking, certificates, and CE transcripts (plan 14.17).
	// Managed in Settings → Global platform (not process env).
	FFCEUTracking bool
	// FFConsortiumSharing enables multi-campus consortium course sharing and cross-institutional enrollment (plan 14.18).
	// Managed in Settings → Global platform (not process env).
	FFConsortiumSharing bool
	// FFSelfPacedMode enables self-paced enrollment with no instructor for homeschool courses (plan 15.2).
	// Managed in Settings → Global platform (not process env).
	FFSelfPacedMode bool
	// FFPublicCatalog enables the public, unauthenticated course catalog and search (plan 15.1).
	// Managed in Settings → Global platform (not process env).
	FFPublicCatalog bool
	// FFCourseMarketplace enables the in-app course marketplace/storefront (plan MKT1).
	// Distinct from FFMarketplace (plugin/OAuth app marketplace, plan 16.9). Default ON.
	// Managed in Settings → Global platform (not process env).
	FFCourseMarketplace bool
	// FFFeedback is always on (platform master removed). In-app product feedback (plan FB0).
	// Retained for API compatibility with platform settings payloads.
	FFFeedback bool
	// FFVisualBoards is deprecated (always on). Collaboration boards are gated only by the
	// per-course visual_boards_enabled flag. Retained for API compatibility with platform settings payloads.
	FFVisualBoards bool
	// FFBoardsRealtime enables the board Y.js WebSocket relay (plan VC.4). Default ON.
	// Requires per-course visual_boards_enabled.
	FFBoardsRealtime bool
	// FFBoardsExternalSharing allows link/public board visibility and share links (plan VC.6). Default OFF.
	FFBoardsExternalSharing bool
	// FFInteractiveQuizzes is always treated as on; Live Quizzes are gated only by per-course
	// interactive_quizzes_enabled (platform master switch removed). Kept for API compatibility.
	FFInteractiveQuizzes bool
	// ScreenShareEnabled is the platform master switch for cableless screen sharing (SS.1). Default OFF.
	// Both this and the per-course screen_share_enabled flag must be on.
	ScreenShareEnabled bool
	// TURNSharedSecret is the coturn REST shared secret for ephemeral ICE credentials (SS.1 FR-11).
	// Env: TURN_SHARED_SECRET. Session create is refused when empty (turn-not-ready).
	TURNSharedSecret string
	// TURNURLs are ICE server URLs (stun/turn/turns), comma-separated via TURN_URLS.
	// Example: stun:localhost:3478,turn:localhost:3478?transport=udp,turn:localhost:443?transport=tcp
	TURNURLs []string
	// FFIqLiveHosting enables the live game hosting engine / WebSocket hub (plan IQ.3). Default ON.
	// Requires per-course interactive_quizzes_enabled. Can still be turned off in platform settings.
	FFIqLiveHosting bool
	// FFIqTeamMode enables team game mode (plan IQ.6). Default OFF.
	FFIqTeamMode bool
	// FFIqStudentPaced enables student-paced game mode (plan IQ.6). Default OFF.
	FFIqStudentPaced bool
	// FFIqHomework enables async homework assignments (plan IQ.6). Default OFF.
	FFIqHomework bool
	// FFIqGradebookPush enables pushing live-quiz scores into the course gradebook (plan IQ.7). Default OFF.
	FFIqGradebookPush bool
	// FFIqPublicKitCatalog enables the curated public live-quiz kit catalog (plan IQ.8). Default OFF.
	// Org sharing and templates work without this flag; public listing requires moderation (pending → listed).
	FFIqPublicKitCatalog bool
	// FFIqGuestJoin enables guest (unauthenticated) join for live quizzes when a game allows it (plan IQ.9).
	// Default OFF. Blocked for under-13/COPPA courses even when on.
	FFIqGuestJoin bool
	// FFIqAiGeneration enables AI-assisted quiz kit generation in the kit editor (plan IQ.10). Default OFF.
	// Requires per-course interactive_quizzes_enabled and configured AI providers.
	FFIqAiGeneration bool
	// FFPublicAPI enables the versioned public REST API for third-party integrations (plan 16.1).
	// Managed in Settings → Global platform (not process env).
	FFPublicAPI bool
	// EnableAPIDocs serves Swagger UI and ReDoc at /api/v1/docs and /api/v1/redoc (plan 16.1).
	EnableAPIDocs bool
	// FFLearningPaths enables learning paths / course bundles for homeschool (plan 15.4).
	// Managed in Settings → Global platform (not process env).
	FFLearningPaths bool
	// FFConditionalRelease enables rule-based module requirements and conditional release (plan 1.11).
	// Managed in Settings → Global platform (not process env).
	FFConditionalRelease bool
	// FFPeerReview enables peer review configuration, allocation, and student review workspace (plan 3.15).
	// Managed in Settings → Global platform (not process env).
	FFPeerReview bool
	// FFCompletionCredentials enables course completion certificates, Open Badges export, and LinkedIn share (plans 15.5, 15.6).
	// Managed in Settings → Global platform (not process env).
	FFCompletionCredentials bool
	// FFCourseReviews enables learner star ratings and text reviews on catalog and course pages (plan 15.7).
	// Managed in Settings → Global platform (not process env).
	FFCourseReviews bool
	// FFGamification enables streaks, XP, leaderboards, and badges for homeschool courses (plan 15.9).
	// Managed in Settings → Global platform (not process env).
	FFGamification bool
	// FFCompetencyBadges enables outcome micro-badges, public backpack, and verify (plan B1).
	// Managed in Settings → Global platform (not process env).
	FFCompetencyBadges bool
	// BadgesDefaultPublic is the tenant default for new award is_public (plan B1).
	// Managed in Settings → Global platform (not process env).
	BadgesDefaultPublic bool
	// FFOnboardingFlow enables the homeschool onboarding wizard with goal capture and diagnostic placement (plan 15.11).
	// Managed in Settings → Global platform (not process env).
	FFOnboardingFlow bool
	// FFStudyReminders enables daily study goal reminders and weekly progress summaries (plan 15.10).
	// Managed in Settings → Global platform (not process env).
	FFStudyReminders bool
	// FFAIStudyBuddy enables the homeschool AI study buddy with persistent memory (plan 15.12).
	// Managed in Settings → Global platform (not process env).
	FFAIStudyBuddy bool
	// FFLessonGenerator enables the AI lesson generator wizard for instructors (plan 19.2).
	// Managed in Settings → Global platform (not process env).
	FFLessonGenerator bool
	// FFPersistentTutor enables named tutor sessions with RAG citations (plan 19.1).
	// Managed in Settings → Global platform (not process env).
	FFPersistentTutor bool
	// FFAPITokens enables personal and institutional API access keys (plan 16.2).
	// Managed in Settings → Global platform (not process env).
	FFAPITokens bool
	// FFBotSlack enables the Lextures Slack classroom bot (plan 16.6).
	// Managed in Settings → Global platform (not process env).
	FFBotSlack bool
	// FFBotTeams enables the Lextures Microsoft Teams bot (plan 16.6).
	// Managed in Settings → Global platform (not process env).
	FFBotTeams bool
	// FFBotDiscord enables the Lextures Discord bot (plan 16.6).
	// Managed in Settings → Global platform (not process env).
	FFBotDiscord bool
	// FFCalendarFeeds enables iCal/CalDAV calendar feed subscriptions (plan 16.5).
	// Managed in Settings → Global platform (not process env).
	FFCalendarFeeds bool
	// FFRedisCache enables Redis-backed object caching for hot read paths (plan 17.5).
	// Managed in Settings → Global platform (not process env).
	FFRedisCache bool

	// SlackBotClientID is the Slack app client id for bot OAuth (plan 16.6).
	SlackBotClientID string
	// SlackBotClientSecret is the Slack app client secret for bot OAuth (plan 16.6).
	SlackBotClientSecret string
	// DiscordBotClientID is the Discord application id (plan 16.6).
	DiscordBotClientID string
	// DiscordBotPublicKey is the Discord interaction verification public key hex (plan 16.6).
	DiscordBotPublicKey string
	// TeamsBotAppID is the Microsoft Bot Framework app id (plan 16.6).
	TeamsBotAppID string
	// TeamsBotAppPassword is the Microsoft Bot Framework app password (plan 16.6).
	TeamsBotAppPassword string

	// FFStripeBilling enables Stripe checkout, subscriptions, and entitlement gating (plan 15.3).
	// Managed in Settings → Global platform (not process env).
	FFStripeBilling bool
	// FFPaymentsEnabled enables multi-provider payment abstraction (Stripe + PayPal) (plan 16.8).
	// Managed in Settings → Global platform (not process env).
	FFPaymentsEnabled bool
	// FFRevenueShare enables creator revenue share, affiliate tracking, and Stripe Connect payouts (plan 15.8).
	// Managed in Settings → Global platform (not process env).
	FFRevenueShare bool
	// FFTaxCollection enables Stripe Tax calculation, collection, and reporting (plan 15.13).
	// Managed in Settings → Global platform (not process env).
	FFTaxCollection bool
	// FFMarketplace enables the marketplace / plugin system with OAuth 2.1 app authorization (plan 16.9).
	// Managed in Settings → Global platform (not process env).
	FFMarketplace bool

	// StripeSecretKey is the Stripe API secret key (sk_live_… or sk_test_…).
	StripeSecretKey string
	// StripeWebhookSecret verifies Stripe webhook signatures (whsec_…).
	StripeWebhookSecret string
	// StripeMonthlyPriceID is the Stripe Price id for monthly platform subscription.
	StripeMonthlyPriceID string
	// StripeAnnualPriceID is the Stripe Price id for annual platform subscription.
	StripeAnnualPriceID string

	// PayPalClientID is the PayPal REST app client id (plan 16.8).
	PayPalClientID string
	// PayPalClientSecret is the PayPal REST app secret (plan 16.8).
	PayPalClientSecret string
	// PayPalWebhookID verifies inbound PayPal webhooks (plan 16.8).
	PayPalWebhookID string
	// PayPalSandbox selects PayPal sandbox API hosts when true.
	PayPalSandbox bool

	// Adaptive-learning platform gates (managed in Settings → Global platform; combined with
	// the per-course flag at the callsite). Previously env-only service flags.
	// DiagnosticAssessmentsEnabled is the platform gate for adaptive diagnostic assessments.
	DiagnosticAssessmentsEnabled bool
	// SRSPracticeEnabled is the platform gate for spaced-repetition practice.
	SRSPracticeEnabled bool
	// IRTCatModeEnabled enables IRT computerized adaptive testing (CAT) item selection.
	IRTCatModeEnabled bool
	// AdaptiveLearnerModelEnabled enables the adaptive learner (θ) model updates.
	AdaptiveLearnerModelEnabled bool
	// LearnerModelEMAAlpha is the EMA smoothing factor for the learner model, in (0,1]. Default 0.3.
	LearnerModelEMAAlpha float64

	// CCRSigningSeedB64 is a base64-encoded 32-byte Ed25519 seed for CLR signing (plan 14.13).
	CCRSigningSeedB64 string
	// CCRInstitutionName is the issuer name on generated CLRs (plan 14.13).
	CCRInstitutionName string

	// AppEnv is the deployment environment (local, staging, production). Used for PII redaction guards (plan 10.14).
	AppEnv string
	// DisablePIIRedaction allows plaintext PII in operational logs for local debugging only (plan 10.14).
	DisablePIIRedaction bool
	// PIIRedactFields adds extra structured log field names to the redaction registry (REDACT_FIELDS).
	PIIRedactFields []string

	// RedisURL is the shared Redis connection string for cross-instance state
	// (JWT blocklist, rate limits, caches) — plan 17.2. Use rediss:// for the
	// TLS-only managed Redis in production. Empty disables Redis (single instance).
	RedisURL string
	// RedisPoolMin is the minimum idle Redis connections per instance (default 5).
	RedisPoolMin int
	// RedisPoolMax is the maximum Redis connections per instance (default 20).
	RedisPoolMax int

	// RateLimits configures Redis-backed request rate limiting (plan 17.6).
	RateLimits RateLimits

	// DBPoolMaxConns caps pgx pool connections per instance for multi-instance
	// deployments (instances × DBPoolMaxConns must stay under Postgres
	// max_connections) — plan 17.2 FR-7. 0 keeps the pgx default.
	DBPoolMaxConns int
	// DBPoolMinConns is the minimum warm pgx pool connections per instance. 0 keeps the pgx default.
	DBPoolMinConns int

	// ShutdownTimeoutSecs is the graceful-shutdown drain window on SIGTERM
	// (plan 17.2 FR-8 / AC-4). Defaults to 30 seconds.
	ShutdownTimeoutSecs int

	// QueueBackend selects the durable message bus: "rabbitmq" (default when
	// RABBITMQ_URL is set), "sqs" (AWS SQS), or "memory" (in-process only).
	// Empty auto-detects: SQS when QUEUE_BACKEND=sqs or any SQS_*_URL is set and
	// QUEUE_BACKEND is not rabbitmq; otherwise RabbitMQ when RABBITMQ_URL is set.
	QueueBackend string
	// RabbitMQURL is the AMQP connection URL for background job queues (Canvas import, etc.).
	RabbitMQURL string
	// SQS queue URLs (full https://sqs.<region>.amazonaws.com/... URLs). Used when QueueBackend is "sqs".
	SQSCanvasImportURL         string
	SQSCanvasSubmissionSyncURL string
	SQSSmsNotificationURL      string
	SQSGradingAgentURL         string
	// CanvasImportQueueName is the RabbitMQ queue for Canvas LMS imports (default canvas.course.import).
	CanvasImportQueueName string
	// CanvasImportConcurrency is how many Canvas import jobs the queue consumer processes in parallel.
	CanvasImportConcurrency int
	// CanvasSubmissionSyncQueueName is the RabbitMQ queue for Canvas grade pushes (default canvas.submission.sync).
	CanvasSubmissionSyncQueueName string
	// CanvasSubmissionSyncConcurrency is how many Canvas grade-push jobs the queue consumer processes in parallel.
	CanvasSubmissionSyncConcurrency int
	// SmsNotificationQueueName is the RabbitMQ queue for SMS notifications (default notifications.sms).
	SmsNotificationQueueName string
	// SmsNotificationConcurrency is how many SMS jobs the queue consumer processes in parallel.
	SmsNotificationConcurrency int

	// TwilioAccountSID is the Twilio account SID for SMS delivery.
	TwilioAccountSID string
	// TwilioAuthToken is the Twilio auth token for SMS delivery.
	TwilioAuthToken string
	// TwilioFromNumber is the Twilio sender phone number (E.164).
	TwilioFromNumber string

	// StatusPageEnabled gates the public status summary proxy and Alertmanager webhook (plan 17.13).
	StatusPageEnabled bool
	// StatusPageURL is the public status page URL shown in the incident banner.
	StatusPageURL string
	// StatuspageAPIKey is the Statuspage.io API key (stored in secrets manager in production).
	StatuspageAPIKey string
	// StatuspagePageID is the Statuspage.io page identifier.
	StatuspagePageID string
	// StatuspageComponentMapJSON maps logical component keys to Statuspage component IDs.
	StatuspageComponentMapJSON string
	// AlertmanagerWebhookSecret authenticates Alertmanager webhook posts.
	AlertmanagerWebhookSecret string
	// StatuspageWebhookSecret verifies HMAC signatures on Statuspage incident webhooks (plan 18.6).
	StatuspageWebhookSecret string
	// StatusPageSummaryCacheSecs is the in-memory cache TTL for the summary proxy.
	StatusPageSummaryCacheSecs int

	// Observability configures Prometheus metrics, OpenTelemetry traces, and
	// Sentry error reporting (plan 17.7). All fields are optional; an empty
	// MetricsAddr disables the metrics endpoint, an empty OTel endpoint disables
	// tracing, and an empty Sentry DSN disables error reporting.
	Observability Observability
}

// Observability holds Prometheus / OpenTelemetry / Sentry settings (plan 17.7).
type Observability struct {
	// ServiceName labels metrics and traces (default "lextures-api").
	ServiceName string
	// Version is the running build/release identifier (build_info, Sentry release).
	Version string

	// MetricsEnabled gates the internal /metrics server. Default true.
	MetricsEnabled bool
	// MetricsAddr is the listen address for the internal metrics server. It MUST
	// be a separate, VPC-internal port — never the public LB port (FR-1, AC-6).
	MetricsAddr string

	// OTelEndpoint is the OTLP/HTTP collector endpoint (host:port). Empty disables tracing.
	OTelEndpoint string
	// OTelInsecure sends plaintext OTLP to an in-VPC collector. Default true.
	OTelInsecure bool
	// OTelSampleRatio is the head-based trace sample rate (0..1). Default 0.1 (FR-3).
	OTelSampleRatio float64

	// SentryDSN is the project DSN (separate per environment — FR-4). Empty disables Sentry.
	SentryDSN string
	// SentryTracesSampleRate samples performance transactions (default 0.1 — FR-3).
	SentryTracesSampleRate float64

	// DeployColor labels Prometheus metrics for blue/green and canary analysis
	// (plan 17.9). Set via DEPLOY_COLOR (e.g. blue, green, stable).
	DeployColor string
}

// Load reads configuration from the environment.
func Load() Config {
	ltiBaseURL := firstNonEmptyTrimmed("LTI_API_BASE_URL")
	if ltiBaseURL == "" {
		ltiBaseURL = "http://localhost:8080"
	}
	ltiBaseURL = trimTrailingSlash(ltiBaseURL)

	samlBaseURL := firstNonEmptyTrimmed("SAML_PUBLIC_BASE_URL", "LTI_API_BASE_URL")
	if samlBaseURL == "" {
		samlBaseURL = "http://localhost:8080"
	}
	samlBaseURL = trimTrailingSlash(samlBaseURL)

	oidcBaseURL := firstNonEmptyTrimmed("OIDC_PUBLIC_BASE_URL", "LTI_API_BASE_URL")
	if oidcBaseURL == "" {
		oidcBaseURL = "http://localhost:8080"
	}
	oidcBaseURL = trimTrailingSlash(oidcBaseURL)

	allowInsecureJWT := boolEnv("ALLOW_INSECURE_JWT")
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" && allowInsecureJWT {
		jwtSecret = insecureJWTFallback
	}

	env := appEnv()
	localDev := env == "local"

	return Config{
		HTTPAddr: httpAddr(),

		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:        jwtSecret,
		AllowInsecureJWT: allowInsecureJWT,
		RunMigrations:    runMigrations(),

		BootstrapAdminEmail: strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),

		// OpenRouterAPIKey is loaded from settings.platform_app_settings (Settings → Intelligence → Models).
		OpenRouterAPIKey: "",
		CourseFilesRoot:  stringDefault(firstNonEmptyTrimmed("COURSE_FILES_ROOT"), "data/course-files"),

		CanvasAllowedHostSuffixes:     canvasAllowedHostSuffixes(),
		PublicWebOrigin:               trimTrailingSlash(stringDefault(firstNonEmptyTrimmed("PUBLIC_WEB_ORIGIN"), "http://localhost:5173")),
		BrandingMultitenantHostSuffix: strings.TrimSpace(strings.ToLower(firstNonEmptyTrimmed("BRANDING_MULTITENANT_HOST_SUFFIX"))),

		PlatformSecretsKey: platformSecretsKeyFromEnv(),

		SMTPHost:     firstNonEmptyTrimmed("SMTP_HOST"),
		SMTPPort:     smtpPort(),
		SMTPUser:     firstNonEmptyTrimmed("SMTP_USER"),
		SMTPPassword: firstNonEmptyTrimmed("SMTP_PASSWORD"),
		SMTPFrom:     firstNonEmptyTrimmed("SMTP_FROM"),

		EmailProvider: strings.ToLower(firstNonEmptyTrimmed("EMAIL_PROVIDER")),
		// Region: SES_REGION, else AWS_REGION. Credentials: optional SES_ACCESS_KEY_ID /
		// SES_SECRET_ACCESS_KEY; when empty the default AWS chain is used (env, IAM role, …).
		SESRegion:           firstNonEmptyTrimmed("SES_REGION", "AWS_REGION"),
		SESFrom:             firstNonEmptyTrimmed("SES_FROM"),
		SESConfigurationSet: firstNonEmptyTrimmed("SES_CONFIGURATION_SET"),
		SESAccessKeyID:      firstNonEmptyTrimmed("SES_ACCESS_KEY_ID"),
		SESSecretAccessKey:  firstNonEmptyTrimmed("SES_SECRET_ACCESS_KEY"),

		LTIAPIBaseURL:       ltiBaseURL,
		LTIRSAPrivateKeyPEM: firstNonEmptyTrimmed("LTI_RSA_PRIVATE_KEY_PEM"),
		LTIRSAKeyID:         stringDefault(firstNonEmptyTrimmed("LTI_RSA_KEY_ID"), "lti-key-1"),

		SAMLSSOEnabled:      false,
		SAMLPublicBaseURL:   samlBaseURL,
		SAMLSPEntityID:      stringDefault(firstNonEmptyTrimmed("SAML_SP_ENTITY_ID"), samlBaseURL+"/auth/saml/metadata"),
		SAMLSPX509PEM:       firstNonEmptyTrimmedOrFile("SAML_SP_X509_PEM", "SAML_SP_X509_PATH"),
		SAMLSPPrivateKeyPEM: firstNonEmptyTrimmedOrFile("SAML_SP_PRIVATE_KEY_PEM", "SAML_SP_PRIVATE_KEY_PATH"),

		OIDCSSOEnabled:            false,
		OIDCPublicBaseURL:         oidcBaseURL,
		OIDCGoogleClientID:        firstNonEmptyTrimmed("OIDC_GOOGLE_CLIENT_ID"),
		OIDCGoogleClientSecret:    firstNonEmptyTrimmed("OIDC_GOOGLE_CLIENT_SECRET"),
		OIDCGoogleHostedDomain:    firstNonEmptyTrimmed("OIDC_GOOGLE_HOSTED_DOMAIN", "OIDC_GOOGLE_HD"),
		OIDCMicrosoftTenant:       stringDefault(firstNonEmptyTrimmed("OIDC_MICROSOFT_TENANT"), "common"),
		OIDCMicrosoftClientID:     firstNonEmptyTrimmed("OIDC_MICROSOFT_CLIENT_ID"),
		OIDCMicrosoftClientSecret: firstNonEmptyTrimmed("OIDC_MICROSOFT_CLIENT_SECRET"),
		OIDCAppleClientID:         firstNonEmptyTrimmed("OIDC_APPLE_CLIENT_ID"),
		OIDCAppleTeamID:           firstNonEmptyTrimmed("OIDC_APPLE_TEAM_ID"),
		OIDCAppleKeyID:            firstNonEmptyTrimmed("OIDC_APPLE_KEY_ID"),
		OIDCApplePrivateKeyPEM:    firstNonEmptyTrimmedOrFile("OIDC_APPLE_PRIVATE_KEY_PEM", "OIDC_APPLE_PRIVATE_KEY_PATH"),
		OIDCAppleNativeAudience: stringDefault(
			firstNonEmptyTrimmed("OIDC_APPLE_NATIVE_AUDIENCE"),
			"com.lextures.ios",
		),
		OIDCGoogleNativeAudience: firstNonEmptyTrimmed("OIDC_GOOGLE_NATIVE_AUDIENCE"),

		CleverSSOEnabled:   false,
		CleverClientID:     firstNonEmptyTrimmed("CLEVER_CLIENT_ID", "CLEVER_OIDC_CLIENT_ID"),
		CleverClientSecret: firstNonEmptyTrimmed("CLEVER_CLIENT_SECRET", "CLEVER_OIDC_CLIENT_SECRET"),
		CleverDistrictID:   firstNonEmptyTrimmed("CLEVER_DISTRICT_ID"),

		ClassLinkSSOEnabled:       false,
		ClassLinkOIDCIssuer:       strings.TrimRight(firstNonEmptyTrimmed("CLASSLINK_OIDC_ISSUER"), "/"),
		ClassLinkOIDCClientID:     firstNonEmptyTrimmed("CLASSLINK_OIDC_CLIENT_ID"),
		ClassLinkOIDCClientSecret: firstNonEmptyTrimmed("CLASSLINK_OIDC_CLIENT_SECRET"),

		OneRosterEnabled:             false,
		OneRosterBearerFallbackToken: firstNonEmptyTrimmed("ONEROSTER_BEARER_FALLBACK_TOKEN"),
		OneRosterBearerFallbackInst:  strings.TrimSpace(os.Getenv("ONEROSTER_BEARER_FALLBACK_INSTITUTION_ID")),

		ScimEnabled: false,

		MFAEnforcement: "none",

		PushNotificationsEnabled: false,
		VAPIDPublicKey:           firstNonEmptyTrimmed("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:          firstNonEmptyTrimmed("VAPID_PRIVATE_KEY"),
		VAPIDSubject:             stringDefault(firstNonEmptyTrimmed("VAPID_SUBJECT"), "mailto:admin@lextures.com"),
		APNSP8Key:                firstNonEmptyTrimmed("APNS_P8_KEY"),
		APNSKeyID:                firstNonEmptyTrimmed("APNS_KEY_ID"),
		APNSTeamID:               firstNonEmptyTrimmed("APNS_TEAM_ID"),
		APNSBundleID:             firstNonEmptyTrimmed("APNS_BUNDLE_ID"),
		APNSEnvironment:          stringDefault(firstNonEmptyTrimmed("APNS_ENVIRONMENT"), "production"),
		FCMServiceAccountJSON:    firstNonEmptyTrimmed("FCM_SERVICE_ACCOUNT_JSON"),

		VirtualClassroomEnabled: true,
		JitsiBaseURL:            stringDefault(firstNonEmptyTrimmed("JITSI_BASE_URL"), "https://meet.jit.si"),
		JitsiAppID:              firstNonEmptyTrimmed("JITSI_APP_ID"),
		JitsiAppSecret:          firstNonEmptyTrimmed("JITSI_APP_SECRET"),
		BBBBaseURL:              firstNonEmptyTrimmed("BBB_BASE_URL"),
		BBBSecret:               firstNonEmptyTrimmed("BBB_SECRET"),

		StorageBackend:         firstNonEmptyTrimmed("STORAGE_BACKEND"),
		StorageBucket:          firstNonEmptyTrimmed("STORAGE_BUCKET", "AWS_BUCKET"),
		StorageEndpoint:        firstNonEmptyTrimmed("STORAGE_ENDPOINT"),
		StorageAccessKeyID:     firstNonEmptyTrimmed("STORAGE_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID"),
		StorageSecretAccessKey: firstNonEmptyTrimmed("STORAGE_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY"),
		StorageRegion:          firstNonEmptyTrimmed("STORAGE_REGION", "AWS_REGION"),
		StorageCDNBaseURL:      strings.TrimSpace(os.Getenv("STORAGE_CDN_BASE_URL")),
		StorageUseSSL:          storageUseSSL(),
		StoragePresignTTL:      storagePresignTTL(),
		StorageMigrateLocal:    boolEnv("STORAGE_MIGRATE_LOCAL"),

		TusUploadTTLHours: tusUploadTTLHours(),

		DRMHMACSecret: firstNonEmptyTrimmed("DRM_HMAC_SECRET"),

		TranscodeRetainSourceDays: transcodeRetainSourceDays(),
		FFmpegPath:                firstNonEmptyTrimmed("FFMPEG_PATH"),

		WhisperBackend: stringDefault(firstNonEmptyTrimmed("WHISPER_BACKEND"), "whisper-api"),
		OpenAIAPIKey:   firstNonEmptyTrimmed("OPENAI_API_KEY"),

		StorageDefaultTenantQuotaGB: storageDefaultTenantQuotaGB(),

		ClamAVAddr: stringDefault(firstNonEmptyTrimmed("CLAMAV_ADDR"), "localhost:3310"),

		// Dev/e2e test doubles — env-only (never Settings → Global platform).
		OriginalityStubExternal: boolEnv("ORIGINALITY_STUB_EXTERNAL"),
		ClamAVStub:              boolEnv("CLAMAV_STUB"),
		OERStub:                 boolEnv("OER_STUB"),

		// Feature flags below are managed in Settings → Global platform (DB-backed) and
		// resolved by platformconfig.Merge / applyPlatformBools. They are intentionally NOT
		// seeded from the process environment — see server/internal/repos/platformconfig.
		// Operational/security controls (DISABLE_PII_REDACTION, ALLOW_INSECURE_JWT, etc.)
		// remain environment-driven below.

		CCRSigningSeedB64:  strings.TrimSpace(os.Getenv("CCR_SIGNING_SEED_B64")),
		CCRInstitutionName: strings.TrimSpace(os.Getenv("CCR_INSTITUTION_NAME")),

		SlackBotClientID:     firstNonEmptyTrimmed("SLACK_BOT_CLIENT_ID"),
		SlackBotClientSecret: firstNonEmptyTrimmed("SLACK_BOT_CLIENT_SECRET"),
		DiscordBotClientID:   firstNonEmptyTrimmed("DISCORD_BOT_CLIENT_ID"),
		DiscordBotPublicKey:  firstNonEmptyTrimmed("DISCORD_BOT_PUBLIC_KEY"),
		TeamsBotAppID:        firstNonEmptyTrimmed("TEAMS_BOT_APP_ID"),
		TeamsBotAppPassword:  firstNonEmptyTrimmed("TEAMS_BOT_APP_PASSWORD"),

		TurnstileSecretKey: strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),

		StripeSecretKey:      strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		StripeWebhookSecret:  strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		StripeMonthlyPriceID: strings.TrimSpace(os.Getenv("STRIPE_MONTHLY_PRICE_ID")),
		StripeAnnualPriceID:  strings.TrimSpace(os.Getenv("STRIPE_ANNUAL_PRICE_ID")),

		PayPalClientID:     strings.TrimSpace(os.Getenv("PAYPAL_CLIENT_ID")),
		PayPalClientSecret: strings.TrimSpace(os.Getenv("PAYPAL_CLIENT_SECRET")),
		PayPalWebhookID:    strings.TrimSpace(os.Getenv("PAYPAL_WEBHOOK_ID")),
		PayPalSandbox:      boolEnv("PAYPAL_SANDBOX"),

		AppEnv:              env,
		DisablePIIRedaction: boolEnv("DISABLE_PII_REDACTION"),
		PIIRedactFields:     commaSeparatedEnv("REDACT_FIELDS"),

		RedisURL: firstNonEmptyTrimmed("REDIS_URL"),

		TURNSharedSecret:    firstNonEmptyTrimmed("TURN_SHARED_SECRET"),
		TURNURLs:            commaSeparatedEnv("TURN_URLS"),
		RedisPoolMin:        intEnvDefault("REDIS_POOL_MIN", redisclient.DefaultPoolMin),
		RedisPoolMax:        intEnvDefault("REDIS_POOL_MAX", redisclient.DefaultPoolMax),
		RateLimits:          rateLimitsFromEnv(),
		DBPoolMaxConns:      intEnvDefault("DB_POOL_MAX_CONNS", 0),
		DBPoolMinConns:      intEnvDefault("DB_POOL_MIN_CONNS", 0),
		ShutdownTimeoutSecs: intEnvDefault("SHUTDOWN_TIMEOUT_SECS", defaultShutdownTimeoutSecs),

		QueueBackend:                    strings.ToLower(firstNonEmptyTrimmed("QUEUE_BACKEND")),
		RabbitMQURL:                     firstNonEmptyTrimmed("RABBITMQ_URL"),
		SQSCanvasImportURL:              firstNonEmptyTrimmed("SQS_CANVAS_IMPORT_URL"),
		SQSCanvasSubmissionSyncURL:      firstNonEmptyTrimmed("SQS_CANVAS_SUBMISSION_SYNC_URL"),
		SQSSmsNotificationURL:           firstNonEmptyTrimmed("SQS_SMS_NOTIFICATION_URL"),
		SQSGradingAgentURL:              firstNonEmptyTrimmed("SQS_GRADING_AGENT_URL"),
		CanvasImportQueueName:           stringDefault(firstNonEmptyTrimmed("CANVAS_IMPORT_QUEUE_NAME"), "canvas.course.import"),
		CanvasImportConcurrency:         canvasImportConcurrency(),
		CanvasSubmissionSyncQueueName:   stringDefault(firstNonEmptyTrimmed("CANVAS_SUBMISSION_SYNC_QUEUE_NAME"), "canvas.submission.sync"),
		CanvasSubmissionSyncConcurrency: canvasSubmissionSyncConcurrency(),
		BackgroundJobsEnabled:           boolEnvDefault("BACKGROUND_JOBS_ENABLED", localDev),
		BackgroundJobsConcurrency:       intEnvDefault("BACKGROUND_JOBS_CONCURRENCY", 4),
		SchedulerEnabled:                boolEnvDefault("SCHEDULER_ENABLED", localDev),
		SmsNotificationsEnabled:         boolEnv("SMS_NOTIFICATIONS_ENABLED"),
		SmsNotificationQueueName:        stringDefault(firstNonEmptyTrimmed("SMS_NOTIFICATION_QUEUE_NAME"), "notifications.sms"),
		SmsNotificationConcurrency:      smsNotificationConcurrency(),
		TwilioAccountSID:                firstNonEmptyTrimmed("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:                 firstNonEmptyTrimmed("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber:                firstNonEmptyTrimmed("TWILIO_FROM_NUMBER"),

		EnableAPIDocs: boolEnv("ENABLE_API_DOCS"),

		AiProviderAbstractionEnabled: boolEnvDefault("AI_PROVIDER_ABSTRACTION_ENABLED", true),

		StatusPageEnabled:          boolEnv("STATUS_PAGE_ENABLED"),
		StatusPageURL:              stringDefault(firstNonEmptyTrimmed("STATUS_PAGE_URL"), "https://status.lextures.io"),
		StatuspageAPIKey:           firstNonEmptyTrimmed("STATUSPAGE_API_KEY"),
		StatuspagePageID:           firstNonEmptyTrimmed("STATUSPAGE_PAGE_ID"),
		StatuspageComponentMapJSON: statuspageComponentMapJSON(),
		AlertmanagerWebhookSecret:  firstNonEmptyTrimmed("ALERTMANAGER_WEBHOOK_SECRET"),
		StatuspageWebhookSecret:    firstNonEmptyTrimmed("STATUSPAGE_WEBHOOK_SECRET"),
		StatusPageSummaryCacheSecs: statusPageSummaryCacheSecs(),

		Observability: observabilityFromEnv(),
	}
}

// observabilityFromEnv reads the plan 17.7 observability settings. Metrics
// default ON (internal :9090) since the endpoint is harmless when unscraped;
// tracing and Sentry default OFF until an endpoint/DSN is supplied.
func observabilityFromEnv() Observability {
	return Observability{
		ServiceName:            stringDefault(firstNonEmptyTrimmed("OTEL_SERVICE_NAME", "OBSERVABILITY_SERVICE_NAME"), "lextures-api"),
		Version:                firstNonEmptyTrimmed("APP_VERSION", "GIT_SHA", "SOURCE_VERSION"),
		MetricsEnabled:         boolEnvDefault("METRICS_ENABLED", true),
		MetricsAddr:            stringDefault(firstNonEmptyTrimmed("METRICS_ADDR"), ":9090"),
		OTelEndpoint:           firstNonEmptyTrimmed("OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_ENDPOINT"),
		OTelInsecure:           boolEnvDefault("OTEL_EXPORTER_OTLP_INSECURE", true),
		OTelSampleRatio:        floatEnvDefault("OTEL_TRACES_SAMPLE_RATIO", 0.1),
		SentryDSN:              firstNonEmptyTrimmed("SENTRY_DSN"),
		SentryTracesSampleRate: floatEnvDefault("SENTRY_TRACES_SAMPLE_RATE", 0.1),
		DeployColor:            stringDefault(firstNonEmptyTrimmed("DEPLOY_COLOR"), "stable"),
	}
}

func storageDefaultTenantQuotaGB() int64 {
	s := strings.TrimSpace(os.Getenv("STORAGE_DEFAULT_TENANT_QUOTA_GB"))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// OIDCGoogleConfigured is true when Google IdP client credentials are present (Rust `OidcState.google` Some).
func (c Config) OIDCGoogleConfigured() bool {
	return strings.TrimSpace(c.OIDCGoogleClientID) != "" && strings.TrimSpace(c.OIDCGoogleClientSecret) != ""
}

// OIDCMicrosoftConfigured is true when Microsoft client credentials are present.
func (c Config) OIDCMicrosoftConfigured() bool {
	return strings.TrimSpace(c.OIDCMicrosoftClientID) != "" && strings.TrimSpace(c.OIDCMicrosoftClientSecret) != ""
}

// OIDCAppleConfigured is true when all Apple “Sign in with Apple” key material is present.
func (c Config) OIDCAppleConfigured() bool {
	return strings.TrimSpace(c.OIDCAppleClientID) != "" &&
		strings.TrimSpace(c.OIDCAppleTeamID) != "" &&
		strings.TrimSpace(c.OIDCAppleKeyID) != "" &&
		strings.TrimSpace(c.OIDCApplePrivateKeyPEM) != ""
}

// OIDCAppleNativeAudiences returns the allow-list of audiences for native Apple ID tokens.
func (c Config) OIDCAppleNativeAudiences() []string {
	raw := strings.TrimSpace(c.OIDCAppleNativeAudience)
	if raw == "" {
		raw = "com.lextures.ios"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// OIDCGoogleNativeAudienceResolved is the audience used to verify native Google ID tokens.
// Prefers OIDC_GOOGLE_NATIVE_AUDIENCE; falls back to the web OIDC Google client ID.
func (c Config) OIDCGoogleNativeAudienceResolved() string {
	if v := strings.TrimSpace(c.OIDCGoogleNativeAudience); v != "" {
		return v
	}
	return strings.TrimSpace(c.OIDCGoogleClientID)
}

// OIDCAppleNativeAvailable is true when native Sign in with Apple has at least one audience
// (default com.lextures.ios). Always on when config loads; no feature flag (MOB.9).
func (c Config) OIDCAppleNativeAvailable() bool {
	return len(c.OIDCAppleNativeAudiences()) > 0
}

// OIDCGoogleNativeAvailable is true when a Google server client ID / native audience is configured.
func (c Config) OIDCGoogleNativeAvailable() bool {
	return c.OIDCGoogleNativeAudienceResolved() != ""
}

// CleverConfigured is true when Clever OAuth client credentials are present.
func (c Config) CleverConfigured() bool {
	return strings.TrimSpace(c.CleverClientID) != "" && strings.TrimSpace(c.CleverClientSecret) != ""
}

// CleverOIDCConfigured is an alias for CleverConfigured (Clever Instant Login uses the same env vars as OAuth PKCE).
func (c Config) CleverOIDCConfigured() bool {
	return c.CleverConfigured()
}

// ClassLinkOIDCConfigured is true when ClassLink OIDC issuer and client credentials are present.
func (c Config) ClassLinkOIDCConfigured() bool {
	return strings.TrimSpace(c.ClassLinkOIDCIssuer) != "" &&
		strings.TrimSpace(c.ClassLinkOIDCClientID) != "" &&
		strings.TrimSpace(c.ClassLinkOIDCClientSecret) != ""
}

// resolvedQueueBackend returns rabbitmq | sqs | memory.
func (c Config) resolvedQueueBackend() string {
	b := strings.ToLower(strings.TrimSpace(c.QueueBackend))
	switch b {
	case "sqs", "rabbitmq", "memory", "none":
		return b
	}
	// Auto-detect: prefer SQS when any SQS URL is configured.
	if c.SQSCanvasImportURL != "" || c.SQSCanvasSubmissionSyncURL != "" ||
		c.SQSSmsNotificationURL != "" || c.SQSGradingAgentURL != "" {
		return "sqs"
	}
	if strings.TrimSpace(c.RabbitMQURL) != "" {
		return "rabbitmq"
	}
	return "memory"
}

// MessageQueueURL picks the connection/queue URL for a named bus.
// sqsURL is the full SQS queue URL when using AWS; rabbit uses RabbitMQURL + queue name elsewhere.
// Empty means in-process memory.
func (c Config) MessageQueueURL(sqsURL string) string {
	switch c.resolvedQueueBackend() {
	case "sqs":
		return strings.TrimSpace(sqsURL)
	case "memory", "none":
		return ""
	default:
		return strings.TrimSpace(c.RabbitMQURL)
	}
}

// CanvasImportQueueURL is the URL passed to canvasimportqueue.NewBus.
func (c Config) CanvasImportQueueURL() string {
	return c.MessageQueueURL(c.SQSCanvasImportURL)
}

// CanvasSubmissionSyncQueueURL is the URL passed to canvassubmissionsyncqueue.NewBus.
func (c Config) CanvasSubmissionSyncQueueURL() string {
	return c.MessageQueueURL(c.SQSCanvasSubmissionSyncURL)
}

// SmsNotificationQueueURL is the URL passed to smsnotificationqueue.NewBus.
func (c Config) SmsNotificationQueueURL() string {
	return c.MessageQueueURL(c.SQSSmsNotificationURL)
}

// GradingAgentQueueURL is the URL passed to gradingagentqueue.NewBus.
func (c Config) GradingAgentQueueURL() string {
	return c.MessageQueueURL(c.SQSGradingAgentURL)
}

// Validate returns an error if required values are missing for a full server start.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if !strings.HasPrefix(c.DatabaseURL, "postgres://") && !strings.HasPrefix(c.DatabaseURL, "postgresql://") {
		return fmt.Errorf("DATABASE_URL must be a postgres:// or postgresql:// URL")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required; set ALLOW_INSECURE_JWT=1 only for local development")
	}
	if c.JWTSecret != insecureJWTFallback && len(strings.TrimSpace(c.JWTSecret)) < JWTSecretMinLen {
		return fmt.Errorf("JWT_SECRET must be at least %d characters", JWTSecretMinLen)
	}
	if c.SAMLSSOEnabled && strings.TrimSpace(c.SAMLSPX509PEM) == "" {
		return fmt.Errorf("SAML SSO is enabled in platform settings but SAML SP X.509 certificate is missing (set SAML_SP_X509_PEM or SAML_SP_X509_PATH in environment)")
	}
	if err := c.validatePIIRedaction(); err != nil {
		return err
	}
	return nil
}

func (c Config) validatePIIRedaction() error {
	if !c.DisablePIIRedaction {
		return nil
	}
	env := strings.ToLower(strings.TrimSpace(c.AppEnv))
	switch env {
	case "production", "staging":
		return fmt.Errorf("PII redaction cannot be disabled in production")
	default:
		return nil
	}
}

func runMigrations() bool {
	v := strings.TrimSpace(os.Getenv("RUN_MIGRATIONS"))
	if v == "" {
		return true
	}
	switch strings.ToLower(v) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// intEnvDefault parses a non-negative integer env var, returning def when unset,
// empty, or invalid.
func intEnvDefault(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func boolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// boolEnvDefault parses a boolean env var, returning def when unset. Used for
// flags that default ON (e.g. METRICS_ENABLED).
func boolEnvDefault(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// floatEnvDefault parses a float env var (e.g. a sample ratio), returning def
// when unset or invalid.
func floatEnvDefault(key string, def float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return def
	}
	return v
}

func firstNonEmptyTrimmed(keys ...string) string {
	for _, key := range keys {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmptyTrimmedOrFile(inlineKey, pathKey string) string {
	if v := firstNonEmptyTrimmed(inlineKey); v != "" {
		return v
	}
	path := firstNonEmptyTrimmed(pathKey)
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func stringDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func trimTrailingSlash(v string) string {
	return strings.TrimRight(v, "/")
}

func canvasAllowedHostSuffixes() []string {
	raw := strings.TrimSpace(os.Getenv("CANVAS_ALLOWED_HOST_SUFFIXES"))
	if raw == "" {
		return append([]string(nil), defaultCanvasAllowedHostSuffixes...)
	}
	parts := strings.Split(raw, ",")
	suffixes := make([]string, 0, len(parts))
	for _, part := range parts {
		suffix := strings.ToLower(strings.TrimSpace(part))
		suffix = strings.TrimPrefix(suffix, "*.")
		suffix = strings.TrimPrefix(suffix, ".")
		if suffix != "" {
			suffixes = append(suffixes, suffix)
		}
	}
	if len(suffixes) == 0 {
		return append([]string(nil), defaultCanvasAllowedHostSuffixes...)
	}
	return suffixes
}

func platformSecretsKeyFromEnv() []byte {
	raw := strings.TrimSpace(os.Getenv("PLATFORM_SECRETS_KEY"))
	if raw == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(b) != 32 {
		return nil
	}
	return b
}

func smtpPort() uint16 {
	raw := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if raw == "" {
		return 587
	}
	n, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 587
	}
	return uint16(n)
}

func httpAddr() string {
	p := strings.TrimSpace(os.Getenv("PORT"))
	if p == "" {
		return ":8080"
	}
	if strings.HasPrefix(p, ":") {
		return p
	}
	if n, err := strconv.Atoi(p); err == nil && n >= 0 {
		return ":" + p
	}
	// e.g. "127.0.0.1:8080"
	return p
}

func canvasImportConcurrency() int {
	raw := strings.TrimSpace(os.Getenv("CANVAS_IMPORT_CONCURRENCY"))
	if raw == "" {
		return 3
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 3
	}
	return n
}

func canvasSubmissionSyncConcurrency() int {
	raw := strings.TrimSpace(os.Getenv("CANVAS_SUBMISSION_SYNC_CONCURRENCY"))
	if raw == "" {
		return 5
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 5
	}
	return n
}

func smsNotificationConcurrency() int {
	raw := strings.TrimSpace(os.Getenv("SMS_NOTIFICATION_CONCURRENCY"))
	if raw == "" {
		return 5
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 5
	}
	return n
}

func tusUploadTTLHours() int {
	raw := strings.TrimSpace(os.Getenv("TUS_UPLOAD_TTL_HOURS"))
	if raw == "" {
		return 48
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 48
	}
	return n
}

func storagePresignTTL() int {
	raw := strings.TrimSpace(os.Getenv("STORAGE_PRESIGN_TTL_SECS"))
	if raw == "" {
		return 3600
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 3600
	}
	return n
}

func transcodeRetainSourceDays() int {
	raw := strings.TrimSpace(os.Getenv("TRANSCODE_RETAIN_SOURCE_DAYS"))
	if raw == "" {
		return 30
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 30
	}
	return n
}

func appEnv() string {
	return stringDefault(strings.ToLower(strings.TrimSpace(firstNonEmptyTrimmed("APP_ENV"))), "local")
}

func commaSeparatedEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func statusPageSummaryCacheSecs() int {
	raw := strings.TrimSpace(os.Getenv("STATUS_PAGE_SUMMARY_CACHE_SECS"))
	if raw == "" {
		return 60
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 60
	}
	return n
}

func statuspageComponentMapJSON() string {
	if raw := strings.TrimSpace(os.Getenv("STATUSPAGE_COMPONENT_MAP_JSON")); raw != "" {
		return raw
	}
	path := strings.TrimSpace(os.Getenv("STATUSPAGE_COMPONENT_MAP_PATH"))
	if path == "" {
		path = "config/statuspage-components.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func storageUseSSL() bool {
	v, ok := os.LookupEnv("STORAGE_USE_SSL")
	if !ok {
		// Default to true for security
		return true
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
