package push

import "encoding/json"

// NativePayload is the data envelope shared by APNs and FCM for deep linking.
type NativePayload struct {
	Title            string `json:"title"`
	Body             string `json:"body"`
	ActionURL        string `json:"action_url,omitempty"`
	NotificationType string `json:"notification_type,omitempty"`
}

// BuildNativePayloadJSON returns a JSON object for custom data fields.
func BuildNativePayloadJSON(title, body, actionURL, notificationType string) []byte {
	p := NativePayload{
		Title:            title,
		Body:             body,
		ActionURL:        actionURL,
		NotificationType: notificationType,
	}
	b, _ := json.Marshal(p)
	return b
}
