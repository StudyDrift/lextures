package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/l10n"
	"github.com/lextures/lextures/server/internal/repos/user"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
)

var phoneNumberPattern = regexp.MustCompile(`^[\d\s().+-]+$`)

type accountProfileResponse struct {
	Email                      string  `json:"email"`
	DisplayName                *string `json:"displayName"`
	FirstName                  *string `json:"firstName"`
	LastName                   *string `json:"lastName"`
	AvatarURL                  *string `json:"avatarUrl"`
	UITheme                    string  `json:"uiTheme"`
	ShowHelpPopover            bool    `json:"showHelpPopover"`
	Locale                     string  `json:"locale"`
	RTLEnabled                 bool    `json:"rtlEnabled"`
	Sid                        *string `json:"sid"`
	SessionManagementUIEnabled bool    `json:"sessionManagementUiEnabled"`
	AccountType                string  `json:"accountType"`
	Timezone                   *string `json:"timezone"`
	PhoneNumber                *string `json:"phoneNumber"`
}

type patchAccountBody struct {
	FirstName       *string `json:"firstName"`
	LastName        *string `json:"lastName"`
	AvatarURL       *string `json:"avatarUrl"`
	UITheme         *string `json:"uiTheme"`
	ShowHelpPopover *bool   `json:"showHelpPopover"`
	Timezone        *string `json:"timezone"`
	PhoneNumber     *string `json:"phoneNumber"`
}

func normalizeName(s *string, label string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, nil
	}
	if len(t) > 80 {
		return nil, apierrError(label + " is too long.")
	}
	return &t, nil
}

func normalizeAvatarURL(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, nil
	}
	if len(t) > 2_000_000 {
		return nil, apierrError("Avatar image URL is too long.")
	}
	isHTTP := strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://")
	isData := strings.HasPrefix(t, "data:image/")
	if !isHTTP && !isData {
		return nil, apierrError("Avatar must be an http(s) URL or a data:image upload.")
	}
	return &t, nil
}

func normalizePhoneNumber(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, nil
	}
	if len(t) > 30 {
		return nil, apierrError("Phone number is too long.")
	}
	if !phoneNumberPattern.MatchString(t) {
		return nil, apierrError("Phone number may only contain digits, spaces, and + ( ) . - characters.")
	}
	return &t, nil
}

func normalizeTheme(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	t := strings.ToLower(strings.TrimSpace(*s))
	if t != "light" && t != "dark" {
		return nil, apierrError("Theme must be \"light\" or \"dark\".")
	}
	return &t, nil
}

type apierrValidationError struct{ msg string }

func (e apierrValidationError) Error() string { return e.msg }

func apierrError(msg string) error { return apierrValidationError{msg: msg} }

func localeOrDefault(locale string) string {
	if strings.TrimSpace(locale) == "" {
		return "en"
	}
	return locale
}

func (d Deps) handleGetSettingsAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load account.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		at := row.AccountType
		if at == "" {
			at = user.AccountTypeStandard
		}
		writeJSON(w, http.StatusOK, accountProfileResponse{
			Email:                      row.Email,
			DisplayName:                row.DisplayName,
			FirstName:                  row.FirstName,
			LastName:                   row.LastName,
			AvatarURL:                  row.AvatarURL,
			UITheme:                    row.UITheme,
			ShowHelpPopover:            row.ShowHelpPopover,
			Locale:                     localeOrDefault(row.Locale),
			Sid:                        row.Sid,
			SessionManagementUIEnabled: d.effectiveConfig().SessionManagementUIEnabled,
			RTLEnabled:                 d.effectiveConfig().RTLEnabled,
			AccountType:                at,
			Timezone:                   row.Timezone,
			PhoneNumber:                row.PhoneNumber,
		})
	}
}

func (d Deps) handlePatchSettingsAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req patchAccountBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		firstName, err := normalizeName(req.FirstName, "First name")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		lastName, err := normalizeName(req.LastName, "Last name")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		avatarURL, err := normalizeAvatarURL(req.AvatarURL)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		uiTheme, err := normalizeTheme(req.UITheme)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		var timezonePtr *string
		if req.Timezone != nil {
			norm, err := l10n.NormalizeTimezone(*req.Timezone)
			if err != nil {
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, "Invalid IANA timezone identifier.")
				return
			}
			timezonePtr = &norm
		}
		var phoneNumberPtr *string
		updatePhoneNumber := false
		if req.PhoneNumber != nil {
			updatePhoneNumber = true
			phoneNumber, err := normalizePhoneNumber(req.PhoneNumber)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			phoneNumberPtr = phoneNumber
		}
		row, err := user.UpdateProfile(r.Context(), d.Pool, userID, firstName, lastName, avatarURL, uiTheme, req.ShowHelpPopover, timezonePtr, phoneNumberPtr, updatePhoneNumber)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update account.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		at := row.AccountType
		if at == "" {
			at = user.AccountTypeStandard
		}
		writeJSON(w, http.StatusOK, accountProfileResponse{
			Email:                      row.Email,
			DisplayName:                row.DisplayName,
			FirstName:                  row.FirstName,
			LastName:                   row.LastName,
			AvatarURL:                  row.AvatarURL,
			UITheme:                    row.UITheme,
			ShowHelpPopover:            row.ShowHelpPopover,
			Locale:                     localeOrDefault(row.Locale),
			Sid:                        row.Sid,
			SessionManagementUIEnabled: d.effectiveConfig().SessionManagementUIEnabled,
			RTLEnabled:                 d.effectiveConfig().RTLEnabled,
			AccountType:                at,
			Timezone:                   row.Timezone,
			PhoneNumber:                row.PhoneNumber,
		})
	}
}

// handleDeleteSettingsAccount is DELETE /api/v1/settings/account — self-service account deletion.
// Permanently anonymizes the signed-in user (same erasure path as admin people delete).
func (d Deps) handleDeleteSettingsAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		result, err := d.eraseUserAccount(ctx, userID, false)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			if errors.Is(err, errAccountAlreadyErased) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This account has already been deleted.")
				return
			}
			if errors.Is(err, errAccountSystemProtected) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "System accounts cannot be deleted.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete account.")
			return
		}

		orgID := result.OrgID
		d.recordPlatformPeopleAudit(r, userID, &orgID, auditservice.EventUserDeactivate, userID, nil)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": userID.String()})
	}
}
