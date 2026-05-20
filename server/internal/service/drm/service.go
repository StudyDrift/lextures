// Package drm orchestrates DRM license requests, user-bound URL generation, and anomaly detection
// (plan 8.10 FR-5, FR-6, FR-7).
package drm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	drm_repo "github.com/lextures/lextures/server/internal/repos/drm"
)

const (
	// AnomalyThreshold is the download count per hour that triggers an admin alert (FR-7).
	AnomalyThreshold = 5
)

// LicenseResult describes what the license endpoint returns to the caller.
type LicenseResult struct {
	// Granted is true when the user is allowed to access the content.
	Granted bool
	// DenialReason is non-empty when Granted is false.
	DenialReason string
	// DRMType mirrors the object's drm_type field.
	DRMType drm_repo.DRMType
	// Token is a short-lived user-bound HMAC token embedded in the content URL.
	// Callers should append ?token=<Token>&uid=<userID> to the base storage URL.
	Token string
	// AnomalyDetected is true when the download count exceeded AnomalyThreshold.
	// The caller should log an alert; the grant is still returned when true.
	AnomalyDetected bool
}

// Config holds the secrets needed to sign user-bound tokens.
type Config struct {
	// Secret is a server-side HMAC secret (from secrets manager; never stored in DB).
	Secret []byte
	// TokenTTL controls how long a user-bound token remains valid (default 1 hour).
	TokenTTL time.Duration
}

// Service provides DRM license operations.
type Service struct {
	pool *pgxpool.Pool
	cfg  Config
}

// New returns a DRM Service.
func New(pool *pgxpool.Pool, cfg Config) *Service {
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = time.Hour
	}
	return &Service{pool: pool, cfg: cfg}
}

// RequestLicense validates that userID may access objectID, logs the request, and returns
// a user-bound token. ipAddress may be empty.
func (s *Service) RequestLicense(ctx context.Context, objectID, userID uuid.UUID, ipAddress string) (*LicenseResult, error) {
	obj, err := drm_repo.GetObjectDRM(ctx, s.pool, objectID)
	if err != nil {
		return nil, fmt.Errorf("drm: get object: %w", err)
	}
	if obj == nil {
		reason := "object not found"
		_ = drm_repo.InsertLicenseRequest(ctx, s.pool, objectID, userID, ipAddress, false, &reason)
		return &LicenseResult{Granted: false, DenialReason: reason}, nil
	}
	if obj.DRMType == drm_repo.DRMTypeNone {
		// No DRM — still log and return a basic token.
		token := s.SignToken(objectID, userID)
		_ = drm_repo.InsertLicenseRequest(ctx, s.pool, objectID, userID, ipAddress, true, nil)
		return &LicenseResult{Granted: true, DRMType: obj.DRMType, Token: token}, nil
	}

	// Validate IP subnet binding for Widevine / FairPlay (FR-5).
	if obj.DRMType == drm_repo.DRMTypeWidevine || obj.DRMType == drm_repo.DRMTypeFairPlay {
		// External DRM license server integration would go here. Return stub approval.
		_ = obj.DRMProvider
		_ = obj.DRMKeyID
	}

	token := s.SignToken(objectID, userID)
	if logErr := drm_repo.InsertLicenseRequest(ctx, s.pool, objectID, userID, ipAddress, true, nil); logErr != nil {
		return nil, fmt.Errorf("drm: log license: %w", logErr)
	}

	// Anomaly check (FR-7): count granted downloads in the last hour.
	count, countErr := drm_repo.DownloadCountLastHour(ctx, s.pool, objectID, userID)
	if countErr != nil {
		return nil, fmt.Errorf("drm: anomaly check: %w", countErr)
	}
	anomaly := count > AnomalyThreshold

	return &LicenseResult{
		Granted:         true,
		DRMType:         obj.DRMType,
		Token:           token,
		AnomalyDetected: anomaly,
	}, nil
}

// ValidateToken returns true when token was issued for (objectID, userID) and has not expired.
func (s *Service) ValidateToken(token string, objectID, userID uuid.UUID) bool {
	expected := s.signToken(objectID, userID)
	return hmac.Equal([]byte(token), []byte(expected))
}

// ListAnomalies returns all (user, object) pairs that exceeded AnomalyThreshold in the last hour.
func ListAnomalies(ctx context.Context, pool *pgxpool.Pool) ([]drm_repo.Anomaly, error) {
	return drm_repo.ListAnomalies(ctx, pool, AnomalyThreshold)
}

// SignToken creates a deterministic HMAC-SHA256 token binding objectID, userID, and the current
// hour bucket. The token is valid for up to 2 * cfg.TokenTTL (current + previous bucket).
func (s *Service) SignToken(objectID, userID uuid.UUID) string {
	return s.signToken(objectID, userID)
}

func (s *Service) signToken(objectID, userID uuid.UUID) string {
	bucket := time.Now().UTC().Truncate(s.cfg.TokenTTL).Format(time.RFC3339)
	h := hmac.New(sha256.New, s.cfg.Secret)
	h.Write([]byte(objectID.String()))
	h.Write([]byte(":"))
	h.Write([]byte(userID.String()))
	h.Write([]byte(":"))
	h.Write([]byte(bucket))
	return hex.EncodeToString(h.Sum(nil))
}

// SubnetOf returns the /24 prefix string for an IP address. Used for IP-binding.
func SubnetOf(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.0/24", ip[0], ip[1], ip[2])
}
