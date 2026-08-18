package marketingcontent

import "encoding/json"

const (
	extraSocialTitleKey       = "socialTitle"
	extraSocialDescriptionKey = "socialDescription"
)

// SocialFromExtra reads optional Open Graph / Twitter overrides stored in extra.
func SocialFromExtra(extra json.RawMessage) (title, description string) {
	var payload map[string]any
	if len(extra) == 0 || !json.Valid(extra) {
		return "", ""
	}
	if err := json.Unmarshal(extra, &payload); err != nil || payload == nil {
		return "", ""
	}
	return extraString(payload[extraSocialTitleKey]), extraString(payload[extraSocialDescriptionKey])
}

// MergeSocialIntoExtra writes social title/description into extra without dropping other keys.
func MergeSocialIntoExtra(extra json.RawMessage, title, description string) json.RawMessage {
	var payload map[string]any
	if len(extra) > 0 && json.Valid(extra) {
		_ = json.Unmarshal(extra, &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	setExtraString(payload, extraSocialTitleKey, title)
	setExtraString(payload, extraSocialDescriptionKey, description)
	out, err := json.Marshal(payload)
	if err != nil {
		if len(extra) == 0 {
			return json.RawMessage(`{}`)
		}
		return extra
	}
	return out
}

func applySocialFromExtra(a *Article) {
	if a == nil {
		return
	}
	a.SocialTitle, a.SocialDescription = SocialFromExtra(a.Extra)
}

func extraString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func setExtraString(payload map[string]any, key, value string) {
	if value == "" {
		delete(payload, key)
		return
	}
	payload[key] = value
}
