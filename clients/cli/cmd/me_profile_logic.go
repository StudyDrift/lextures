package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type meProfile struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName"`
	Org         *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"org"`
}

type sessionRow struct {
	ID        string `json:"id"`
	UserAgent string `json:"userAgent"`
	IPAddress string `json:"ipAddress"`
	CreatedAt string `json:"createdAt"`
	Current   bool   `json:"current"`
}

type mfaRow struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
}

type oidcIdentityRow struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Subject  string `json:"subject"`
	Email    string `json:"email"`
}

type parentChildRow struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

func displayName(me meProfile) string {
	if me.DisplayName != nil && *me.DisplayName != "" {
		return *me.DisplayName
	}
	return me.Email
}

func fetchMeProfile(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func patchMeProfileFields(c *client.Client, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/me/profile-fields", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func listMySessions(c *client.Client) ([]sessionRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/sessions", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Sessions []sessionRow `json:"sessions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Sessions, body, nil
}

func revokeSession(c *client.Client, id string) error {
	path := "/api/v1/me/sessions/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func revokeOtherSessions(c *client.Client) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/me/sessions", nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func listMyMFA(c *client.Client) ([]mfaRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/mfa", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Factors []mfaRow `json:"factors"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Factors, body, nil
}

func deleteMyMFA(c *client.Client, id string) error {
	path := "/api/v1/me/mfa/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func listOIDCIdentities(c *client.Client) ([]oidcIdentityRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/oidc-identities", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Identities []oidcIdentityRow `json:"identities"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Identities, body, nil
}

func unlinkOIDCIdentity(c *client.Client, id string) error {
	path := "/api/v1/me/oidc-identities/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func getMyDemographics(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/demographics")
}

func patchMyDemographics(c *client.Client, payload map[string]any) ([]byte, error) {
	return mePATCH(c, "/api/v1/me/demographics", payload)
}

func getMyProfileFields(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/profile-fields")
}

func listConsentStudies(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/consent-studies")
}

func respondConsentStudy(c *client.Client, id string, accept bool) ([]byte, error) {
	return mePOST(c, "/api/v1/me/consent-studies/"+url.PathEscape(id)+"/respond",
		map[string]any{"accepted": accept})
}

func getOnboardingStatus(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/onboarding-status")
}

func getMyEntitlements(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/entitlements")
}

func listParentChildren(c *client.Client) ([]parentChildRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/parent/children", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Children []parentChildRow `json:"children"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Children, body, nil
}

func getParentStudentGrades(c *client.Client, childID string) ([]byte, error) {
	path := "/api/v1/parent/students/" + url.PathEscape(childID) + "/grades"
	return meGET(c, path)
}

func getParentStudentAttendance(c *client.Client, childID string) ([]byte, error) {
	path := "/api/v1/parent/students/" + url.PathEscape(childID) + "/attendance-summary"
	return meGET(c, path)
}

func parentLinkChild(c *client.Client, orgID, parentEmail, studentEmail string) ([]byte, error) {
	payload := map[string]any{
		"parentEmail":  parentEmail,
		"studentEmail": studentEmail,
	}
	b, _ := json.Marshal(payload)
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/parent-links"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func parentUnlinkChild(c *client.Client, orgID, linkID string) error {
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/parent-links/" + url.PathEscape(linkID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func revokeAllSessionsExceptCurrent(sessions []sessionRow, includeCurrent bool) []string {
	var ids []string
	for _, s := range sessions {
		if s.Current && !includeCurrent {
			continue
		}
		ids = append(ids, s.ID)
	}
	return ids
}

func sessionRevokeConfirmMessage(all, includeCurrent bool) string {
	if all && !includeCurrent {
		return "Revoke all sessions except the current one? Re-run with --yes to confirm."
	}
	if all && includeCurrent {
		return "Revoke ALL sessions including the current CLI session? Re-run with --yes --include-current to confirm."
	}
	return "Re-run with --yes to confirm session revoke."
}

func validateParentChildID(children []parentChildRow, childID string) error {
	for _, ch := range children {
		if ch.ID == childID {
			return nil
		}
	}
	return fmt.Errorf("child %q is not linked to your account", childID)
}