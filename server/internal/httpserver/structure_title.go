package httpserver

import (
	"errors"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// writeAPIErrorPayloadTitle writes 400 when err is a structure title that is an
// API error envelope. Returns true when the response was written.
func writeAPIErrorPayloadTitle(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, coursestructure.ErrAPIErrorPayloadTitle) {
		return false
	}
	apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title cannot be an API error response.")
	return true
}
