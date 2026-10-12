package managedlearners

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/familysignin"
	impersonationrepo "github.com/lextures/lextures/server/internal/repos/impersonation"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Family-device learner sign-in (#771).
const (
	SignInMethodPin     = "pin"
	SignInMethodPicture = "picture"

	PinMinDigits       = 4
	PinMaxDigits       = 6
	PictureMinLength   = 3
	PictureMaxLength   = 6
	MaxFailedSignIns   = 5
	SignInLockDuration = 15 * time.Minute

	deviceLabelMaxRunes  = 80
	defaultDeviceLabel   = "Family device"
	deviceTokenBytes     = 32
	lockedNotifEventType = "managed_learner_sign_in_locked"
)

// PictureChoices is the fixed picture-password alphabet shared with clients.
var PictureChoices = []string{"apple", "ball", "cat", "dog", "fish", "moon", "star", "sun", "tree"}

var (
	ErrSignInNotSet      = errors.New("managedlearners: learner sign-in not set up")
	ErrDeviceNotFamily   = errors.New("managedlearners: device is not a family device")
	ErrSignInUnavailable = errors.New("managedlearners: learner sign-in store unavailable")
)

// WrongSecretError reports a failed PIN / picture attempt that did not lock the learner.
type WrongSecretError struct{ AttemptsRemaining int }

func (e *WrongSecretError) Error() string { return "managedlearners: wrong learner sign-in" }

// LockedError reports a learner locked out after too many failed attempts.
type LockedError struct{ Until time.Time }

func (e *LockedError) Error() string { return "managedlearners: learner sign-in locked" }

// FamilyDevice is the parent-facing view of a registered family device.
type FamilyDevice struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// PickerLearner is a "Who's learning?" entry (no secrets).
type PickerLearner struct {
	ID           string     `json:"id"`
	DisplayName  string     `json:"displayName"`
	SignInMethod string     `json:"signInMethod"`
	LockedUntil  *time.Time `json:"lockedUntil,omitempty"`
}

// SetSignInParams configures PIN or picture login for a managed learner.
type SetSignInParams struct {
	ActorID     uuid.UUID
	DependentID uuid.UUID
	Method      string
	Pin         string
	Picture     []string
}

// FamilySignInParams is a learner's sign-in attempt from a family device.
type FamilySignInParams struct {
	DeviceToken string
	LearnerID   uuid.UUID
	Pin         string
	Picture     []string
}

func isPictureChoice(s string) bool {
	for _, c := range PictureChoices {
		if c == s {
			return true
		}
	}
	return false
}

// normalizeSecret validates a PIN or picture sequence and returns the canonical string to hash.
func normalizeSecret(method, pin string, picture []string) (string, error) {
	switch method {
	case SignInMethodPin:
		p := strings.TrimSpace(pin)
		if len(p) < PinMinDigits || len(p) > PinMaxDigits {
			return "", fmt.Errorf("%w: PIN must be %d–%d digits", ErrInvalidInput, PinMinDigits, PinMaxDigits)
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return "", fmt.Errorf("%w: PIN must contain only digits", ErrInvalidInput)
			}
		}
		return "pin:" + p, nil
	case SignInMethodPicture:
		if len(picture) < PictureMinLength || len(picture) > PictureMaxLength {
			return "", fmt.Errorf("%w: picture password must be %d–%d pictures", ErrInvalidInput, PictureMinLength, PictureMaxLength)
		}
		parts := make([]string, 0, len(picture))
		for _, raw := range picture {
			p := strings.ToLower(strings.TrimSpace(raw))
			if !isPictureChoice(p) {
				return "", fmt.Errorf("%w: unknown picture %q", ErrInvalidInput, raw)
			}
			parts = append(parts, p)
		}
		return "picture:" + strings.Join(parts, ","), nil
	default:
		return "", fmt.Errorf("%w: method must be pin or picture", ErrInvalidInput)
	}
}

func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newDeviceToken() (string, error) {
	b := make([]byte, deviceTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func normalizeDeviceLabel(s string) string {
	l := strings.TrimSpace(s)
	if l == "" {
		return defaultDeviceLabel
	}
	if utf8.RuneCountInString(l) > deviceLabelMaxRunes {
		l = string([]rune(l)[:deviceLabelMaxRunes])
	}
	return l
}

func assertOwnsActive(ctx context.Context, pool *pgxpool.Pool, actorID, dependentID uuid.UUID) (uuid.UUID, error) {
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return uuid.Nil, err
	}
	ok, err := ownsManaged(ctx, pool, orgID, actorID, dependentID)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, ErrNotFound
	}
	return orgID, nil
}

// SetSignIn sets or resets a learner's PIN / picture login (clears any lockout).
func SetSignIn(ctx context.Context, pool *pgxpool.Pool, p SetSignInParams) (*Dependent, error) {
	method := strings.ToLower(strings.TrimSpace(p.Method))
	canonical, err := normalizeSecret(method, p.Pin, p.Picture)
	if err != nil {
		return nil, err
	}
	orgID, err := assertOwnsActive(ctx, pool, p.ActorID, p.DependentID)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(canonical)
	if err != nil {
		return nil, fmt.Errorf("managedlearners: hash sign-in: %w", err)
	}
	if err := familysignin.UpsertSignIn(ctx, pool, p.DependentID, method, hash, p.ActorID); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("managed_learner_sign_in_set")
	return getDependent(ctx, pool, orgID, p.ActorID, p.DependentID)
}

// DisableSignIn removes a learner's PIN / picture login.
func DisableSignIn(ctx context.Context, pool *pgxpool.Pool, actorID, dependentID uuid.UUID) (*Dependent, error) {
	orgID, err := assertOwnsActive(ctx, pool, actorID, dependentID)
	if err != nil {
		return nil, err
	}
	if err := familysignin.DeleteSignIn(ctx, pool, dependentID); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("managed_learner_sign_in_disabled")
	return getDependent(ctx, pool, orgID, actorID, dependentID)
}

// RegisterFamilyDevice marks the caller's browser as a family device and returns its secret once.
func RegisterFamilyDevice(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID, label string) (string, *FamilyDevice, error) {
	if _, _, err := assertCanManageDependents(ctx, pool, actorID); err != nil {
		return "", nil, err
	}
	token, err := newDeviceToken()
	if err != nil {
		return "", nil, fmt.Errorf("managedlearners: device token: %w", err)
	}
	row, err := familysignin.InsertDevice(ctx, pool, actorID, hashDeviceToken(token), normalizeDeviceLabel(label))
	if err != nil {
		return "", nil, err
	}
	telemetry.RecordBusinessEvent("family_device_registered")
	return token, deviceFromRow(*row), nil
}

// ListFamilyDevices returns the parent's active family devices.
func ListFamilyDevices(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID) ([]FamilyDevice, error) {
	if _, _, err := assertCanManageDependents(ctx, pool, actorID); err != nil {
		return nil, err
	}
	rows, err := familysignin.ListDevices(ctx, pool, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]FamilyDevice, 0, len(rows))
	for _, r := range rows {
		out = append(out, *deviceFromRow(r))
	}
	return out, nil
}

// RevokeFamilyDevice turns off learner sign-in on one of the parent's devices.
func RevokeFamilyDevice(ctx context.Context, pool *pgxpool.Pool, actorID, deviceID uuid.UUID) error {
	if _, _, err := assertCanManageDependents(ctx, pool, actorID); err != nil {
		return err
	}
	ok, err := familysignin.RevokeDevice(ctx, pool, actorID, deviceID, time.Now())
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func deviceFromRow(r familysignin.Device) *FamilyDevice {
	return &FamilyDevice{ID: r.ID.String(), Label: r.Label, CreatedAt: r.CreatedAt.UTC(), LastUsedAt: r.LastUsedAt}
}

// resolveFamilyDevice maps a device secret to the parent who registered it.
func resolveFamilyDevice(ctx context.Context, pool *pgxpool.Pool, token string) (*familysignin.Device, uuid.UUID, error) {
	t := strings.TrimSpace(token)
	if t == "" || len(t) > 128 {
		return nil, uuid.Nil, ErrDeviceNotFamily
	}
	dev, err := familysignin.FindActiveDeviceByHash(ctx, pool, hashDeviceToken(t))
	if err != nil {
		return nil, uuid.Nil, err
	}
	if dev == nil {
		return nil, uuid.Nil, ErrDeviceNotFamily
	}
	_, orgID, err := assertCanManageDependents(ctx, pool, dev.ParentUserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrActorBlocked) {
			return nil, uuid.Nil, ErrDeviceNotFamily
		}
		return nil, uuid.Nil, err
	}
	return dev, orgID, nil
}

// FamilyPicker lists the learners a family device may sign in.
func FamilyPicker(ctx context.Context, pool *pgxpool.Pool, deviceToken string) ([]PickerLearner, error) {
	dev, orgID, err := resolveFamilyDevice(ctx, pool, deviceToken)
	if err != nil {
		return nil, err
	}
	rows, err := familysignin.ListPickerLearners(ctx, pool, dev.ParentUserID, orgID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]PickerLearner, 0, len(rows))
	for _, r := range rows {
		p := PickerLearner{ID: r.ID.String(), DisplayName: r.DisplayName, SignInMethod: r.Method}
		if p.SignInMethod == "" {
			p.SignInMethod = "none"
		}
		if r.LockedUntil != nil && r.LockedUntil.After(now) {
			lu := r.LockedUntil.UTC()
			p.LockedUntil = &lu
		}
		out = append(out, p)
	}
	return out, nil
}

// FamilySignIn verifies a learner's PIN / picture on a family device and issues a
// managed_learner session scoped to the learner. The parent's own session is untouched.
func FamilySignIn(ctx context.Context, pool *pgxpool.Pool, signer *auth.JWTSigner, p FamilySignInParams) (SessionStartResult, error) {
	if pool == nil || signer == nil {
		return SessionStartResult{}, ErrStoreDown
	}
	dev, orgID, err := resolveFamilyDevice(ctx, pool, p.DeviceToken)
	if err != nil {
		return SessionStartResult{}, err
	}
	parentID := dev.ParentUserID
	ok, err := ownsManaged(ctx, pool, orgID, parentID, p.LearnerID)
	if err != nil {
		return SessionStartResult{}, err
	}
	if !ok {
		return SessionStartResult{}, ErrNotFound
	}
	login, err := familysignin.GetSignIn(ctx, pool, p.LearnerID)
	if err != nil {
		return SessionStartResult{}, err
	}
	if login == nil {
		return SessionStartResult{}, ErrSignInNotSet
	}
	now := time.Now().UTC()
	if login.LockedUntil != nil && login.LockedUntil.After(now) {
		return SessionStartResult{}, &LockedError{Until: login.LockedUntil.UTC()}
	}
	canonical, normErr := normalizeSecret(login.Method, p.Pin, p.Picture)
	match := false
	if normErr == nil {
		match, err = auth.VerifyPassword(canonical, login.SecretHash)
		if err != nil {
			return SessionStartResult{}, fmt.Errorf("managedlearners: verify sign-in: %w", err)
		}
	}
	child, err := user.FindByID(ctx, pool, p.LearnerID)
	if err != nil || child == nil {
		return SessionStartResult{}, ErrNotFound
	}
	if !match {
		return SessionStartResult{}, recordFailedSignIn(ctx, pool, parentID, child, now)
	}
	if err := familysignin.ResetFailures(ctx, pool, p.LearnerID); err != nil {
		return SessionStartResult{}, err
	}
	orgSlug, err := organization.OrgSlugForUser(ctx, pool, p.LearnerID)
	if err != nil {
		return SessionStartResult{}, err
	}
	tok, sess, err := signer.SignManagedLearnerVia(
		parentID.String(), p.LearnerID.String(), child.Email, orgID.String(), orgSlug, auth.ManagedLearnerViaFamilyDevice,
	)
	if err != nil {
		return SessionStartResult{}, fmt.Errorf("managedlearners: sign: %w", err)
	}
	if err := impersonationrepo.Insert(ctx, pool, sess.JTI, parentID, p.LearnerID, sess.ExpiresAt); err != nil {
		return SessionStartResult{}, ErrStoreDown
	}
	if err := familysignin.TouchDevice(ctx, pool, dev.ID, now); err != nil {
		slog.Warn("family_device.touch_failed", "device_id", dev.ID.String(), "err", err)
	}
	telemetry.RecordBusinessEvent("managed_learner_family_sign_in")
	return SessionStartResult{
		Token:     tok,
		ExpiresAt: sess.ExpiresAt,
		Target:    DependentSummary{ID: child.ID, DisplayName: child.DisplayName},
	}, nil
}

func recordFailedSignIn(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, child *user.Row, now time.Time) error {
	childID, err := uuid.Parse(child.ID)
	if err != nil {
		return ErrNotFound
	}
	attempts, lockedUntil, justLocked, err := familysignin.RecordFailure(ctx, pool, childID, MaxFailedSignIns, SignInLockDuration, now)
	if err != nil {
		return err
	}
	telemetry.RecordBusinessEvent("managed_learner_family_sign_in_failed")
	if justLocked && lockedUntil != nil {
		notifyParentLocked(ctx, pool, parentID, child)
		telemetry.RecordBusinessEvent("managed_learner_family_sign_in_locked")
		return &LockedError{Until: lockedUntil.UTC()}
	}
	remaining := MaxFailedSignIns - attempts
	if remaining < 0 {
		remaining = 0
	}
	return &WrongSecretError{AttemptsRemaining: remaining}
}

func notifyParentLocked(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, child *user.Row) {
	name := "Your learner"
	if child.DisplayName != nil && strings.TrimSpace(*child.DisplayName) != "" {
		name = strings.TrimSpace(*child.DisplayName)
	}
	title := fmt.Sprintf("%s's sign-in is locked", name)
	body := fmt.Sprintf(
		"Too many incorrect sign-in attempts on a family device. %s can try again in %d minutes, or you can reset their PIN or picture on the Learners page.",
		name, int(SignInLockDuration/time.Minute),
	)
	actionURL := "/learners?learner=" + child.ID
	if _, err := notificationsinbox.Insert(ctx, pool, parentID, lockedNotifEventType, title, body, actionURL); err != nil {
		slog.Warn("managed_learner.sign_in_lock_notify_failed", "parent_id", parentID.String(), "err", err)
	}
}

// decorateSignIn fills sign-in status on parent-facing dependent rows.
func decorateSignIn(ctx context.Context, pool *pgxpool.Pool, deps []Dependent) error {
	if len(deps) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(deps))
	for _, d := range deps {
		if id, err := uuid.Parse(d.ID); err == nil {
			ids = append(ids, id)
		}
	}
	statuses, err := familysignin.StatusesFor(ctx, pool, ids)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range deps {
		deps[i].SignInMethod = "none"
		id, err := uuid.Parse(deps[i].ID)
		if err != nil {
			continue
		}
		st, ok := statuses[id]
		if !ok {
			continue
		}
		deps[i].SignInMethod = st.Method
		if st.LockedUntil != nil && st.LockedUntil.After(now) {
			lu := st.LockedUntil.UTC()
			deps[i].SignInLockedUntil = &lu
		}
	}
	return nil
}
