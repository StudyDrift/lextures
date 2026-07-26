package contenttools

import (
	"encoding/json"
)

const sensitiveAnnotation = "x-lex-sensitive"

// RedactSensitiveConfig strips config fields marked "x-lex-sensitive": true in
// the manifest's configSchema (FR-10). The framework performs stripping; tools
// must not be trusted to do it.
func RedactSensitiveConfig(configSchema json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	if len(config) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}
	sensitive := sensitiveKeys(configSchema)
	if len(sensitive) == 0 {
		return config, nil
	}
	for k := range sensitive {
		delete(cfg, k)
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StripSensitiveSchemaAnnotations removes x-lex-sensitive keys from a schema
// document before serving manifests to clients (FR-15).
func StripSensitiveSchemaAnnotations(schema json.RawMessage) (json.RawMessage, error) {
	if len(schema) == 0 {
		return schema, nil
	}
	var doc any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, err
	}
	stripAnnotations(doc)
	return json.Marshal(doc)
}

func sensitiveKeys(configSchema json.RawMessage) map[string]struct{} {
	out := map[string]struct{}{}
	if len(configSchema) == 0 {
		return out
	}
	var doc map[string]any
	if err := json.Unmarshal(configSchema, &doc); err != nil {
		return out
	}
	props, _ := doc["properties"].(map[string]any)
	for name, prop := range props {
		pm, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		if flagged, _ := pm[sensitiveAnnotation].(bool); flagged {
			out[name] = struct{}{}
		}
	}
	return out
}

func stripAnnotations(v any) {
	switch t := v.(type) {
	case map[string]any:
		delete(t, sensitiveAnnotation)
		for _, child := range t {
			stripAnnotations(child)
		}
	case []any:
		for _, child := range t {
			stripAnnotations(child)
		}
	}
}
