package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// PageQuery appends page/limit or cursor query params.
func PageQuery(base url.Values, page, limit int, cursor string) url.Values {
	q := cloneValues(base)
	if cursor != "" {
		q.Set("cursor", cursor)
		return q
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return q
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		for _, val := range vals {
			out.Add(k, val)
		}
	}
	return out
}

// PageResult is a generic paginated API envelope.
type PageResult struct {
	Items      []json.RawMessage
	NextCursor string
	HasMore    bool
}

// ExtractPage parses common pagination shapes from a JSON body.
func ExtractPage(body []byte, itemsKey string) (PageResult, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return PageResult{}, err
	}
	var items []json.RawMessage
	if raw, ok := root[itemsKey]; ok {
		_ = json.Unmarshal(raw, &items)
	}
	// Also accept plural keys used by some endpoints.
	if len(items) == 0 {
		for _, alt := range []string{"items", "data", "results"} {
			if raw, ok := root[alt]; ok {
				_ = json.Unmarshal(raw, &items)
				if len(items) > 0 {
					break
				}
			}
		}
	}
	res := PageResult{Items: items}
	if raw, ok := root["nextCursor"]; ok {
		_ = json.Unmarshal(raw, &res.NextCursor)
	}
	if raw, ok := root["hasMore"]; ok {
		_ = json.Unmarshal(raw, &res.HasMore)
	}
	if res.NextCursor == "" {
		if raw, ok := root["next"]; ok {
			var next string
			_ = json.Unmarshal(raw, &next)
			res.NextCursor = next
		}
	}
	if !res.HasMore && res.NextCursor != "" {
		res.HasMore = true
	}
	return res, nil
}

// FollowAllPages fetches every page using fetchPage until no more pages.
// fetchPage receives a query string suffix (including leading ?) and returns body bytes.
func FollowAllPages(fetchPage func(query string) ([]byte, error), baseQuery url.Values, limit int) ([][]byte, error) {
	if limit <= 0 {
		limit = 100
	}
	var bodies [][]byte
	cursor := ""
	page := 1
	for {
		q := PageQuery(baseQuery, page, limit, cursor)
		qs := ""
		if len(q) > 0 {
			qs = "?" + q.Encode()
		}
		body, err := fetchPage(qs)
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, body)
		pageRes, err := ExtractPage(body, "")
		if err != nil {
			// Not paginated — single page.
			return bodies, nil
		}
		if len(pageRes.Items) == 0 || !pageRes.HasMore {
			return bodies, nil
		}
		if pageRes.NextCursor != "" {
			cursor = pageRes.NextCursor
			continue
		}
		if len(pageRes.Items) < limit {
			return bodies, nil
		}
		page++
		if page > 1000 {
			return nil, fmt.Errorf("pagination exceeded 1000 pages")
		}
	}
}

// MergePageItems concatenates item arrays from multiple page bodies.
func MergePageItems(bodies [][]byte, itemsKey string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	for _, body := range bodies {
		page, err := ExtractPage(body, itemsKey)
		if err != nil {
			return nil, err
		}
		if len(page.Items) > 0 {
			all = append(all, page.Items...)
			continue
		}
		// Whole-body is a single object list at itemsKey.
		var root map[string]json.RawMessage
		if err := json.Unmarshal(body, &root); err != nil {
			continue
		}
		if raw, ok := root[itemsKey]; ok {
			var items []json.RawMessage
			if err := json.Unmarshal(raw, &items); err == nil {
				all = append(all, items...)
			}
		}
	}
	return all, nil
}

// SetClientRequestID adds a client request id header.
func SetClientRequestID(req *http.Request, id string) {
	if id != "" {
		req.Header.Set("X-Client-Request-Id", id)
	}
}