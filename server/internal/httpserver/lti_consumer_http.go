package httpserver

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/lti"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	ltidb "github.com/lextures/lextures/server/internal/repos/lti"
)

type ltiLaunchBody struct {
	CourseID  string `json:"courseId"`
	ItemID    string `json:"itemId"`
	ModuleID  string `json:"moduleId"`
	DeepLink  bool   `json:"deepLink"`
	ReturnURL string `json:"returnUrl"`
	Locale    string `json:"locale"`
}

func (d Deps) handleLtiPlatformLaunch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireLtiHandler(w) {
			return
		}
		if d.Lti == nil || d.Lti.Keys == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		toolID, err := uuid.Parse(chi.URLParam(r, "registration_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid registration id.")
			return
		}
		tool, err := ltidb.GetExternalToolByID(r.Context(), d.Pool, toolID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Database error.")
			return
		}
		if tool == nil || !tool.Active {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "External tool not found.")
			return
		}

		var body ltiLaunchBody
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if len(b) > 0 {
			if err := json.Unmarshal(b, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		courseID := strings.TrimSpace(body.CourseID)
		if courseID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseId is required.")
			return
		}
		if _, err := uuid.Parse(courseID); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
			return
		}
		itemID := strings.TrimSpace(body.ItemID)
		moduleID := strings.TrimSpace(body.ModuleID)
		if body.DeepLink && strings.TrimSpace(body.ReturnURL) == "" {
			body.ReturnURL = strings.TrimRight(strings.TrimSpace(d.effectiveConfig().PublicWebOrigin), "/")
		}

		platformISS := d.Lti.APIBaseURL
		var targetLinkURI string
		if body.DeepLink {
			targetLinkURI = lti.DeepLinkReturnURL(platformISS)
		} else if itemID != "" {
			targetLinkURI = lti.ConsumerTargetURI(platformISS, courseID, itemID)
		} else {
			targetLinkURI = lti.PlatformAuthCallbackURL(platformISS)
		}

		loginHint, err := d.Lti.Keys.SignConsumerLoginHint(
			platformISS, tool.ToolIssuer, userID.String(), tool.ID,
			courseID, itemID, moduleID, strings.TrimSpace(body.ReturnURL), body.DeepLink,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not build login hint.")
			return
		}

		v := url.Values{}
		v.Set("iss", strings.TrimRight(strings.TrimSpace(platformISS), "/"))
		v.Set("login_hint", loginHint)
		v.Set("target_link_uri", targetLinkURI)
		v.Set("client_id", tool.ClientID)
		v.Set("lti_deployment_id", lti.DefaultDeploymentID)

		if body.DeepLink {
			dataJSON, _ := json.Marshal(lti.DeepLinkData{
				CourseID: courseID,
				ModuleID: moduleID,
				ToolID:   tool.ID,
			})
			msgHint, err := d.Lti.Keys.SignDeepLinkingMessageHint(
				platformISS, tool.ClientID, userID.String(),
				lti.DeepLinkReturnURL(platformISS), string(dataJSON),
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not build deep link hint.")
				return
			}
			v.Set("lti_message_hint", msgHint)
		}

		dest := strings.TrimRight(strings.TrimSpace(tool.ToolOidcAuthURL), "?")
		sep := "?"
		if strings.Contains(dest, "?") {
			sep = "&"
		}
		http.Redirect(w, r, dest+sep+v.Encode(), http.StatusFound)
	}
}

func (d Deps) handleLtiConsumerCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireLtiHandler(w) {
			return
		}
		if d.Lti == nil || d.Lti.Keys == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}

		q := r.URL.Query()
		redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
		clientID := strings.TrimSpace(q.Get("client_id"))
		loginHint := strings.TrimSpace(q.Get("login_hint"))
		state := strings.TrimSpace(q.Get("state"))
		nonce := strings.TrimSpace(q.Get("nonce"))
		if redirectURI == "" || clientID == "" || loginHint == "" || state == "" || nonce == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing required OIDC parameters.")
			return
		}
		if strings.TrimSpace(q.Get("response_type")) != "id_token" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported response_type.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}

		tool, err := d.findExternalToolByClientID(r, clientID)
		if err != nil || tool == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unknown tool registration.")
			return
		}

		hintClaims, err := lti.VerifyConsumerLoginHint(loginHint, d.Lti.Keys.PublicKey(), d.Lti.APIBaseURL, tool.ToolIssuer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid login_hint.")
			return
		}
		if hintClaims.ToolID != "" && hintClaims.ToolID != tool.ID {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "login_hint tool mismatch.")
			return
		}

		expT := time.Now().UTC().Add(5 * time.Minute)
		if hintClaims.ExpiresAt != nil {
			expT = hintClaims.ExpiresAt.UTC()
		}
		ok, err := ltidb.TryInsertConsumedNonce(r.Context(), d.Pool, nonce, expT)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Database error.")
			return
		}
		if !ok {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "nonce_already_used")
			return
		}

		var messageType string
		var deepLinkSettings map[string]any
		var targetLinkURI string
		if hintClaims.DeepLink {
			messageType = lti.MsgDeepLinkingRequest
			targetLinkURI = lti.DeepLinkReturnURL(d.Lti.APIBaseURL)
			dataJSON, _ := json.Marshal(lti.DeepLinkData{
				CourseID: hintClaims.CourseID,
				ModuleID: hintClaims.ModuleID,
				ToolID:   tool.ID,
			})
			deepLinkSettings = map[string]any{
				"deep_link_return_url": lti.DeepLinkReturnURL(d.Lti.APIBaseURL),
				"accept_types":         []string{"ltiResourceLink", "link"},
				"accept_multiple":      true,
				"data":                 string(dataJSON),
			}
		} else {
			messageType = lti.MsgResourceLinkRequest
			if hintClaims.ItemID != "" {
				targetLinkURI = lti.ConsumerTargetURI(d.Lti.APIBaseURL, hintClaims.CourseID, hintClaims.ItemID)
			} else {
				targetLinkURI = strings.TrimSpace(q.Get("target_link_uri"))
			}
		}

		resourceLinkID := hintClaims.ItemID
		if resourceLinkID == "" {
			resourceLinkID = uuid.NewString()
		}
		idClaims := lti.PlatformLaunchIDTokenClaims(
			d.Lti.APIBaseURL, tool.ClientID, hintClaims.Subject, nonce, targetLinkURI,
			messageType, hintClaims.CourseID, hintClaims.ItemID, resourceLinkID, "",
			deepLinkSettings, instructorRoles(), "en-US",
		)
		idToken, err := d.Lti.Keys.SignPlatformIDToken(idClaims)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not sign id_token.")
			return
		}

		writeLTIFormPost(w, redirectURI, idToken, state)
	}
}

func (d Deps) handleLtiDeepLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireLtiHandler(w) {
			return
		}
		if d.Lti == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}

		_ = r.ParseForm()
		jwtRaw := strings.TrimSpace(r.FormValue("JWT"))
		if jwtRaw == "" {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			var body struct {
				JWT string `json:"JWT"`
			}
			if len(b) > 0 && json.Unmarshal(b, &body) == nil {
				jwtRaw = strings.TrimSpace(body.JWT)
			}
		}
		if jwtRaw == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing JWT.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}

		payload, err := lti.DecodeJWTPayloadJSON(jwtRaw)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JWT.")
			return
		}
		toolISS, _ := payload["iss"].(string)
		toolClientID := firstAudString(payload["aud"])
		if toolClientID == "" {
			toolClientID, _ = payload["client_id"].(string)
		}
		if toolISS == "" || toolClientID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "JWT missing iss or aud.")
			return
		}

		tool, err := d.findExternalToolByIssuerClient(r, toolISS, toolClientID)
		if err != nil || tool == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Unknown tool.")
			return
		}
		pub, err := lti.PublicKeyForJWT(tool.ToolJWKSURL, jwtRaw)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not load tool JWKS.")
			return
		}
		platformISS := strings.TrimRight(strings.TrimSpace(d.Lti.APIBaseURL), "/")
		claims, err := lti.VerifyToolMessageJWT(jwtRaw, pub, platformISS, tool.ClientID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid deep linking JWT: "+err.Error())
			return
		}
		items, err := lti.ParseDeepLinkingContentItems(claims)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		var dlData lti.DeepLinkData
		if dataRaw, ok := claims[lti.ClaimDLData].(string); ok {
			dlData, _ = lti.ParseDeepLinkDataJSON(dataRaw)
		} else if settings, ok := claims[lti.ClaimDLSettings].(map[string]any); ok {
			if dataStr, ok := settings["data"].(string); ok {
				dlData, _ = lti.ParseDeepLinkDataJSON(dataStr)
			}
		}
		if dlData.ToolID == "" {
			dlData.ToolID = tool.ID
		}

		created := make([]map[string]any, 0)
		if dlData.CourseID != "" && dlData.ModuleID != "" {
			cid, err := uuid.Parse(dlData.CourseID)
			if err == nil {
				mid, err := uuid.Parse(dlData.ModuleID)
				if err == nil {
					toolUUID, err := uuid.Parse(dlData.ToolID)
					if err == nil {
						for _, item := range items {
							createdItem, err := d.createModuleItemFromDeepLinkContent(r, cid, mid, toolUUID, item)
							if err == nil && createdItem != nil {
								created = append(created, createdItem)
							}
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"contentItems": items,
			"createdItems": created,
		})
	}
}

func (d Deps) handleLtiConsumerTarget() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !d.requireLtiHandler(w) {
			return
		}
		courseID := strings.TrimSpace(r.URL.Query().Get("courseId"))
		itemID := strings.TrimSpace(r.URL.Query().Get("itemId"))
		public := strings.TrimRight(strings.TrimSpace(d.effectiveConfig().PublicWebOrigin), "/")
		var dest string
		if courseID != "" && itemID != "" && d.Pool != nil {
			if cid, err := uuid.Parse(courseID); err == nil {
				if cc, err := course.GetCourseCodeByID(r.Context(), d.Pool, cid); err == nil && cc != nil {
					dest = fmt.Sprintf("%s/courses/%s/modules/lti/%s", public, url.PathEscape(*cc), url.PathEscape(itemID))
				}
			}
		}
		if dest == "" {
			dest = public + "/"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"/><title>LTI launch</title>
<meta http-equiv="refresh" content="0;url=%s"/></head>
<body><p>Continuing to your course… <a href="%s">Click here</a> if you are not redirected.</p></body></html>`,
			html.EscapeString(dest), html.EscapeString(dest))))
	}
}

func (d Deps) findExternalToolByClientID(r *http.Request, clientID string) (*ltidb.ExternalTool, error) {
	tools, err := ltidb.ListExternalToolsForScores(r.Context(), d.Pool)
	if err != nil {
		return nil, err
	}
	for i := range tools {
		if tools[i].Active && tools[i].ClientID == clientID {
			return &tools[i], nil
		}
	}
	return nil, nil
}

func (d Deps) findExternalToolByIssuerClient(r *http.Request, iss, clientID string) (*ltidb.ExternalTool, error) {
	tools, err := ltidb.ListExternalToolsForScores(r.Context(), d.Pool)
	if err != nil {
		return nil, err
	}
	for i := range tools {
		if tools[i].Active && tools[i].ToolIssuer == iss && tools[i].ClientID == clientID {
			return &tools[i], nil
		}
	}
	return nil, nil
}

func (d Deps) createModuleItemFromDeepLinkContent(
	r *http.Request,
	courseID, moduleID, toolID uuid.UUID,
	item map[string]any,
) (map[string]any, error) {
	itemType, _ := item["type"].(string)
	title, _ := item["title"].(string)
	if strings.TrimSpace(title) == "" {
		title = "LTI content"
	}
	switch strings.TrimSpace(itemType) {
	case "ltiResourceLink":
		urlStr, _ := item["url"].(string)
		custom, _ := item["custom"].(map[string]any)
		var resourceLinkID string
		if custom != nil {
			if v, ok := custom["resource_link_id"].(string); ok {
				resourceLinkID = v
			}
		}
		if resourceLinkID == "" {
			if v, ok := item["url"].(string); ok {
				resourceLinkID = v
			}
		}
		var lineItem *string
		if v, ok := item["lineItem"].(map[string]any); ok {
			if id, ok := v["id"].(string); ok && id != "" {
				lineItem = &id
			}
		}
		row, err := coursestructure.InsertLTILinkUnderModule(
			r.Context(), d.Pool, courseID, moduleID, toolID, title, resourceLinkID, lineItem,
		)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"itemId": row.ID.String(),
			"kind":   "lti_link",
			"title":  title,
			"url":    urlStr,
		}, nil
	case "link":
		urlStr, _ := item["url"].(string)
		if strings.TrimSpace(urlStr) == "" {
			return nil, fmt.Errorf("link missing url")
		}
		row, err := coursestructure.InsertExternalLinkUnderModule(r.Context(), d.Pool, courseID, moduleID, title, urlStr)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"itemId": row.ID.String(),
			"kind":   "external_link",
			"title":  title,
			"url":    urlStr,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported content type %q", itemType)
	}
}

func instructorRoles() []string {
	return []string{
		"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Instructor",
		"http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor",
	}
}

func writeLTIFormPost(w http.ResponseWriter, action, idToken, state string) {
	actionEsc := html.EscapeString(action)
	idEsc := html.EscapeString(idToken)
	stateEsc := html.EscapeString(state)
	h := fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"/><title>LTI launch</title></head>
<body><p>Launching tool…</p>
<form id="lti" method="post" action="%s">
  <input type="hidden" name="id_token" value="%s" />
  <input type="hidden" name="state" value="%s" />
</form>
<script>document.getElementById('lti').submit();</script>
</body></html>`, actionEsc, idEsc, stateEsc)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h))
}