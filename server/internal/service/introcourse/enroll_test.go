package introcourse

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
)

func TestEnrollPath_Constants(t *testing.T) {
	paths := []EnrollPath{PathSignup, PathSSO, PathClever, PathCanvas, PathAdminImport, PathBackfill}
	seen := make(map[EnrollPath]struct{})
	for _, p := range paths {
		if p == "" {
			t.Fatal("empty enroll path constant")
		}
		if _, ok := seen[p]; ok {
			t.Fatalf("duplicate path: %q", p)
		}
		seen[p] = struct{}{}
	}
}

func TestEnsureEnrollment_NilServiceNoOp(t *testing.T) {
	var svc *Service
	if err := svc.EnsureEnrollment(context.TODO(), testCfg(true), nil, uuid.New(), PathSignup); err != nil {
		t.Fatalf("nil service: %v", err)
	}
}

func testCfg(enabled bool) config.Config {
	return config.Config{IntroCourseEnabled: enabled}
}