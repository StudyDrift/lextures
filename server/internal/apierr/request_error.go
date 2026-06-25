package apierr

import (
	"context"
	"net/http"
)

type serverErrorKey struct{}

type serverErrorState struct {
	message string
	err     error
}

// WithServerErrorTracking attaches a per-request slot for the underlying error behind 5xx responses.
func WithServerErrorTracking(r *http.Request) *http.Request {
	if r == nil {
		return r
	}
	if _, ok := r.Context().Value(serverErrorKey{}).(*serverErrorState); ok {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), serverErrorKey{}, &serverErrorState{}))
}

// RecordServerError stores the server-side failure for inclusion in the access log.
func RecordServerError(r *http.Request, message string, err error) {
	if r == nil {
		return
	}
	state, ok := r.Context().Value(serverErrorKey{}).(*serverErrorState)
	if !ok || state == nil {
		return
	}
	state.message = message
	state.err = err
}

// ServerErrorFromRequest returns the recorded 5xx message and error, if any.
func ServerErrorFromRequest(r *http.Request) (message string, err error) {
	if r == nil {
		return "", nil
	}
	state, ok := r.Context().Value(serverErrorKey{}).(*serverErrorState)
	if !ok || state == nil {
		return "", nil
	}
	return state.message, state.err
}