package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAccessKeyAllowsCourse(t *testing.T) {
	t.Parallel()
	courseA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	courseB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ctx := context.WithValue(context.Background(), apiTokenAuthKey{}, &APITokenAuth{
		CourseIDs: []uuid.UUID{courseA},
	})
	if !AccessKeyAllowsCourse(ctx, courseA) {
		t.Fatal("expected courseA allowed")
	}
	if AccessKeyAllowsCourse(ctx, courseB) {
		t.Fatal("expected courseB denied")
	}
	if !AccessKeyAllowsCourse(context.Background(), courseB) {
		t.Fatal("expected non-access-key auth to allow any course")
	}
}
