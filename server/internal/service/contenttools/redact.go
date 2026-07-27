package contenttools

import (
	"encoding/json"
)

const sensitiveAnnotation = "x-lex-sensitive"

// RedactSensitiveConfig strips config fields marked "x-lex-sensitive": true in
// the manifest's configSchema (FR-10), including nested object/array properties.
// The framework performs stripping; tools must not be trusted to do it.
func RedactSensitiveConfig(configSchema json.RawMessage, config json.RawMessage) (json.RawMessage, error) {
	if len(config) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var cfg any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}
	var schema any
	if len(configSchema) > 0 {
		if err := json.Unmarshal(configSchema, &schema); err != nil {
			return nil, err
		}
	}
	redactBySchema(schema, cfg)
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

// redactBySchema walks schema + value together and deletes sensitive properties.
func redactBySchema(schema, value any) {
	sm, ok := schema.(map[string]any)
	if !ok || value == nil {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		if props, ok := sm["properties"].(map[string]any); ok {
			for name, propSchema := range props {
				pm, ok := propSchema.(map[string]any)
				if !ok {
					continue
				}
				if flagged, _ := pm[sensitiveAnnotation].(bool); flagged {
					delete(typed, name)
					continue
				}
				if child, exists := typed[name]; exists {
					redactBySchema(pm, child)
				}
			}
		}
		// additionalProperties schema (object maps)
		if add, ok := sm["additionalProperties"].(map[string]any); ok {
			for _, child := range typed {
				redactBySchema(add, child)
			}
		}
	case []any:
		if items, ok := sm["items"].(map[string]any); ok {
			for _, child := range typed {
				redactBySchema(items, child)
			}
		}
	}
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
