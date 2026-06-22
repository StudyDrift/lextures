package lti

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LTI Advantage claim URIs (platform consumer role).
const (
	ClaimMessageType        = "https://purl.imsglobal.org/spec/lti/claim/message_type"
	ClaimVersion            = "https://purl.imsglobal.org/spec/lti/claim/version"
	ClaimDeploymentID       = "https://purl.imsglobal.org/spec/lti/claim/deployment_id"
	ClaimTargetLinkURI      = "https://purl.imsglobal.org/spec/lti/claim/target_link_uri"
	ClaimRoles              = "https://purl.imsglobal.org/spec/lti/claim/roles"
	ClaimContext            = "https://purl.imsglobal.org/spec/lti/claim/context"
	ClaimResourceLink       = "https://purl.imsglobal.org/spec/lti/claim/resource_link"
	ClaimCustom             = "https://purl.imsglobal.org/spec/lti/claim/custom"
	ClaimLaunchPresentation = "https://purl.imsglobal.org/spec/lti/claim/launch_presentation"
	ClaimDLSettings         = "https://purl.imsglobal.org/spec/lti-dl/claim/deep_linking_settings"
	ClaimDLContentItems     = "https://purl.imsglobal.org/spec/lti-dl/claim/content_items"
	ClaimDLData             = "https://purl.imsglobal.org/spec/lti-dl/claim/data"

	MsgResourceLinkRequest  = "LtiResourceLinkRequest"
	MsgDeepLinkingRequest   = "LtiDeepLinkingRequest"
	MsgDeepLinkingResponse  = "LtiDeepLinkingResponse"
	LTIVersion              = "1.3.0"
	DefaultDeploymentID     = "1"
)

// ConsumerLoginHint is the signed login_hint JWT payload for platform→tool OIDC login.
type ConsumerLoginHint struct {
	jwt.RegisteredClaims
	CourseID  string `json:"courseId,omitempty"`
	ItemID    string `json:"itemId,omitempty"`
	ModuleID  string `json:"moduleId,omitempty"`
	DeepLink  bool   `json:"deepLink,omitempty"`
	ReturnURL string `json:"returnUrl,omitempty"`
	ToolID    string `json:"toolId"`
}

// SignConsumerLoginHint builds a short-lived login_hint JWT for OIDC login initiation.
func (k *RsaKeyPair) SignConsumerLoginHint(platformISS, toolIssuer, userID, toolID string, courseID, itemID, moduleID, returnURL string, deepLink bool) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(15 * time.Minute)
	iss := strings.TrimRight(strings.TrimSpace(platformISS), "/")
	claims := ConsumerLoginHint{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{toolIssuer},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		CourseID:  courseID,
		ItemID:    itemID,
		ModuleID:  moduleID,
		DeepLink:  deepLink,
		ReturnURL: returnURL,
		ToolID:    toolID,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = k.kid
	return tok.SignedString(k.private)
}

// VerifyConsumerLoginHint validates a platform-issued login_hint JWT.
func VerifyConsumerLoginHint(token string, pub *rsa.PublicKey, platformISS, toolIssuer string) (*ConsumerLoginHint, error) {
	if pub == nil {
		return nil, errors.New("lti: nil public key")
	}
	iss := strings.TrimRight(strings.TrimSpace(platformISS), "/")
	claims := &ConsumerLoginHint{}
	p := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithLeeway(30*time.Second),
	)
	_, err := p.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return nil, fmt.Errorf("lti: invalid login_hint: %w", err)
	}
	if claims.Issuer != iss {
		return nil, errors.New("lti: login_hint iss mismatch")
	}
	if !claimStringsMatch(claims.Audience, toolIssuer) {
		return nil, errors.New("lti: login_hint aud mismatch")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.New("lti: login_hint sub required")
	}
	return claims, nil
}

// SignDeepLinkingMessageHint builds lti_message_hint for a deep linking request launch.
func (k *RsaKeyPair) SignDeepLinkingMessageHint(platformISS, toolClientID, userID, deepLinkReturnURL, dataJSON string) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(15 * time.Minute)
	iss := strings.TrimRight(strings.TrimSpace(platformISS), "/")
	settings := map[string]any{
		"deep_link_return_url": deepLinkReturnURL,
		"accept_types":         []string{"ltiResourceLink", "link"},
		"accept_multiple":      true,
	}
	if strings.TrimSpace(dataJSON) != "" {
		settings["data"] = dataJSON
	}
	claims := jwt.MapClaims{
		"iss":   iss,
		"aud":   toolClientID,
		"sub":   userID,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		ClaimMessageType: MsgDeepLinkingRequest,
		ClaimVersion:     LTIVersion,
		ClaimDeploymentID: DefaultDeploymentID,
		ClaimDLSettings:  settings,
	}
	return k.SignRS256HintJWT(claims)
}

// PlatformLaunchIDTokenClaims builds an LTI id_token for platform→tool OIDC authentication response.
func PlatformLaunchIDTokenClaims(
	platformISS, toolClientID, userID, nonce, targetLinkURI string,
	messageType string,
	courseID, itemID, resourceLinkID, title string,
	deepLinkSettings map[string]any,
	roles []string,
	locale string,
) map[string]any {
	now := time.Now().UTC()
	exp := now.Add(5 * time.Minute)
	iss := strings.TrimRight(strings.TrimSpace(platformISS), "/")
	if locale == "" {
		locale = "en-US"
	}
	if len(roles) == 0 {
		roles = []string{"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Instructor"}
	}
	claims := map[string]any{
		"iss":   iss,
		"aud":   toolClientID,
		"sub":   userID,
		"nonce": nonce,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		ClaimMessageType:  messageType,
		ClaimVersion:      LTIVersion,
		ClaimDeploymentID: DefaultDeploymentID,
		ClaimTargetLinkURI: targetLinkURI,
		ClaimRoles:        roles,
		ClaimLaunchPresentation: map[string]any{"locale": locale},
	}
	if strings.TrimSpace(courseID) != "" {
		claims[ClaimContext] = map[string]any{
			"id":    courseID,
			"type":  []string{"http://purl.imsglobal.org/vocab/lis/v2/course#CourseOffering"},
			"label": courseID,
		}
	}
	if messageType == MsgResourceLinkRequest && strings.TrimSpace(itemID) != "" {
		rl := map[string]any{"id": resourceLinkID}
		if title != "" {
			rl["title"] = title
		}
		claims[ClaimResourceLink] = rl
		claims[ClaimCustom] = map[string]any{
			"courseId":        courseID,
			"structureItemId": itemID,
		}
	}
	if messageType == MsgDeepLinkingRequest && deepLinkSettings != nil {
		claims[ClaimDLSettings] = deepLinkSettings
	}
	return claims
}

// SignPlatformIDToken signs an LTI platform id_token JWT.
func (k *RsaKeyPair) SignPlatformIDToken(claims map[string]any) (string, error) {
	return k.SignRS256HintJWT(claims)
}

// VerifyToolMessageJWT verifies a JWT from an external tool (deep linking response, etc.).
func VerifyToolMessageJWT(token string, pub *rsa.PublicKey, platformISS, toolClientID string) (map[string]any, error) {
	if pub == nil {
		return nil, errors.New("lti: nil public key")
	}
	claims := jwt.MapClaims{}
	p := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithLeeway(60 * time.Second),
	)
	_, err := p.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return nil, fmt.Errorf("lti: invalid tool JWT: %w", err)
	}
	iss, _ := claims["iss"].(string)
	if iss == "" {
		return nil, errors.New("lti: iss required")
	}
	platformISS = strings.TrimRight(strings.TrimSpace(platformISS), "/")
	if !audienceContains(claims["aud"], platformISS) && !audienceContains(claims["aud"], toolClientID) {
		if cid, _ := claims["client_id"].(string); cid != platformISS && cid != toolClientID {
			return nil, errors.New("lti: aud mismatch")
		}
	}
	now := float64(time.Now().UTC().Unix())
	if exp, ok := claims["exp"].(float64); ok && now > exp+60 {
		return nil, errors.New("lti: token expired")
	}
	return claims, nil
}

// ParseDeepLinkingContentItems extracts content_items from an LtiDeepLinkingResponse JWT.
func ParseDeepLinkingContentItems(claims map[string]any) ([]map[string]any, error) {
	msg, _ := claims[ClaimMessageType].(string)
	if msg != MsgDeepLinkingResponse {
		return nil, fmt.Errorf("lti: expected %s, got %q", MsgDeepLinkingResponse, msg)
	}
	raw, ok := claims[ClaimDLContentItems]
	if !ok {
		return nil, errors.New("lti: missing content_items claim")
	}
	switch items := raw.(type) {
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, m)
		}
		if len(out) == 0 {
			return nil, errors.New("lti: no content items")
		}
		return out, nil
	default:
		return nil, errors.New("lti: invalid content_items claim")
	}
}

// DeepLinkData decodes the opaque data field from deep linking settings.
type DeepLinkData struct {
	CourseID string `json:"courseId"`
	ModuleID string `json:"moduleId"`
	ToolID   string `json:"toolId"`
}

// ParseDeepLinkDataJSON parses the data JSON blob from deep linking settings/response.
func ParseDeepLinkDataJSON(data string) (DeepLinkData, error) {
	var d DeepLinkData
	if strings.TrimSpace(data) == "" {
		return d, nil
	}
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return d, err
	}
	return d, nil
}

// ConsumerTargetURI builds the platform target_link_uri for resource link launches.
func ConsumerTargetURI(platformBase, courseID, itemID string) string {
	iss := strings.TrimRight(strings.TrimSpace(platformBase), "/")
	return fmt.Sprintf("%s/api/v1/lti/consumer/target?courseId=%s&itemId=%s", iss, courseID, itemID)
}

// PlatformAuthCallbackURL returns the OIDC authentication endpoint URL for this platform.
func PlatformAuthCallbackURL(platformBase string) string {
	iss := strings.TrimRight(strings.TrimSpace(platformBase), "/")
	return iss + "/api/v1/lti/callback"
}

// DeepLinkReturnURL returns the deep linking response handler URL.
func DeepLinkReturnURL(platformBase string) string {
	iss := strings.TrimRight(strings.TrimSpace(platformBase), "/")
	return iss + "/api/v1/lti/deep-link"
}