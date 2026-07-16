package board

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/webhooks"
	"golang.org/x/net/html"
)

const (
	unfurlMaxRedirects = 3
	unfurlMaxBodyBytes = 512 * 1024
	unfurlTimeout      = 5 * time.Second
)

// LinkPreview is the cached unfurl payload stored on link posts.
type LinkPreview struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	SiteName    string `json:"siteName,omitempty"`
	FetchedAt   string `json:"fetchedAt,omitempty"`
}

// ErrUnfurlSSRF is returned when a URL targets a private/blocked address.
var ErrUnfurlSSRF = fmt.Errorf("board: link preview blocked by SSRF policy")

// ValidateUnfurlURL checks scheme and resolves host against private ranges.
func ValidateUnfurlURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("board: url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("board: invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("board: url must be http or https")
	}
	if u.User != nil {
		return ErrUnfurlSSRF
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("board: url must include a hostname")
	}
	if strings.EqualFold(host, "localhost") {
		return ErrUnfurlSSRF
	}
	if ip := net.ParseIP(host); ip != nil {
		if webhooks.BlockedIP(ip) {
			return ErrUnfurlSSRF
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("board: could not resolve hostname: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("board: hostname did not resolve")
	}
	for _, ip := range ips {
		if webhooks.BlockedIP(ip) {
			return ErrUnfurlSSRF
		}
	}
	return nil
}

// FetchLinkPreview performs an SSRF-safe GET and parses Open Graph / basic meta tags.
// On soft failures (network, parse), returns a bare preview with only the URL context.
func FetchLinkPreview(ctx context.Context, rawURL string) (*LinkPreview, error) {
	if err := ValidateUnfurlURL(rawURL); err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: unfurlTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= unfurlMaxRedirects {
				return fmt.Errorf("board: too many redirects")
			}
			if err := ValidateUnfurlURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(rawURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LexturesLinkPreview/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return &LinkPreview{FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &LinkPreview{FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	limited := io.LimitReader(resp.Body, unfurlMaxBodyBytes)
	doc, err := html.Parse(limited)
	if err != nil {
		return &LinkPreview{FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	preview := parseOGMeta(doc)
	preview.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	if preview.Image != "" {
		preview.Image = resolveURL(rawURL, preview.Image)
	}
	return preview, nil
}

func parseOGMeta(n *html.Node) *LinkPreview {
	out := &LinkPreview{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "meta":
				prop, name, content := "", "", ""
				for _, a := range node.Attr {
					switch strings.ToLower(a.Key) {
					case "property":
						prop = strings.ToLower(a.Val)
					case "name":
						name = strings.ToLower(a.Val)
					case "content":
						content = strings.TrimSpace(a.Val)
					}
				}
				switch {
				case prop == "og:title" && out.Title == "":
					out.Title = content
				case prop == "og:description" && out.Description == "":
					out.Description = content
				case prop == "og:image" && out.Image == "":
					out.Image = content
				case prop == "og:site_name" && out.SiteName == "":
					out.SiteName = content
				case name == "description" && out.Description == "":
					out.Description = content
				case name == "twitter:title" && out.Title == "":
					out.Title = content
				case name == "twitter:description" && out.Description == "":
					out.Description = content
				case name == "twitter:image" && out.Image == "":
					out.Image = content
				}
			case "title":
				if out.Title == "" && node.FirstChild != nil {
					out.Title = strings.TrimSpace(node.FirstChild.Data)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

// YouTubeVideoID extracts an 11-char video id from common YouTube URL shapes.
func YouTubeVideoID(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "youtu.be":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			return ""
		}
		return parts[0]
	case strings.Contains(host, "youtube.com"):
		if id := u.Query().Get("v"); id != "" {
			return id
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i, p := range parts {
			if (p == "embed" || p == "shorts" || p == "v") && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	return ""
}

// VimeoVideoID extracts a numeric id from vimeo.com/{id} URLs.
func VimeoVideoID(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	if !strings.Contains(strings.ToLower(u.Hostname()), "vimeo.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	id := parts[len(parts)-1]
	for _, c := range id {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return id
}
