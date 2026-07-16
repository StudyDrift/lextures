package boardfilter

import "encoding/json"

func jsonUnmarshalImpl(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
