package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/lextures/lextures/server/internal/config"
)

// smtpProvider delivers mail over SMTP (SendGrid, Mailgun SMTP, Amazon SES SMTP
// interface, self-hosted, etc.).
type smtpProvider struct{}

func (smtpProvider) Name() string { return ProviderSMTP }

func (smtpProvider) Configured(cfg config.Config) bool {
	return strings.TrimSpace(cfg.SMTPHost) != ""
}

func (smtpProvider) Send(_ context.Context, cfg config.Config, msg Message) error {
	host := strings.TrimSpace(cfg.SMTPHost)
	from := strings.TrimSpace(cfg.SMTPFrom)
	if from == "" {
		return fmt.Errorf("SMTP_FROM is required when SMTP_HOST is set")
	}
	fromAddr, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("parse SMTP_FROM: %w", err)
	}
	if strings.TrimSpace(msg.FromDisplayName) != "" {
		fromAddr.Name = strings.TrimSpace(msg.FromDisplayName)
	}

	var data []byte
	if strings.TrimSpace(msg.ICSContent) != "" {
		data, err = buildMIMEWithICS(fromAddr.String(), msg)
		if err != nil {
			return err
		}
	} else if strings.TrimSpace(msg.HTMLBody) == "" {
		data = buildMIMEPlain(fromAddr.String(), msg)
	} else {
		data = buildMIMEAlternative(fromAddr.String(), msg)
	}

	addr := fmt.Sprintf("%s:%d", host, cfg.SMTPPort)
	if cfg.SMTPUser != "" && cfg.SMTPPassword != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, host)
		return smtp.SendMail(addr, auth, fromAddr.Address, []string{msg.To}, data)
	}
	return smtp.SendMail(addr, nil, fromAddr.Address, []string{msg.To}, data)
}

func buildMIMEAlternative(fromHeader string, msg Message) []byte {
	boundary := "lextures-boundary-7bit"
	parts := []string{
		"To: " + msg.To,
		"From: " + fromHeader,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=utf-8",
		"",
		msg.BodyText,
		"",
		"--" + boundary,
		"Content-Type: text/html; charset=utf-8",
		"",
		msg.HTMLBody,
		"",
		"--" + boundary + "--",
	}
	return []byte(strings.Join(parts, "\r\n"))
}

func buildMIMEWithICS(fromHeader string, msg Message) ([]byte, error) {
	icsFilename := msg.ICSFilename
	if icsFilename == "" {
		icsFilename = "event.ics"
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=utf-8")
	tw, err := w.CreatePart(th)
	if err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(msg.BodyText)); err != nil {
		return nil, err
	}

	hh := make(textproto.MIMEHeader)
	hh.Set("Content-Type", "text/html; charset=utf-8")
	hw, err := w.CreatePart(hh)
	if err != nil {
		return nil, err
	}
	if _, err := hw.Write([]byte(msg.HTMLBody)); err != nil {
		return nil, err
	}

	ah := make(textproto.MIMEHeader)
	ah.Set("Content-Type", "text/calendar; charset=utf-8; method=PUBLISH")
	ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, icsFilename))
	ah.Set("Content-Transfer-Encoding", "base64")
	aw, err := w.CreatePart(ah)
	if err != nil {
		return nil, err
	}
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(msg.ICSContent)))
	base64.StdEncoding.Encode(enc, []byte(msg.ICSContent))
	if _, err := aw.Write(enc); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	headers := []string{
		"To: " + msg.To,
		"From: " + fromHeader,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=" + w.Boundary(),
		"",
		body.String(),
	}
	return []byte(strings.Join(headers, "\r\n")), nil
}

// buildMIMEPlain builds a simple text/plain message (magic link, etc.).
func buildMIMEPlain(fromHeader string, msg Message) []byte {
	parts := []string{
		"To: " + msg.To,
		"From: " + fromHeader,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		msg.BodyText,
	}
	return []byte(strings.Join(parts, "\r\n"))
}
