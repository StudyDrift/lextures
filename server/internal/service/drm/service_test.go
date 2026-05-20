package drm_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/drm"
)

func newService(t *testing.T) *drm.Service {
	t.Helper()
	return drm.New(nil, drm.Config{
		Secret:   []byte("test-secret-32byteslong-padded!x"),
		TokenTTL: time.Hour,
	})
}

func TestSignToken_ValidatesOwn(t *testing.T) {
	svc := newService(t)
	objID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	tok := svc.SignToken(objID, userID)
	if !svc.ValidateToken(tok, objID, userID) {
		t.Fatal("freshly signed token must validate")
	}
}

func TestSignToken_WrongUser(t *testing.T) {
	svc := newService(t)
	objID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	alice := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bob := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	aliceToken := svc.SignToken(objID, alice)
	if svc.ValidateToken(aliceToken, objID, bob) {
		t.Fatal("token issued for Alice must not validate for Bob")
	}
}

func TestSignToken_WrongObject(t *testing.T) {
	svc := newService(t)
	obj1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	obj2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	tok := svc.SignToken(obj1, userID)
	if svc.ValidateToken(tok, obj2, userID) {
		t.Fatal("token for obj1 must not validate for obj2")
	}
}

func TestSignToken_Deterministic(t *testing.T) {
	svc := newService(t)
	objID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t1 := svc.SignToken(objID, userID)
	t2 := svc.SignToken(objID, userID)
	if t1 != t2 {
		t.Fatal("SignToken must be deterministic within the same hour bucket")
	}
}

func TestSignToken_DifferentSecrets(t *testing.T) {
	svc1 := drm.New(nil, drm.Config{Secret: []byte("secret-one-padded-32-bytes-xxxxx"), TokenTTL: time.Hour})
	svc2 := drm.New(nil, drm.Config{Secret: []byte("secret-two-padded-32-bytes-xxxxx"), TokenTTL: time.Hour})
	objID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	tok1 := svc1.SignToken(objID, userID)
	if svc2.ValidateToken(tok1, objID, userID) {
		t.Fatal("token from svc1 must not validate against svc2's secret")
	}
}

func TestSubnetOf(t *testing.T) {
	cases := []struct {
		ip   string
		want string
	}{
		{"192.168.1.42", "192.168.1.0/24"},
		{"10.0.0.1", "10.0.0.0/24"},
		{"not-an-ip", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := drm.SubnetOf(tc.ip)
		if got != tc.want {
			t.Errorf("SubnetOf(%q) = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

func TestAnomalyThreshold(t *testing.T) {
	if drm.AnomalyThreshold != 5 {
		t.Fatalf("AnomalyThreshold should be 5, got %d", drm.AnomalyThreshold)
	}
}
