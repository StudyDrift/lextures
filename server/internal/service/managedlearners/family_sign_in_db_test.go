package managedlearners

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func familyTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

func TestFamilySignIn_PinFlow_Pg(t *testing.T) {
	ctx, pool := familyTestPool(t)
	signer := auth.NewJWTSigner("01234567890123456789012345678901")

	stamp := time.Now().Format("20060102150405.000000")
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	parentRow, err := user.InsertUser(ctx, pool, "ml-family-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	otherRow, err := user.InsertUser(ctx, pool, "ml-family-other-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	otherID := uuid.MustParse(otherRow.ID)

	dep, err := Create(ctx, pool, CreateParams{ActorID: parentID, DisplayName: "Family Kid"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	depID := uuid.MustParse(dep.ID)

	token, dev, err := RegisterFamilyDevice(ctx, pool, parentID, "")
	if err != nil {
		t.Fatalf("register device: %v", err)
	}
	if dev.Label != defaultDeviceLabel || token == "" {
		t.Fatalf("device: %+v token=%q", dev, token)
	}

	// Picker lists the learner even before a PIN is set.
	picker, err := FamilyPicker(ctx, pool, token)
	if err != nil {
		t.Fatalf("picker: %v", err)
	}
	if len(picker) != 1 || picker[0].ID != dep.ID || picker[0].SignInMethod != "none" {
		t.Fatalf("picker: %+v", picker)
	}
	if _, err := FamilyPicker(ctx, pool, "not-a-device"); !errors.Is(err, ErrDeviceNotFamily) {
		t.Fatalf("unknown device: %v", err)
	}

	signIn := func(pin string) (SessionStartResult, error) {
		return FamilySignIn(ctx, pool, signer, FamilySignInParams{DeviceToken: token, LearnerID: depID, Pin: pin})
	}
	if _, err := signIn("1234"); !errors.Is(err, ErrSignInNotSet) {
		t.Fatalf("no pin yet: %v", err)
	}

	// Only the managing parent may set the PIN.
	if _, err := SetSignIn(ctx, pool, SetSignInParams{ActorID: otherID, DependentID: depID, Method: "pin", Pin: "2468"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other parent set pin: %v", err)
	}
	updated, err := SetSignIn(ctx, pool, SetSignInParams{ActorID: parentID, DependentID: depID, Method: "pin", Pin: "2468"})
	if err != nil {
		t.Fatalf("set pin: %v", err)
	}
	if updated.SignInMethod != SignInMethodPin {
		t.Fatalf("method: %+v", updated)
	}

	for i := 1; i < MaxFailedSignIns; i++ {
		_, err := signIn("0000")
		var wrong *WrongSecretError
		if !errors.As(err, &wrong) {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if wrong.AttemptsRemaining != MaxFailedSignIns-i {
			t.Fatalf("attempt %d remaining %d", i, wrong.AttemptsRemaining)
		}
	}
	_, err = signIn("0000")
	var locked *LockedError
	if !errors.As(err, &locked) || !locked.Until.After(time.Now()) {
		t.Fatalf("expected lockout, got %v", err)
	}
	// Even the right PIN is refused while locked.
	if _, err := signIn("2468"); !errors.As(err, &locked) {
		t.Fatalf("locked right pin: %v", err)
	}
	var notified int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM settings.notifications WHERE user_id = $1 AND event_type = $2
`, parentID, lockedNotifEventType).Scan(&notified); err != nil {
		t.Fatalf("notifications: %v", err)
	}
	if notified != 1 {
		t.Fatalf("parent notifications = %d", notified)
	}
	list, err := List(ctx, pool, parentID)
	if err != nil || len(list) != 1 || list[0].SignInLockedUntil == nil {
		t.Fatalf("list after lock: %+v err %v", list, err)
	}

	// Resetting the PIN clears the lockout.
	if _, err := SetSignIn(ctx, pool, SetSignInParams{ActorID: parentID, DependentID: depID, Method: "pin", Pin: "1357"}); err != nil {
		t.Fatalf("reset pin: %v", err)
	}
	res, err := signIn("1357")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	ml, err := signer.VerifyManagedLearner(res.Token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ml.Via != auth.ManagedLearnerViaFamilyDevice || ml.TargetUserID != dep.ID || ml.ActorID != parentID.String() {
		t.Fatalf("session: %+v", ml)
	}
	if err := EndSession(ctx, pool, EndSessionParams{JTI: ml.JTI, ActorID: parentID, TargetUserID: depID}); err != nil {
		t.Fatalf("end: %v", err)
	}

	// Disable, then revoke the device.
	disabled, err := DisableSignIn(ctx, pool, parentID, depID)
	if err != nil || disabled.SignInMethod != "none" {
		t.Fatalf("disable: %+v err %v", disabled, err)
	}
	if _, err := signIn("1357"); !errors.Is(err, ErrSignInNotSet) {
		t.Fatalf("disabled: %v", err)
	}
	devices, err := ListFamilyDevices(ctx, pool, parentID)
	if err != nil || len(devices) != 1 {
		t.Fatalf("devices: %+v err %v", devices, err)
	}
	if err := RevokeFamilyDevice(ctx, pool, otherID, uuid.MustParse(dev.ID)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other parent revoke: %v", err)
	}
	if err := RevokeFamilyDevice(ctx, pool, parentID, uuid.MustParse(dev.ID)); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := FamilyPicker(ctx, pool, token); !errors.Is(err, ErrDeviceNotFamily) {
		t.Fatalf("revoked device: %v", err)
	}
}

func TestFamilySignIn_PictureAndScope_Pg(t *testing.T) {
	ctx, pool := familyTestPool(t)
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	stamp := time.Now().Format("20060102150405.000000")
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	parentRow, err := user.InsertUser(ctx, pool, "ml-family-pic-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	otherRow, err := user.InsertUser(ctx, pool, "ml-family-pic-other-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	otherID := uuid.MustParse(otherRow.ID)
	dep, err := Create(ctx, pool, CreateParams{ActorID: parentID, DisplayName: "Picture Kid"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	otherDep, err := Create(ctx, pool, CreateParams{ActorID: otherID, DisplayName: "Not Yours"})
	if err != nil {
		t.Fatalf("create other: %v", err)
	}
	seq := []string{"cat", "sun", "tree"}
	if _, err := SetSignIn(ctx, pool, SetSignInParams{ActorID: parentID, DependentID: uuid.MustParse(dep.ID), Method: "picture", Picture: seq}); err != nil {
		t.Fatalf("set picture: %v", err)
	}
	token, _, err := RegisterFamilyDevice(ctx, pool, parentID, "Kitchen laptop")
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	picker, err := FamilyPicker(ctx, pool, token)
	if err != nil || len(picker) != 1 || picker[0].SignInMethod != SignInMethodPicture {
		t.Fatalf("picker: %+v err %v", picker, err)
	}
	// Another family's learner is not reachable from this device.
	if _, err := FamilySignIn(ctx, pool, signer, FamilySignInParams{DeviceToken: token, LearnerID: uuid.MustParse(otherDep.ID), Picture: seq}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-family sign-in: %v", err)
	}
	var wrong *WrongSecretError
	if _, err := FamilySignIn(ctx, pool, signer, FamilySignInParams{DeviceToken: token, LearnerID: uuid.MustParse(dep.ID), Picture: []string{"tree", "sun", "cat"}}); !errors.As(err, &wrong) {
		t.Fatalf("wrong order: %v", err)
	}
	if _, err := FamilySignIn(ctx, pool, signer, FamilySignInParams{DeviceToken: token, LearnerID: uuid.MustParse(dep.ID), Picture: seq}); err != nil {
		t.Fatalf("picture sign-in: %v", err)
	}
}
