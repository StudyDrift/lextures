package cli

import "strings"

// RedactSecrets removes known secret fields from a map before JSON output.
func RedactSecrets(m map[string]any) {
	for k, v := range m {
		kl := strings.ToLower(k)
		switch kl {
		case "token", "secret", "clientsecret", "apikey", "password", "accesstoken", "refreshtoken":
			m[k] = "[redacted]"
		case "client_secret", "api_key", "access_token", "refresh_token":
			m[k] = "[redacted]"
		default:
			if sub, ok := v.(map[string]any); ok {
				RedactSecrets(sub)
			}
		}
	}
}

// RedactString returns a redacted placeholder when the key looks sensitive.
func RedactString(key, value string) string {
	kl := strings.ToLower(key)
	if strings.Contains(kl, "secret") || strings.Contains(kl, "token") || strings.Contains(kl, "password") {
		if value != "" {
			return "[redacted]"
		}
	}
	return value
}