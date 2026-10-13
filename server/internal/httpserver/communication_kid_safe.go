package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/kidmessaging"
)

const (
	kidSafeSenderDenied    = "You can only message your parent or a teacher they approved."
	kidSafeRecipientDenied = "This learner can only receive messages from their family and approved teachers."
)

// resolveMessageRecipient finds the Compose recipient by user id (preferred) or email and
// applies the kid-safe messaging rule (#775). Writes the error response when ok is false.
func (d Deps) resolveMessageRecipient(w http.ResponseWriter, r *http.Request, senderID uuid.UUID, toUserID, toEmail string) (*user.Row, bool) {
	sender, err := user.FindByID(r.Context(), d.Pool, senderID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not send message.")
		return nil, false
	}
	var recipient *user.Row
	if strings.TrimSpace(toUserID) != "" {
		rid, perr := uuid.Parse(strings.TrimSpace(toUserID))
		if perr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid recipient id.")
			return nil, false
		}
		recipient, err = user.FindByID(r.Context(), d.Pool, rid)
	} else {
		recipient, err = user.FindByEmail(r.Context(), d.Pool, user.NormalizeEmail(strings.TrimSpace(toEmail)))
	}
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not send message.")
		return nil, false
	}
	if recipient == nil || recipient.DeactivatedAt != nil {
		if kidmessaging.IsManaged(sender) {
			// Managed learners never learn whether an arbitrary address exists.
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, kidSafeSenderDenied)
			return nil, false
		}
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No user registered with that email.")
		return nil, false
	}
	if !d.checkKidSafeSend(w, r, sender, recipient) {
		return nil, false
	}
	return recipient, true
}

// checkKidSafeSend writes 403 when a managed learner is on either side and the other side is
// not one of their allowed contacts (parent/guardian or parent-approved teacher).
func (d Deps) checkKidSafeSend(w http.ResponseWriter, r *http.Request, sender, recipient *user.Row) bool {
	err := kidmessaging.CheckSend(r.Context(), d.Pool, sender, recipient)
	if err == nil {
		return true
	}
	if errors.Is(err, kidmessaging.ErrRecipientNotAllowed) {
		msg := kidSafeRecipientDenied
		if kidmessaging.IsManaged(sender) {
			msg = kidSafeSenderDenied
		}
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, msg)
		return false
	}
	apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not send message.")
	return false
}

// handleCommRecipients is GET /api/v1/communication/recipients — the people a managed learner's
// Compose may offer. Other accounts get { restricted: false, recipients: [] } and keep free-text To.
func (d Deps) handleCommRecipients() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load recipients.")
			return
		}
		if !kidmessaging.IsManaged(row) {
			writeJSON(w, http.StatusOK, map[string]any{"restricted": false, "recipients": []kidmessaging.Recipient{}})
			return
		}
		contacts, err := kidmessaging.AllowedContacts(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load recipients.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"restricted": true, "recipients": kidmessaging.ToRecipients(contacts)})
	}
}
