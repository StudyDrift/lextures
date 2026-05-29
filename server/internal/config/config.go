// Package config loads process configuration from the environment.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// JWTSecretMinLen matches the legacy Rust server's minimum accepted JWT secret length.
	JWTSecretMinLen = 32

	insecureJWTFallback = "dev-secret-do-not-use-in-production"
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

	LTIEnabled          bool
	LTIAPIBaseURL       string
	LTIRSAPrivateKeyPEM string
	LTIRSAKeyID         string

	AnnotationEnabled           bool
	FeedbackMediaEnabled        bool
	BlindGradingEnabled         bool
	ModeratedGradingEnabled     bool
	OriginalityDetectionEnabled bool
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

	CleverSSOEnabled   bool
	CleverClientID     string
	CleverClientSecret string
	CleverDistrictID   string // optional; skips Clever school picker when set

	ClassLinkSSOEnabled         bool
	ClassLinkOIDCIssuer         string // e.g. https://launchpad.classlink.com/v2_0/sis/{tenant}
	ClassLinkOIDCClientID       string
	ClassLinkOIDCClientSecret   string

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

	// SessionManagementUIEnabled gates /api/v1/me/sessions and related UI (plan 4.9).
	SessionManagementUIEnabled bool

	// EmailNotificationsEnabled gates event-driven transactional email (plan 6.2).
	EmailNotificationsEnabled bool

	// PushNotificationsEnabled gates Web Push (VAPID) notifications (plan 6.3).
	PushNotificationsEnabled bool
	// VAPIDPublicKey is the base64url-encoded P-256 public key for VAPID.
	VAPIDPublicKey string
	// VAPIDPrivateKey is the base64url-encoded P-256 private key for VAPID signing.
	VAPIDPrivateKey string
	// VAPIDSubject is the mailto: or https: URI sent in the VAPID JWT sub claim.
	VAPIDSubject string

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
	// ClamAVAddr is the clamd TCP address (default localhost:3310).
	ClamAVAddr string
	// ClamAVStub when true uses in-process EICAR detection (tests/dev without clamd).
	ClamAVStub bool

	// OERLibraryEnabled gates the OER search and import UI (plan 8.9).
	OERLibraryEnabled bool
	// OERStub uses embedded catalog data instead of live OER provider APIs (dev/e2e).
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
	// InstructorInsightsEnabled gates the "What's Working" instructor signals dashboard (plan 9.10).
	InstructorInsightsEnabled bool

	// EquationEditorEnabled gates the visual equation editor in the web client (plan 8.11).
	EquationEditorEnabled bool
	// ReadingLevelEnabled gates Flesch-Kincaid scoring and AI content simplification (plan 11.6).
	ReadingLevelEnabled bool
	// AltTextEnforcementEnabled gates alt-text prompts, AI suggestions, and coverage reporting (plan 12.5).
	AltTextEnforcementEnabled bool
	// FFAltTextEnforcement when true hard-blocks content save until alt text is resolved (plan 12.5).
	FFAltTextEnforcement bool
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
	// DataResidencyEnabled gates per-tenant region pinning enforcement and the data residency compliance admin API (plan 10.12).
	DataResidencyEnabled bool
	// AiDisclosureEnabled gates AI opt-out, gateway enforcement, inference logging, and disclosure APIs (plan 10.17). Defaults to true.
	AiDisclosureEnabled bool
	// SecurityDisclosureModuleEnabled gates responsible-disclosure report triage APIs (plan 10.16).
	SecurityDisclosureModuleEnabled bool
	// BackupModuleEnabled gates backup/restore ops: backup status and restore drill APIs (plan 10.15).
	BackupModuleEnabled bool
	// RTLEnabled gates mirrored RTL layout for RTL locales (plan 11.2). Defaults to false until audit complete.
	RTLEnabled bool

	// AppEnv is the deployment environment (local, staging, production). Used for PII redaction guards (plan 10.14).
	AppEnv string
	// DisablePIIRedaction allows plaintext PII in operational logs for local debugging only (plan 10.14).
	DisablePIIRedaction bool
	// PIIRedactFields adds extra structured log field names to the redaction registry (REDACT_FIELDS).
	PIIRedactFields []string
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

	return Config{
		HTTPAddr: httpAddr(),

		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:        jwtSecret,
		AllowInsecureJWT: allowInsecureJWT,
		RunMigrations:    runMigrations(),

		BootstrapAdminEmail: strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),

		OpenRouterAPIKey: firstNonEmptyTrimmed("OPENROUTER_API_KEY", "OPEN_ROUTER_API_KEY"),
		CourseFilesRoot:  stringDefault(firstNonEmptyTrimmed("COURSE_FILES_ROOT"), "data/course-files"),

		CanvasAllowedHostSuffixes: canvasAllowedHostSuffixes(),
		PublicWebOrigin:           trimTrailingSlash(stringDefault(firstNonEmptyTrimmed("PUBLIC_WEB_ORIGIN"), "http://localhost:5173")),
		BrandingMultitenantHostSuffix: strings.TrimSpace(strings.ToLower(firstNonEmptyTrimmed("BRANDING_MULTITENANT_HOST_SUFFIX"))),

		PlatformSecretsKey: platformSecretsKeyFromEnv(),

		SMTPHost:     firstNonEmptyTrimmed("SMTP_HOST"),
		SMTPPort:     smtpPort(),
		SMTPUser:     firstNonEmptyTrimmed("SMTP_USER"),
		SMTPPassword: firstNonEmptyTrimmed("SMTP_PASSWORD"),
		SMTPFrom:     firstNonEmptyTrimmed("SMTP_FROM"),

		LTIAPIBaseURL:       ltiBaseURL,
		LTIRSAPrivateKeyPEM: firstNonEmptyTrimmed("LTI_RSA_PRIVATE_KEY_PEM"),
		LTIRSAKeyID:         stringDefault(firstNonEmptyTrimmed("LTI_RSA_KEY_ID"), "lti-key-1"),

		SAMLSSOEnabled: false,
		SAMLPublicBaseURL:   samlBaseURL,
		SAMLSPEntityID:      stringDefault(firstNonEmptyTrimmed("SAML_SP_ENTITY_ID"), samlBaseURL+"/auth/saml/metadata"),
		SAMLSPX509PEM:       firstNonEmptyTrimmedOrFile("SAML_SP_X509_PEM", "SAML_SP_X509_PATH"),
		SAMLSPPrivateKeyPEM: firstNonEmptyTrimmedOrFile("SAML_SP_PRIVATE_KEY_PEM", "SAML_SP_PRIVATE_KEY_PATH"),

		OIDCSSOEnabled: false,
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

		CleverSSOEnabled: false,
		CleverClientID:     firstNonEmptyTrimmed("CLEVER_CLIENT_ID", "CLEVER_OIDC_CLIENT_ID"),
		CleverClientSecret: firstNonEmptyTrimmed("CLEVER_CLIENT_SECRET", "CLEVER_OIDC_CLIENT_SECRET"),
		CleverDistrictID:   firstNonEmptyTrimmed("CLEVER_DISTRICT_ID"),

		ClassLinkSSOEnabled: false,
		ClassLinkOIDCIssuer:       strings.TrimRight(firstNonEmptyTrimmed("CLASSLINK_OIDC_ISSUER"), "/"),
		ClassLinkOIDCClientID:     firstNonEmptyTrimmed("CLASSLINK_OIDC_CLIENT_ID"),
		ClassLinkOIDCClientSecret: firstNonEmptyTrimmed("CLASSLINK_OIDC_CLIENT_SECRET"),

		OneRosterEnabled: false,
		OneRosterBearerFallbackToken: firstNonEmptyTrimmed("ONEROSTER_BEARER_FALLBACK_TOKEN"),
		OneRosterBearerFallbackInst:  strings.TrimSpace(os.Getenv("ONEROSTER_BEARER_FALLBACK_INSTITUTION_ID")),

		ScimEnabled: false,

		MFAEnforcement: "none",

		PushNotificationsEnabled: false,
		VAPIDPublicKey:           firstNonEmptyTrimmed("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:          firstNonEmptyTrimmed("VAPID_PRIVATE_KEY"),
		VAPIDSubject:             stringDefault(firstNonEmptyTrimmed("VAPID_SUBJECT"), "mailto:admin@lextures.com"),

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

		XAPIEmissionEnabled: boolEnv("XAPI_EMISSION_ENABLED") || boolEnv("FEATURE_XAPI_EMISSION"),
		LRSAnonymizeActors:  boolEnv("LRS_ANONYMIZE_ACTORS"),

		FERPAWorkflowEnabled: boolEnv("FERPA_WORKFLOW_ENABLED") || boolEnv("FEATURE_FERPA_WORKFLOW"),
		CoppaWorkflowEnabled: boolEnv("COPPA_WORKFLOW_ENABLED") || boolEnv("FEATURE_COPPA_WORKFLOW"),
		GDPRModuleEnabled:    boolEnv("GDPR_MODULE_ENABLED") || boolEnv("FEATURE_GDPR_MODULE"),
		CCPAModuleEnabled:    boolEnv("CCPA_MODULE_ENABLED") || boolEnv("FEATURE_CCPA_MODULE"),
		DPAPortalEnabled:     boolEnv("DPA_PORTAL_ENABLED") || boolEnv("FEATURE_DPA_PORTAL"),
		StatePrivacyEnabled:  boolEnv("STATE_PRIVACY_ENABLED") || boolEnv("FEATURE_STATE_PRIVACY"),
		SOC2ModuleEnabled:    boolEnv("SOC2_MODULE_ENABLED") || boolEnv("FEATURE_SOC2_MODULE"),
		IsoIsmsEnabled:       boolEnv("ISO_ISMS_ENABLED") || boolEnv("FEATURE_ISO_ISMS"),
		AdminAuditLogEnabled: true, // plan 10.11 default on; disable via platform settings
		DataResidencyEnabled:            boolEnv("DATA_RESIDENCY_ENABLED") || boolEnv("FEATURE_DATA_RESIDENCY"),
		AiDisclosureEnabled:             !boolEnv("AI_DISCLOSURE_DISABLED"),
		SecurityDisclosureModuleEnabled: boolEnv("SECURITY_DISCLOSURE_MODULE_ENABLED") || boolEnv("FEATURE_SECURITY_DISCLOSURE"),
		BackupModuleEnabled:             boolEnv("BACKUP_MODULE_ENABLED") || boolEnv("FEATURE_BACKUP_MODULE"),
		RTLEnabled:                      boolEnv("RTL_ENABLED") || boolEnv("FEATURE_RTL_ENABLED"),

		AppEnv:              appEnv(),
		DisablePIIRedaction: boolEnv("DISABLE_PII_REDACTION"),
		PIIRedactFields:     commaSeparatedEnv("REDACT_FIELDS"),
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

func boolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
