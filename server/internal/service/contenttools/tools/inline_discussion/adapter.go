package inline_discussion

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/discussions"
)

// EnsureThread finds or creates the hidden forum + instance thread.
func EnsureThread(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID, authorID uuid.UUID, cfg Config, st *State) (uuid.UUID, error) {
	if st.ThreadID != "" {
		if tid, err := uuid.Parse(st.ThreadID); err == nil {
			ok, err := discussions.ThreadBelongsToCourse(ctx, pool, courseID, tid)
			if err != nil {
				return uuid.Nil, err
			}
			if ok {
				return tid, nil
			}
		}
	}
	forum, err := discussions.FindForumByName(ctx, pool, courseID, HiddenForumName)
	if err != nil {
		return uuid.Nil, err
	}
	if forum == nil {
		desc := "Content tool inline discussions (hidden from forum index)"
		forum, err = discussions.CreateForum(ctx, pool, courseID, HiddenForumName, desc, 9999)
		if err != nil {
			return uuid.Nil, err
		}
	}
	title := ThreadTitleForInstance(instanceID.String())
	th, err := discussions.FindThreadByTitle(ctx, pool, forum.ID, title)
	if err != nil {
		return uuid.Nil, err
	}
	if th != nil {
		st.ThreadID = th.ID.String()
		return th.ID, nil
	}
	body := TipTapDocFromText(cfg.Prompt, nil)
	created, err := discussions.CreateThread(ctx, pool, forum.ID, authorID, title, body, nil, cfg.PostBeforeYouSee)
	if err != nil {
		th2, err2 := discussions.FindThreadByTitle(ctx, pool, forum.ID, title)
		if err2 == nil && th2 != nil {
			st.ThreadID = th2.ID.String()
			return th2.ID, nil
		}
		return uuid.Nil, err
	}
	st.ThreadID = created.ID.String()
	return created.ID, nil
}

// SoftDeletePosts soft-deletes posts by author in the instance thread.
func SoftDeletePosts(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID, authorID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	forum, err := discussions.FindForumByName(ctx, pool, courseID, HiddenForumName)
	if err != nil || forum == nil {
		return err
	}
	th, err := discussions.FindThreadByTitle(ctx, pool, forum.ID, ThreadTitleForInstance(instanceID.String()))
	if err != nil || th == nil {
		return err
	}
	ids, err := discussions.ListPostIDsByAuthor(ctx, pool, th.ID, authorID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		post, err := discussions.GetPost(ctx, pool, courseID, id, &authorID)
		if err != nil || post == nil {
			continue
		}
		meta := MetaFromTipTap(post.Body)
		meta.Removed = true
		_, _ = discussions.UpdatePostBody(ctx, pool, id, WithMeta(post.Body, meta))
	}
	return nil
}

// ProjectPost builds the API view for one post.
func ProjectPost(cfg Config, post *discussions.PostRow, viewer uuid.UUID, staff, canSeePeers bool) map[string]any {
	if post == nil {
		return nil
	}
	meta := MetaFromTipTap(post.Body)
	isOwn := post.AuthorID == viewer
	if meta.Removed && !staff && !isOwn {
		return nil
	}
	if !canSeePeers && !staff && !isOwn {
		return nil
	}
	text := TextFromTipTap(post.Body)
	if meta.Removed {
		text = ""
	}
	out := map[string]any{
		"id":            post.ID.String(),
		"parentPostId":  nil,
		"text":          text,
		"upvoteCount":   post.UpvoteCount,
		"viewerUpvoted": post.ViewerUpvoted,
		"createdAt":     post.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":     post.UpdatedAt.UTC().Format(time.RFC3339),
		"endorsed":      meta.Endorsed,
		"removed":       meta.Removed,
		"editedAt":      meta.EditedAt,
		"isOwn":         isOwn,
		"canEdit":       isOwn && !meta.Removed && WithinEditWindow(cfg, post.CreatedAt, time.Now().UTC()),
		"canDelete":     cfg.AllowDelete && isOwn && !meta.Removed,
	}
	if post.ParentPostID != nil {
		out["parentPostId"] = post.ParentPostID.String()
	}
	if meta.Endorsed {
		out["endorsedAt"] = meta.EndorsedAt
	}
	if meta.Removed {
		out["tombstone"] = true
	}
	hideIdentity := cfg.Anonymity == AnonymityAnonymousToPeers && !staff && !isOwn
	if hideIdentity {
		out["authorDisplay"] = "Anonymous classmate"
		out["anonymous"] = true
	} else {
		out["authorId"] = post.AuthorID.String()
		out["anonymous"] = false
		if isOwn {
			out["authorDisplay"] = "You"
		} else if staff {
			out["authorDisplay"] = post.AuthorID.String()
		} else {
			out["authorDisplay"] = "Classmate"
		}
	}
	return out
}

// StateView is the participation state returned to clients.
func StateView(st State) map[string]any {
	return map[string]any{
		"v":           st.V,
		"threadId":    st.ThreadID,
		"myPostIds":   st.MyPostIDs,
		"myReplyIds":  st.MyReplyIDs,
		"lastReadAt":  st.LastReadAt,
		"completedAt": st.CompletedAt,
		"draft":       st.Draft,
	}
}

// BuildThreadPayload lists posts with gates applied.
func BuildThreadPayload(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, instanceID, principalID uuid.UUID,
	cfg Config,
	st State,
	staff bool,
	page int,
) (map[string]any, error) {
	canSee := CanSeePeers(cfg, st, staff)
	out := map[string]any{
		"prompt":      cfg.Prompt,
		"canSeePeers": canSee,
		"anonymity":   string(cfg.Anonymity),
		"archived":    false,
		"page":        page,
		"pageSize":    cfg.PageSize,
		"sort":        string(cfg.Sort),
		"state":       StateView(st),
		"requirements": map[string]any{
			"requiredPosts":   cfg.RequiredPosts,
			"requiredReplies": cfg.RequiredReplies,
			"myPosts":         len(st.MyPostIDs),
			"myReplies":       len(st.MyReplyIDs),
			"completed":       st.CompletedAt != "",
		},
	}
	if pool == nil {
		out["posts"] = []any{}
		out["total"] = 0
		if !canSee {
			out["locked"] = true
			out["lockReason"] = "post_before_you_see"
		}
		return out, nil
	}

	var threadID uuid.UUID
	if st.ThreadID != "" {
		parsed, err := uuid.Parse(st.ThreadID)
		if err != nil {
			out["posts"] = []any{}
			out["total"] = 0
			return out, nil
		}
		threadID = parsed
	} else {
		forum, err := discussions.FindForumByName(ctx, pool, courseID, HiddenForumName)
		if err != nil || forum == nil {
			out["posts"] = []any{}
			out["total"] = 0
			if !canSee {
				out["locked"] = true
				out["lockReason"] = "post_before_you_see"
			}
			return out, nil
		}
		th, err := discussions.FindThreadByTitle(ctx, pool, forum.ID, ThreadTitleForInstance(instanceID.String()))
		if err != nil || th == nil {
			out["posts"] = []any{}
			out["total"] = 0
			if !canSee {
				out["locked"] = true
				out["lockReason"] = "post_before_you_see"
			}
			return out, nil
		}
		threadID = th.ID
	}

	hidePeers := !canSee
	newestFirst := cfg.Sort == SortNewest
	rows, _, err := discussions.ListPostsOrdered(ctx, pool, threadID, principalID, staff, hidePeers, newestFirst, cfg.PageSize*5, 0)
	if err != nil {
		return nil, err
	}

	views := make([]map[string]any, 0, len(rows))
	filteredTotal := 0
	for i := range rows {
		meta := MetaFromTipTap(rows[i].Body)
		if meta.Removed && !staff && rows[i].AuthorID != principalID {
			continue
		}
		filteredTotal++
		v := ProjectPost(cfg, &rows[i], principalID, staff, canSee)
		if v == nil {
			continue
		}
		views = append(views, v)
	}
	sort.SliceStable(views, func(i, j int) bool {
		ei, _ := views[i]["endorsed"].(bool)
		ej, _ := views[j]["endorsed"].(bool)
		if ei != ej {
			return ei
		}
		return false
	})
	offset := (page - 1) * cfg.PageSize
	if offset > len(views) {
		offset = len(views)
	}
	end := offset + cfg.PageSize
	if end > len(views) {
		end = len(views)
	}
	out["posts"] = views[offset:end]
	out["total"] = filteredTotal
	out["threadId"] = threadID.String()
	if !canSee {
		out["locked"] = true
		out["lockReason"] = "post_before_you_see"
	}
	return out, nil
}
