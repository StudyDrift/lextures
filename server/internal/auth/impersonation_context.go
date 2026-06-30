package auth

import "context"

// ImpersonationSession carries actor metadata for an active impersonation JWT.
type ImpersonationSession struct {
	AdminID      string
	TargetUserID string
	JTI          string
}

type impersonationCtxKey struct{}

// WithImpersonation attaches impersonation session metadata to ctx.
func WithImpersonation(ctx context.Context, s ImpersonationSession) context.Context {
	return context.WithValue(ctx, impersonationCtxKey{}, s)
}

// ImpersonationFromContext returns impersonation metadata when the request used an impersonation JWT.
func ImpersonationFromContext(ctx context.Context) (ImpersonationSession, bool) {
	v, ok := ctx.Value(impersonationCtxKey{}).(ImpersonationSession)
	if !ok || v.AdminID == "" || v.TargetUserID == "" {
		return ImpersonationSession{}, false
	}
	return v, true
}
