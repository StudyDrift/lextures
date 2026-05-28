package config

import "testing"

func TestValidate_PIIRedactionBlockedInProduction(t *testing.T) {
	c := Config{
		DatabaseURL:         "postgres://a:b@localhost:5432/db",
		JWTSecret:           validTestJWT,
		DisablePIIRedaction: true,
		AppEnv:              "production",
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error when disabling PII redaction in production")
	}
	if err.Error() != "PII redaction cannot be disabled in production" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_PIIRedactionAllowedInLocal(t *testing.T) {
	c := Config{
		DatabaseURL:         "postgres://a:b@localhost:5432/db",
		JWTSecret:           validTestJWT,
		DisablePIIRedaction: true,
		AppEnv:              "local",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_PIIRedactionBlockedInStaging(t *testing.T) {
	c := Config{
		DatabaseURL:         "postgres://a:b@localhost:5432/db",
		JWTSecret:           validTestJWT,
		DisablePIIRedaction: true,
		AppEnv:              "staging",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for staging")
	}
}
