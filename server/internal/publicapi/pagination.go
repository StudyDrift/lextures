package publicapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CollectionMeta carries pagination metadata for collection responses.
type CollectionMeta struct {
	Total  int    `json:"total"`
	Cursor string `json:"cursor,omitempty"`
}

// CollectionLinks carries HATEOAS navigation links.
type CollectionLinks struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// CollectionResponse is the standard paginated collection envelope.
type CollectionResponse struct {
	Data  any             `json:"data"`
	Meta  CollectionMeta  `json:"meta"`
	Links CollectionLinks `json:"links"`
}

// PageParams holds decoded cursor pagination parameters.
type PageParams struct {
	Offset int
	Limit  int
}

const (
	defaultPageLimit = 25
	maxPageLimit     = 100
)

// ParsePageParams reads ?cursor= and ?limit= from the query string.
func ParsePageParams(q url.Values) (PageParams, error) {
	limit := defaultPageLimit
	if lim := strings.TrimSpace(q.Get("limit")); lim != "" {
		v, err := strconv.Atoi(lim)
		if err != nil || v <= 0 || v > maxPageLimit {
			return PageParams{}, fmt.Errorf("invalid limit")
		}
		limit = v
	}
	offset, err := DecodeCursor(q.Get("cursor"))
	if err != nil {
		return PageParams{}, err
	}
	return PageParams{Offset: offset, Limit: limit}, nil
}

// EncodeCursor returns an opaque cursor for the given row offset.
func EncodeCursor(offset int) string {
	payload, _ := json.Marshal(map[string]int{"offset": offset})
	return base64.RawURLEncoding.EncodeToString(payload)
}

// DecodeCursor parses an opaque cursor into a row offset.
func DecodeCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	var payload struct {
		Offset int `json:"offset"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Offset < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return payload.Offset, nil
}

// BuildCollectionResponse slices items and builds pagination metadata.
func BuildCollectionResponse(items []any, total, offset, limit int, basePath string, q url.Values) CollectionResponse {
	if items == nil {
		items = []any{}
	}
	end := offset + len(items)
	cursor := ""
	if end < total {
		cursor = EncodeCursor(end)
	}
	links := CollectionLinks{}
	if end < total {
		links.Next = buildPageLink(basePath, q, end, limit)
	}
	if offset > 0 {
		prevOff := offset - limit
		if prevOff < 0 {
			prevOff = 0
		}
		links.Prev = buildPageLink(basePath, q, prevOff, limit)
	}
	return CollectionResponse{
		Data: items,
		Meta: CollectionMeta{Total: total, Cursor: cursor},
		Links: links,
	}
}

// SetLinkHeader sets RFC 5988 Link header for collection navigation.
func SetLinkHeader(w http.ResponseWriter, links CollectionLinks) {
	var parts []string
	if links.Next != "" {
		parts = append(parts, `<`+links.Next+`>; rel="next"`)
	}
	if links.Prev != "" {
		parts = append(parts, `<`+links.Prev+`>; rel="prev"`)
	}
	if len(parts) > 0 {
		w.Header().Set("Link", strings.Join(parts, ", "))
	}
}

func buildPageLink(basePath string, q url.Values, offset, limit int) string {
	params := url.Values{}
	for k, vs := range q {
		if k == "cursor" || k == "limit" {
			continue
		}
		for _, v := range vs {
			params.Add(k, v)
		}
	}
	params.Set("limit", strconv.Itoa(limit))
	if offset > 0 {
		params.Set("cursor", EncodeCursor(offset))
	}
	return basePath + "?" + params.Encode()
}

// SlicePage returns the page slice and total count from a full in-memory list.
func SlicePage[T any](all []T, offset, limit int) ([]T, int) {
	total := len(all)
	if offset >= total {
		return []T{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total
}

// ToAnySlice converts a typed slice to []any for JSON collection envelopes.
func ToAnySlice[T any](items []T) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}
