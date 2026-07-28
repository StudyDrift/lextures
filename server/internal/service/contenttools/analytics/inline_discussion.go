package analytics

import (
	"encoding/json"
)

func init() {
	RegisterProjector("inline_discussion", projectInlineDiscussion, []FacetSchema{
		{Key: "postCount", Label: "contentTools.analytics.facets.postCount", Type: "number"},
		{Key: "replyCount", Label: "contentTools.analytics.facets.replyCount", Type: "number"},
		{Key: "hasPosted", Label: "contentTools.analytics.facets.hasPosted", Type: "boolean"},
		{Key: "hasReplied", Label: "contentTools.analytics.facets.hasReplied", Type: "boolean"},
	})
}

func projectInlineDiscussion(in ProjectInput) Summary {
	s := defaultProject(in)
	var st struct {
		MyPostIDs   []string `json:"myPostIds"`
		MyReplyIDs  []string `json:"myReplyIds"`
		CompletedAt string   `json:"completedAt"`
		LastReadAt  string   `json:"lastReadAt"`
	}
	_ = json.Unmarshal(in.StateJSON, &st)
	posts := len(st.MyPostIDs)
	replies := len(st.MyReplyIDs)
	if posts > 0 || replies > 0 || st.LastReadAt != "" {
		s.Engaged = true
	}
	if st.CompletedAt != "" || in.Status == "completed" {
		s.Completed = true
	}
	s.Facets["postCount"] = posts
	s.Facets["replyCount"] = replies
	s.Facets["hasPosted"] = posts > 0
	s.Facets["hasReplied"] = replies > 0
	return s
}
