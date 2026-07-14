package mail

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"github.com/lextures/lextures/server/internal/config"
)

// sesProvider delivers mail via Amazon SES (API v2).
// Credentials use the default AWS chain (env, shared config, IAM role) unless
// SES_ACCESS_KEY_ID / SES_SECRET_ACCESS_KEY are set.
type sesProvider struct{}

func (sesProvider) Name() string { return ProviderSES }

func (sesProvider) Configured(cfg config.Config) bool {
	return sesFromAddress(cfg) != ""
}

func (sesProvider) Send(ctx context.Context, cfg config.Config, msg Message) error {
	from := sesFromAddress(cfg)
	if from == "" {
		return fmt.Errorf("SES_FROM or SMTP_FROM is required when using the SES email provider")
	}
	fromAddr, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("parse from address: %w", err)
	}
	if strings.TrimSpace(msg.FromDisplayName) != "" {
		fromAddr.Name = strings.TrimSpace(msg.FromDisplayName)
	}

	client, err := newSESClient(ctx, cfg)
	if err != nil {
		return err
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromAddr.String()),
		Destination: &types.Destination{
			ToAddresses: []string{msg.To},
		},
	}
	if cs := strings.TrimSpace(cfg.SESConfigurationSet); cs != "" {
		input.ConfigurationSetName = aws.String(cs)
	}

	// ICS attachments require raw MIME; simple path for plain/HTML.
	if strings.TrimSpace(msg.ICSContent) != "" {
		raw, buildErr := buildMIMEWithICS(fromAddr.String(), msg)
		if buildErr != nil {
			return buildErr
		}
		input.Content = &types.EmailContent{
			Raw: &types.RawMessage{Data: raw},
		}
	} else if msg.HTMLBody != "" {
		input.Content = &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(msg.BodyText), Charset: aws.String("UTF-8")},
					Html: &types.Content{Data: aws.String(msg.HTMLBody), Charset: aws.String("UTF-8")},
				},
			},
		}
	} else {
		input.Content = &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(msg.BodyText), Charset: aws.String("UTF-8")},
				},
			},
		}
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = client.SendEmail(sendCtx, input)
	return err
}

func sesFromAddress(cfg config.Config) string {
	if v := strings.TrimSpace(cfg.SESFrom); v != "" {
		return v
	}
	return strings.TrimSpace(cfg.SMTPFrom)
}

func sesRegion(cfg config.Config) string {
	if v := strings.TrimSpace(cfg.SESRegion); v != "" {
		return v
	}
	// Reuse storage/AWS region when SES-specific region is unset.
	if v := strings.TrimSpace(cfg.StorageRegion); v != "" {
		return v
	}
	return "us-east-1"
}

func newSESClient(ctx context.Context, cfg config.Config) (*sesv2.Client, error) {
	region := sesRegion(cfg)
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(region))

	ak := strings.TrimSpace(cfg.SESAccessKeyID)
	sk := strings.TrimSpace(cfg.SESSecretAccessKey)
	if ak != "" && sk != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(ak, sk, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return sesv2.NewFromConfig(awsCfg), nil
}
