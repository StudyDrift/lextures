package auth

import "context"

// ManagedLearnerSession carries actor metadata for an active Learn-as JWT (#640).
type ManagedLearnerSession struct {
	ActorID      string
	TargetUserID string
	JTI          string
}

type managedLearnerCtxKey struct{}

// WithManagedLearner attaches Learn-as session metadata to ctx.
func WithManagedLearner(ctx context.Context, s ManagedLearnerSession) context.Context {
	return context.WithValue(ctx, managedLearnerCtxKey{}, s)
}

// ManagedLearnerFromContext returns Learn-as metadata when the request used a managed_learner JWT.
func ManagedLearnerFromContext(ctx context.Context) (ManagedLearnerSession, bool) {
	v, ok := ctx.Value(managedLearnerCtxKey{}).(ManagedLearnerSession)
	if !ok || v.ActorID == "" || v.TargetUserID == "" {
		return ManagedLearnerSession{}, false
	}
	return v, true
}
