package httpserver

import (
	"net/url"
	"strings"
)

func marketingPreviewURL(origin, articlePath, token string) string {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		origin = "https://lextures.com"
	}
	path := strings.TrimSpace(articlePath)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return origin + path + "?preview_token=" + url.QueryEscape(token)
}
