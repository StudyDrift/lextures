package aiprovidercreds

import (
	"testing"

	"github.com/google/uuid"
)

func TestTenantBYOKPolicy_AllowTenantProvider(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		policy   TenantBYOKPolicy
		provider string
		want     bool
	}{
		{"default allow", TenantBYOKPolicy{Allowed: true}, "anthropic", true},
		{"disallowed", TenantBYOKPolicy{Allowed: false}, "anthropic", false},
		{"allowlist hit", TenantBYOKPolicy{Allowed: true, AllowedProviders: []string{"openai", "anthropic"}}, "anthropic", true},
		{"allowlist miss", TenantBYOKPolicy{Allowed: true, AllowedProviders: []string{"openai"}}, "anthropic", false},
		{"case insensitive", TenantBYOKPolicy{Allowed: true, AllowedProviders: []string{"OpenAI"}}, "openai", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.policy.AllowTenantProvider(tc.provider); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestValidateScope(t *testing.T) {
	t.Parallel()
	if err := validateScope(ScopePlatform, nil); err != nil {
		t.Fatal(err)
	}
	oid := uuid.New()
	if err := validateScope(ScopeOrg, &oid); err != nil {
		t.Fatal(err)
	}
	if err := validateScope(ScopePlatform, &oid); err == nil {
		t.Fatal("expected error for platform+org")
	}
	if err := validateScope(ScopeOrg, nil); err == nil {
		t.Fatal("expected error for org without id")
	}
	if err := validateScope("other", nil); err == nil {
		t.Fatal("expected error for invalid scope")
	}
}
