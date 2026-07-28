package contenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/discussions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_discussion"
)

var (
	inlineDiscussionMetricsOnce sync.Once
	inlineDiscussionPostsTotal  = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_posts_total",
		Help:      "Inline Discussion post outcomes (CT.22).",
	}, []string{"outcome"})
)

func registerInlineDiscussionMetrics() {
	inlineDiscussionMetricsOnce.Do(func() {
		prometheus.MustRegister(inlineDiscussionPostsTotal)
		inlineDiscussionPostsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveInlineDiscussionPost increments lextures_content_tool_posts_total{outcome}.
func ObserveInlineDiscussionPost(outcome string) {
	registerInlineDiscussionMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	inlineDiscussionPostsTotal.WithLabelValues(outcome).Inc()
}

func init() {
	RegisterActionHandler(inline_discussion.ID, "post", handleInlineDiscussionPost)
	RegisterActionHandler(inline_discussion.ID, "thread", handleInlineDiscussionThread)
	RegisterActionHandler(inline_discussion.ID, "edit", handleInlineDiscussionEdit)
	RegisterActionHandler(inline_discussion.ID, "delete", handleInlineDiscussionDelete)
	RegisterActionHandler(inline_discussion.ID, "upvote", handleInlineDiscussionUpvote)
	RegisterActionHandler(inline_discussion.ID, "endorse", handleInlineDiscussionEndorse)
	RegisterActionHandler(inline_discussion.ID, "report", handleInlineDiscussionReport)
	RegisterActionHandler(inline_discussion.ID, "moderate", handleInlineDiscussionModerate)
	RegisterActionHandler(inline_discussion.ID, "get_post", handleInlineDiscussionGetPost)
}

func inlineDiscussionStaff(role string) bool {
	r := strings.ToLower(strings.TrimSpace(role))
	return r == "instructor" || r == "ta"
}

func handleInlineDiscussionPost(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	st := inline_discussion.ParseState(ctx.StateJSON)

	var in struct {
		Text           string `json:"text"`
		ParentPostID   string `json:"parentPostId"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid post input: %w", err)
		}
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		text = strings.TrimSpace(st.Draft)
	}
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if len(text) > 8000 {
		return nil, fmt.Errorf("text is too long (max 8000 characters)")
	}

	parentIDStr := strings.TrimSpace(in.ParentPostID)
	isReply := parentIDStr != ""
	if isReply && !cfg.AllowReplies {
		ObserveInlineDiscussionPost("replies_disabled")
		return &ActionResult{Result: map[string]any{
			"error": "replies_disabled", "message": "Replies are not enabled for this discussion.", "preserveInput": true,
		}}, nil
	}

	screen := ScreenFreeText(text, FilterActionBlock, true)
	if screen.Crisis {
		ObserveInlineDiscussionPost("crisis")
		if ctx.Pool != nil {
			actor := ctx.PrincipalID
			_ = RecordFilterFlag(ctx.Ctx, ctx.Pool, ctx.InstanceID, ctx.CourseID, &actor, FilterCategoryCrisis, FilterActionFlag)
			_ = RecordCrisisEscalation(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.InstanceID, &actor, inline_discussion.ID)
		}
		st.Draft = text
		patch, _ := json.Marshal(st)
		return &ActionResult{
			Result:     map[string]any{"error": "filtered", "message": screen.Guidance, "crisis": true, "preserveInput": true},
			StatePatch: patch,
		}, nil
	}
	if screen.Action == FilterActionBlock {
		ObserveInlineDiscussionPost("filtered")
		if ctx.Pool != nil {
			_ = RecordFilterFlag(ctx.Ctx, ctx.Pool, ctx.InstanceID, ctx.CourseID, &ctx.PrincipalID, screen.Category, FilterActionBlock)
		}
		return &ActionResult{Result: map[string]any{
			"error": "filtered", "message": screen.Guidance, "preserveInput": true,
		}}, nil
	}
	if screen.Action == FilterActionFlag && ctx.Pool != nil {
		_ = RecordFilterFlag(ctx.Ctx, ctx.Pool, ctx.InstanceID, ctx.CourseID, &ctx.PrincipalID, screen.Category, FilterActionFlag)
	}
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required for inline discussion posts")
	}

	threadID, err := inline_discussion.EnsureThread(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.InstanceID, ctx.PrincipalID, cfg, &st)
	if err != nil {
		return nil, err
	}

	var parentUUID *uuid.UUID
	if isReply {
		pid, err := uuid.Parse(parentIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid parentPostId")
		}
		ok, err := discussions.ParentPostThread(ctx.Ctx, ctx.Pool, threadID, pid)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("parent post not found in this thread")
		}
		depth, err := discussions.ParentPostDepth(ctx.Ctx, ctx.Pool, pid)
		if err != nil {
			return nil, err
		}
		if depth != 0 {
			ObserveInlineDiscussionPost("depth_exceeded")
			return &ActionResult{Result: map[string]any{
				"error": "depth_exceeded", "message": "Only one level of replies is allowed.", "preserveInput": true,
			}}, nil
		}
		parentUUID = &pid
	}

	idem := strings.TrimSpace(in.IdempotencyKey)
	if idem != "" {
		existing, err := discussions.FindIdempotentPost(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.PrincipalID, threadID, idem)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, *existing, &ctx.PrincipalID)
			if err != nil {
				return nil, err
			}
			view := inline_discussion.ProjectPost(cfg, post, ctx.PrincipalID, inlineDiscussionStaff(ctx.InteractRole), true)
			ObserveInlineDiscussionPost("idempotent")
			return &ActionResult{Result: map[string]any{"post": view, "state": inline_discussion.StateView(st)}}, nil
		}
	}

	body := inline_discussion.TipTapDocFromText(text, &inline_discussion.PostMeta{})
	tx, err := ctx.Pool.Begin(ctx.Ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx.Ctx) }()
	post, err := discussions.CreatePost(ctx.Ctx, tx, ctx.CourseID, threadID, ctx.PrincipalID, parentUUID, body, idem)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx.Ctx); err != nil {
		return nil, err
	}

	now := inline_discussion.NowRFC3339()
	st.ThreadID = threadID.String()
	st.Draft = ""
	st.V = 1
	if isReply {
		st.MyReplyIDs = inline_discussion.AppendUnique(st.MyReplyIDs, post.ID.String())
	} else {
		st.MyPostIDs = inline_discussion.AppendUnique(st.MyPostIDs, post.ID.String())
	}
	if inline_discussion.IsComplete(cfg, st) && st.CompletedAt == "" {
		st.CompletedAt = now
	}
	st.LastReadAt = now
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	staff := inlineDiscussionStaff(ctx.InteractRole)
	view := inline_discussion.ProjectPost(cfg, post, ctx.PrincipalID, staff, true)
	ObserveInlineDiscussionPost("ok")
	status := StatusInProgress
	if st.CompletedAt != "" {
		status = StatusCompleted
	}
	threadPayload, _ := inline_discussion.BuildThreadPayload(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.InstanceID, ctx.PrincipalID, cfg, st, staff, 1)
	result := map[string]any{"post": view, "state": inline_discussion.StateView(st)}
	for k, v := range threadPayload {
		result[k] = v
	}
	return &ActionResult{Result: result, StatePatch: patch, Status: status}, nil
}

func handleInlineDiscussionThread(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	st := inline_discussion.ParseState(ctx.StateJSON)
	staff := inlineDiscussionStaff(ctx.InteractRole)
	var in struct {
		Page int    `json:"page"`
		Sort string `json:"sort"`
	}
	if len(ctx.Input) > 0 {
		_ = json.Unmarshal(ctx.Input, &in)
	}
	if in.Sort == "newest" || in.Sort == "oldest" {
		cfg.Sort = inline_discussion.SortOrder(in.Sort)
	}
	page := in.Page
	if page < 1 {
		page = 1
	}
	payload, err := inline_discussion.BuildThreadPayload(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.InstanceID, ctx.PrincipalID, cfg, st, staff, page)
	if err != nil {
		return nil, err
	}
	return &ActionResult{Result: payload}, nil
}

func handleInlineDiscussionEdit(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	var in struct {
		PostID string `json:"postId"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid edit input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	screen := ScreenFreeText(text, FilterActionBlock, true)
	if screen.Action == FilterActionBlock || screen.Crisis {
		return &ActionResult{Result: map[string]any{
			"error": "filtered", "message": screen.Guidance, "preserveInput": true, "crisis": screen.Crisis,
		}}, nil
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found", "message": "Post not found."}}, nil
	}
	if post.AuthorID != ctx.PrincipalID {
		return &ActionResult{Result: map[string]any{"error": "forbidden", "message": "You can only edit your own posts."}}, nil
	}
	if !inline_discussion.WithinEditWindow(cfg, post.CreatedAt, time.Now().UTC()) {
		return &ActionResult{Result: map[string]any{
			"error": "edit_window_closed", "message": "The edit window for this post has closed.",
		}}, nil
	}
	meta := inline_discussion.MetaFromTipTap(post.Body)
	if meta.Removed {
		return &ActionResult{Result: map[string]any{"error": "not_found", "message": "Post not found."}}, nil
	}
	meta.EditedAt = inline_discussion.NowRFC3339()
	updated, err := discussions.UpdatePostBody(ctx.Ctx, ctx.Pool, postID, inline_discussion.TipTapDocFromText(text, &meta))
	if err != nil {
		return nil, err
	}
	view := inline_discussion.ProjectPost(cfg, updated, ctx.PrincipalID, inlineDiscussionStaff(ctx.InteractRole), true)
	return &ActionResult{Result: map[string]any{"post": view}}, nil
}

func handleInlineDiscussionDelete(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	if !cfg.AllowDelete {
		return &ActionResult{Result: map[string]any{
			"error": "delete_disabled", "message": "Deleting posts is not allowed in this discussion.",
		}}, nil
	}
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	var in struct {
		PostID string `json:"postId"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid delete input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found", "message": "Post not found."}}, nil
	}
	if post.AuthorID != ctx.PrincipalID && !inlineDiscussionStaff(ctx.InteractRole) {
		return &ActionResult{Result: map[string]any{"error": "forbidden", "message": "You can only delete your own posts."}}, nil
	}
	meta := inline_discussion.MetaFromTipTap(post.Body)
	meta.Removed = true
	updated, err := discussions.UpdatePostBody(ctx.Ctx, ctx.Pool, postID, inline_discussion.WithMeta(post.Body, meta))
	if err != nil {
		return nil, err
	}
	view := inline_discussion.ProjectPost(cfg, updated, ctx.PrincipalID, inlineDiscussionStaff(ctx.InteractRole), true)
	return &ActionResult{Result: map[string]any{"post": view}}, nil
}

func handleInlineDiscussionUpvote(ctx ActionContext) (*ActionResult, error) {
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	var in struct {
		PostID string `json:"postId"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid upvote input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found"}}, nil
	}
	meta := inline_discussion.MetaFromTipTap(post.Body)
	if meta.Removed && !inlineDiscussionStaff(ctx.InteractRole) {
		return &ActionResult{Result: map[string]any{"error": "not_found"}}, nil
	}
	_, count, err := discussions.Upvote(ctx.Ctx, ctx.Pool, postID, ctx.PrincipalID)
	if err != nil {
		return nil, err
	}
	post.UpvoteCount = count
	post.ViewerUpvoted = true
	view := inline_discussion.ProjectPost(cfg, post, ctx.PrincipalID, inlineDiscussionStaff(ctx.InteractRole), true)
	return &ActionResult{Result: map[string]any{"post": view}}, nil
}

func handleInlineDiscussionEndorse(ctx ActionContext) (*ActionResult, error) {
	if !inlineDiscussionStaff(ctx.InteractRole) {
		return &ActionResult{Result: map[string]any{
			"error": "forbidden", "message": "Only instructors can endorse posts.",
		}}, nil
	}
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	var in struct {
		PostID string `json:"postId"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid endorse input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found"}}, nil
	}
	meta := inline_discussion.MetaFromTipTap(post.Body)
	meta.Endorsed = true
	meta.EndorsedAt = inline_discussion.NowRFC3339()
	meta.EndorsedBy = ctx.PrincipalID.String()
	updated, err := discussions.UpdatePostBody(ctx.Ctx, ctx.Pool, postID, inline_discussion.WithMeta(post.Body, meta))
	if err != nil {
		return nil, err
	}
	view := inline_discussion.ProjectPost(cfg, updated, ctx.PrincipalID, true, true)
	return &ActionResult{Result: map[string]any{"post": view}}, nil
}

func handleInlineDiscussionReport(ctx ActionContext) (*ActionResult, error) {
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	var in struct {
		PostID   string `json:"postId"`
		Category string `json:"category"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid report input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found"}}, nil
	}
	path := "/discussionPosts/" + postID.String()
	cat := strings.TrimSpace(in.Category)
	if cat == "" {
		cat = "other"
	}
	reason := strings.TrimSpace(in.Reason)
	actor := ctx.PrincipalID
	subject := post.AuthorID
	row := ctrepo.ModerationRow{
		InstanceID: ctx.InstanceID, ContentPath: &path, Action: "reported",
		Category: &cat, ActorUserID: &actor, SubjectUserID: &subject,
	}
	if reason != "" {
		row.Reason = &reason
	}
	saved, err := ctrepo.InsertModeration(ctx.Ctx, ctx.Pool, row)
	if err != nil {
		return nil, err
	}
	IncModerationAction("reported")
	modID := ""
	if saved != nil {
		modID = saved.ID.String()
	}
	return &ActionResult{Result: map[string]any{"ok": true, "moderationId": modID}}, nil
}

func handleInlineDiscussionModerate(ctx ActionContext) (*ActionResult, error) {
	if !inlineDiscussionStaff(ctx.InteractRole) {
		return &ActionResult{Result: map[string]any{"error": "forbidden"}}, nil
	}
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	var in struct {
		PostID   string `json:"postId"`
		Action   string `json:"action"`
		Category string `json:"category"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid moderate input: %w", err)
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	switch action {
	case "hidden", "removed", "restored", "warned":
	default:
		return &ActionResult{Result: map[string]any{
			"error": "invalid_action", "message": "action must be hidden, removed, restored, or warned",
		}}, nil
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found"}}, nil
	}
	path := "/discussionPosts/" + postID.String()
	actor := ctx.PrincipalID
	subject := post.AuthorID
	row := ctrepo.ModerationRow{
		InstanceID: ctx.InstanceID, ContentPath: &path, Action: action,
		ActorUserID: &actor, SubjectUserID: &subject,
	}
	if c := strings.TrimSpace(in.Category); c != "" {
		row.Category = &c
	}
	if r := strings.TrimSpace(in.Reason); r != "" {
		row.Reason = &r
	}
	saved, err := ctrepo.InsertModeration(ctx.Ctx, ctx.Pool, row)
	if err != nil {
		return nil, err
	}
	IncModerationAction(action)
	meta := inline_discussion.MetaFromTipTap(post.Body)
	switch action {
	case "hidden", "removed":
		meta.Removed = true
	case "restored":
		meta.Removed = false
	}
	updated, err := discussions.UpdatePostBody(ctx.Ctx, ctx.Pool, postID, inline_discussion.WithMeta(post.Body, meta))
	if err != nil {
		return nil, err
	}
	view := inline_discussion.ProjectPost(cfg, updated, ctx.PrincipalID, true, true)
	modID := ""
	if saved != nil {
		modID = saved.ID.String()
	}
	return &ActionResult{Result: map[string]any{"post": view, "moderationId": modID}}, nil
}

func handleInlineDiscussionGetPost(ctx ActionContext) (*ActionResult, error) {
	if ctx.Pool == nil {
		return nil, fmt.Errorf("database required")
	}
	cfg := inline_discussion.ParseConfig(ctx.ConfigJSON)
	staff := inlineDiscussionStaff(ctx.InteractRole)
	var in struct {
		PostID string `json:"postId"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid get_post input: %w", err)
	}
	postID, err := uuid.Parse(strings.TrimSpace(in.PostID))
	if err != nil {
		return nil, fmt.Errorf("invalid postId")
	}
	post, err := discussions.GetPost(ctx.Ctx, ctx.Pool, ctx.CourseID, postID, &ctx.PrincipalID)
	if err != nil || post == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found", "status": 404}}, nil
	}
	meta := inline_discussion.MetaFromTipTap(post.Body)
	if meta.Removed && !staff && post.AuthorID != ctx.PrincipalID {
		return &ActionResult{Result: map[string]any{"error": "not_found", "status": 404}}, nil
	}
	view := inline_discussion.ProjectPost(cfg, post, ctx.PrincipalID, staff, true)
	if view == nil {
		return &ActionResult{Result: map[string]any{"error": "not_found", "status": 404}}, nil
	}
	return &ActionResult{Result: map[string]any{"post": view}}, nil
}

// GuardInlineDiscussionStatePut refuses participation mutations via PUT.
func GuardInlineDiscussionStatePut(toolID string, current, next json.RawMessage) (blocked bool, message string) {
	if toolID != inline_discussion.ID {
		return false, ""
	}
	return inline_discussion.GuardStatePut(current, next)
}

// SoftDeleteInlineDiscussionPosts soft-deletes posts authored by userID in the instance thread.
func SoftDeleteInlineDiscussionPosts(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID, authorID uuid.UUID) error {
	return inline_discussion.SoftDeletePosts(ctx, pool, courseID, instanceID, authorID)
}
