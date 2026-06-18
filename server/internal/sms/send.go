// Package sms sends transactional SMS messages via Twilio.
package sms

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/config"
)

const maxSMSBodyLen = 1600

// Configured reports whether Twilio credentials are present for live delivery.
func Configured(cfg config.Config) bool {
	return strings.TrimSpace(cfg.TwilioAccountSID) != "" &&
		strings.TrimSpace(cfg.TwilioAuthToken) != "" &&
		strings.TrimSpace(cfg.TwilioFromNumber) != ""
}

// BuildMessage composes a concise SMS body from notification fields.
func BuildMessage(title, body, actionURL string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	actionURL = strings.TrimSpace(actionURL)

	var parts []string
	if title != "" {
		parts = append(parts, title)
	}
	if body != "" {
		if len(parts) > 0 && !strings.HasPrefix(body, title) {
			parts = append(parts, body)
		} else if len(parts) == 0 {
			parts = append(parts, body)
		}
	}
	if actionURL != "" {
		parts = append(parts, actionURL)
	}
	text := strings.Join(parts, " — ")
	if len(text) > maxSMSBodyLen {
		text = text[:maxSMSBodyLen-3] + "..."
	}
	return text
}

// Send delivers an SMS via Twilio. When Twilio is not configured, logs the message and returns nil.
func Send(cfg config.Config, toPhone, body string) error {
	toPhone = strings.TrimSpace(toPhone)
	body = strings.TrimSpace(body)
	if toPhone == "" {
		return fmt.Errorf("sms: missing recipient phone number")
	}
	if body == "" {
		return fmt.Errorf("sms: empty message body")
	}

	sid := strings.TrimSpace(cfg.TwilioAccountSID)
	token := strings.TrimSpace(cfg.TwilioAuthToken)
	from := strings.TrimSpace(cfg.TwilioFromNumber)
	if sid == "" || token == "" || from == "" {
		log.Printf("sms: would send to %q body=%q (Twilio not configured)", toPhone, body)
		return nil
	}

	form := url.Values{}
	form.Set("To", toPhone)
	form.Set("From", from)
	form.Set("Body", body)

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", sid)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(sid+":"+token)))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("twilio returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}