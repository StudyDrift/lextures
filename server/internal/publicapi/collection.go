package publicapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// CollectionMeta holds pagination metadata for JSON:API-style collections.
type CollectionMeta struct {
	Total  int    `json:"total"`
	Cursor string `json:"cursor,omitempty"`
}

// CollectionLinks holds pagination links.
type CollectionLinks struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// CollectionResponse is the standard paginated collection envelope.
type CollectionResponse struct {
	Data  any              `json:"data"`
	Meta  CollectionMeta   `json:"meta"`
	Links *CollectionLinks `json:"links,omitempty"`
}

// WriteCollection writes a paginated collection with optional Link header.
func WriteCollection(w http.ResponseWriter, status int, resp CollectionResponse) {
	if resp.Links != nil && resp.Links.Next != "" {
		w.Header().Set("Link", `<`+resp.Links.Next+`>; rel="next"`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// BuildPageLinks constructs next/prev URLs for cursor pagination.
func BuildPageLinks(basePath string, q url.Values, offset, limit, total int) *CollectionLinks {
	if total <= offset+limit && offset <= 0 {
		return nil
	}
	var links CollectionLinks
	if offset+limit < total {
		nq := cloneValues(q)
		nq.Set("cursor", EncodeCursor(offset+limit))
		nq.Set("limit", strconv.Itoa(limit))
		links.Next = basePath + "?" + nq.Encode()
	}
	if offset > 0 {
		pq := cloneValues(q)
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		if prev > 0 {
			pq.Set("cursor", EncodeCursor(prev))
		} else {
			pq.Del("cursor")
		}
		pq.Set("limit", strconv.Itoa(limit))
		links.Prev = basePath + "?" + pq.Encode()
	}
	return &links
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}

// PaginateSlice returns a page slice and next cursor token.
func PaginateSlice[T any](items []T, offset, limit int) ([]T, string) {
	if offset >= len(items) {
		return []T{}, ""
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	page := items[offset:end]
	next := ""
	if end < len(items) {
		next = EncodeCursor(end)
	}
	return page, next
}
