package integrations

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // LTI 1.1 mandates HMAC-SHA1 per the IMS spec.
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LTILaunch describes a configured LTI 1.1 external-tool embed (Edpuzzle, Khan
// Academy, Quizlet, etc.) — plan 16.4 FR-6.
type LTILaunch struct {
	LaunchURL      string
	ConsumerKey    string
	ConsumerSecret string
	// ResourceLinkID uniquely identifies the placement (module item).
	ResourceLinkID string
	// ContextID/Title identify the Lextures course for the tool.
	ContextID    string
	ContextTitle string
	// User identity passed to the tool.
	UserID    string
	Roles     string // e.g. "Learner" or "Instructor"
	UserEmail string
	UserName  string
}

// GenerateLTILaunchParams returns the signed POST body for an LTI 1.1 launch.
// It implements the OAuth 1.0 HMAC-SHA1 body signature required by the IMS LTI
// 1.1 specification. now and nonce are injectable for deterministic tests.
func GenerateLTILaunchParams(l LTILaunch, now time.Time, nonce string) (map[string]string, error) {
	if l.LaunchURL == "" || l.ConsumerKey == "" || l.ConsumerSecret == "" {
		return nil, fmt.Errorf("integrations: lti launch requires url, key, and secret")
	}
	if nonce == "" {
		nonce = randomNonce()
	}
	roles := l.Roles
	if roles == "" {
		roles = "Learner"
	}
	params := map[string]string{
		"lti_message_type":                 "basic-lti-launch-request",
		"lti_version":                      "LTI-1p0",
		"resource_link_id":                 l.ResourceLinkID,
		"context_id":                       l.ContextID,
		"context_title":                    l.ContextTitle,
		"user_id":                          l.UserID,
		"roles":                            roles,
		"lis_person_contact_email_primary": l.UserEmail,
		"lis_person_name_full":             l.UserName,
		"oauth_consumer_key":               l.ConsumerKey,
		"oauth_nonce":                      nonce,
		"oauth_signature_method":           "HMAC-SHA1",
		"oauth_timestamp":                  strconv.FormatInt(now.Unix(), 10),
		"oauth_version":                    "1.0",
		"oauth_callback":                   "about:blank",
	}
	// Drop empties so we don't sign blank optional fields.
	for k, v := range params {
		if v == "" {
			delete(params, k)
		}
	}
	sig := signLTI("POST", l.LaunchURL, params, l.ConsumerSecret)
	params["oauth_signature"] = sig
	return params, nil
}

// signLTI computes the OAuth 1.0 HMAC-SHA1 signature over the request.
func signLTI(method, rawURL string, params map[string]string, consumerSecret string) string {
	// Build the normalized parameter string (sorted, percent-encoded).
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, oauthEscape(k)+"="+oauthEscape(params[k]))
	}
	paramString := strings.Join(pairs, "&")

	base := strings.ToUpper(method) + "&" + oauthEscape(normalizeURL(rawURL)) + "&" + oauthEscape(paramString)
	// LTI 1.1: token secret is empty, so the signing key is "secret&".
	signingKey := oauthEscape(consumerSecret) + "&"
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// normalizeURL lower-cases scheme/host and strips the query/default ports per
// the OAuth 1.0 base-string URL normalization rules.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	if (scheme == "http" && strings.HasSuffix(host, ":80")) ||
		(scheme == "https" && strings.HasSuffix(host, ":443")) {
		host = host[:strings.LastIndex(host, ":")]
	}
	return scheme + "://" + host + u.EscapedPath()
}

// oauthEscape implements RFC 3986 percent-encoding as required by OAuth 1.0.
func oauthEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}
