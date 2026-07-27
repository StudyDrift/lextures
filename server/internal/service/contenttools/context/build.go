package context

import (
	stdctx "context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	coursemodulecontent "github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// Build assembles a grounded context pack for a tool instance (FR-1, FR-2, FR-18).
func Build(ctx stdctx.Context, pool *pgxpool.Pool, instanceID uuid.UUID, opts BuildOpts) (*ContextPack, error) {
	start := time.Now()
	defer func() { observeBuild(time.Since(start)) }()

	if pool == nil {
		return nil, ErrInstanceNotFound
	}
	inst, err := ctrepo.GetInstanceByID(ctx, pool, instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}
	budget := opts.TokenBudget
	if budget <= 0 {
		budget = DefaultRequestContextTokens
	}
	topK := opts.TopKLinks
	if topK <= 0 {
		topK = DefaultTopKLinks
	}

	settings, _ := ctrepo.GetSettings(ctx, pool, inst.CourseID)
	policy := IngestPolicy{Mode: IngestPublic}
	if settings != nil {
		policy.Mode = settings.LinkIngestionMode
		policy.Allowlist = settings.LinkHostAllowlist
	}

	orgID, err := ctrepo.CourseOrgID(ctx, pool, inst.CourseID)
	if err != nil {
		return nil, err
	}
	if orgID == nil {
		zero := uuid.Nil
		orgID = &zero
	}

	var itemID uuid.UUID
	if inst.StructureItemID != nil {
		itemID = *inst.StructureItemID
	}

	markdown := opts.ServeVariantText
	var variantID *uuid.UUID = opts.ServeVariantID
	if markdown == "" && itemID != uuid.Nil {
		page, err := coursemodulecontent.GetForCourseItem(ctx, pool, inst.CourseID, itemID)
		if err != nil {
			return nil, err
		}
		if page != nil {
			markdown = page.Markdown
			// ACE served variant when learner id present (FR-18 / AC-10).
			if opts.LearnerUserID != nil {
				res := acsvc.ResolveServing(ctx, pool, acsvc.ServeRequest{
					CourseID:          inst.CourseID,
					BaseContentItemID: itemID,
					UserID:            *opts.LearnerUserID,
					BaseMarkdown:      markdown,
					CourseFlag:        true,
					GatewayAllowed:    true,
					EnqueueOnMiss:     false,
				})
				if res.IsAdapted && res.Markdown != "" {
					markdown = res.Markdown
					if res.ServedVariantID != nil {
						variantID = res.ServedVariantID
					}
				}
			}
		}
	}

	courseTitle, itemTitle, moduleTitle, _ := ctrepo.CourseTitles(ctx, pool, inst.CourseID, itemID)

	sectionText := SectionBodyForInstance(markdown, instanceID.String())
	pack := &ContextPack{
		InstanceID:     instanceID,
		Segments:       nil,
		PendingSources: nil,
		VariantID:      variantID,
	}

	add := func(seg ContextSegment) {
		if seg.Text == "" {
			return
		}
		seg.Tokens = EstimateTokens(seg.Text)
		if pack.TotalTokens+seg.Tokens > budget && len(pack.Segments) > 0 {
			return
		}
		if pack.TotalTokens+seg.Tokens > budget {
			// Truncate last resort for first segment.
			runes := []rune(seg.Text)
			maxRunes := budget * 4
			if maxRunes < len(runes) {
				seg.Text = string(runes[:maxRunes])
				seg.Tokens = EstimateTokens(seg.Text)
			}
		}
		pack.Segments = append(pack.Segments, seg)
		pack.TotalTokens += seg.Tokens
	}

	// Priority order (FR-2): section → siblings → activity → module/course → notes → files → links.
	secID := instanceID.String()
	if inst.SectionKey != nil && *inst.SectionKey != "" {
		secID = *inst.SectionKey
	}
	add(ContextSegment{Kind: KindSection, ID: secID, Title: "Section", Text: sectionText})

	// Sibling sections: remaining page body excluding the primary section slice.
	if markdown != "" && sectionText != "" && markdown != sectionText {
		sibling := strings.TrimSpace(strings.Replace(markdown, sectionText, "", 1))
		if sibling != "" {
			add(ContextSegment{Kind: KindSection, ID: itemID.String() + ":siblings", Title: "Related sections", Text: sibling})
		}
	}

	if itemTitle != "" {
		desc := itemTitle
		if moduleTitle != "" {
			desc = moduleTitle + " / " + itemTitle
		}
		add(ContextSegment{Kind: KindActivity, ID: itemID.String(), Title: itemTitle, Text: desc})
	}
	if courseTitle != "" {
		text := courseTitle
		if moduleTitle != "" {
			text = courseTitle + " — " + moduleTitle
		}
		add(ContextSegment{Kind: KindCourse, ID: inst.CourseID.String(), Title: courseTitle, Text: text})
	}

	notes := opts.PinnedNotes
	if notes == "" {
		notes = pinnedNotesFromConfig(inst.ConfigJSON)
	}
	if notes != "" {
		add(ContextSegment{Kind: KindNote, ID: instanceID.String() + ":notes", Title: "Instructor notes", Text: notes})
	}

	// Discover + ensure sources.
	links := DiscoverLinks(markdown, inst.ConfigJSON, opts.ConfigLinks)
	excluded := map[string]struct{}{}
	if itemID != uuid.Nil {
		existing, _ := ctrepo.ListActivitySources(ctx, pool, inst.CourseID, itemID)
		for _, row := range existing {
			if row.Excluded && row.URL != "" {
				if n, err := NormalizeURL(row.URL); err == nil {
					excluded[n] = struct{}{}
				} else {
					excluded[row.URL] = struct{}{}
				}
			}
		}
	}

	type ranked struct {
		url    string
		source *ctrepo.LinkSourceRow
		score  int
		origin string
	}
	var ready []ranked

	for _, link := range links {
		if _, skip := excluded[link]; skip {
			continue // AC-11
		}
		origin := OriginBodyLink
		for _, cfg := range linksFromConfig(inst.ConfigJSON) {
			if n, err := NormalizeURL(cfg); err == nil && n == link {
				origin = OriginConfigLink
			}
		}
		for _, cfg := range opts.ConfigLinks {
			if n, err := NormalizeURL(cfg); err == nil && n == link {
				origin = OriginConfigLink
			}
		}
		if opts.SkipLinkIngest || policy.Mode == IngestOff || KillSwitchActive() {
			pack.PendingSources = append(pack.PendingSources, PendingSource{
				URL: link, Status: StatusBlocked, Reason: "blocked: ingestion disabled",
			})
			continue
		}
		src, err := EnsureSource(ctx, pool, *orgID, link, policy)
		if err != nil || src == nil {
			st, reason := ReasonForFetchError(err)
			pack.PendingSources = append(pack.PendingSources, PendingSource{URL: link, Status: st, Reason: reason})
			continue
		}
		if itemID != uuid.Nil {
			_ = ctrepo.UpsertActivitySource(ctx, pool, inst.CourseID, itemID, src.ID, origin)
		}
		needsFetch := src.Status == StatusPending ||
			src.Status == StatusFailed ||
			(src.Status == StatusReady && src.ExpiresAt != nil && !src.ExpiresAt.After(time.Now()))
		if needsFetch && !opts.SkipLinkIngest {
			if opts.ForceSyncIngest {
				src, _ = IngestSource(ctx, pool, src.ID, policy)
			} else if opts.EnqueueIngest {
				IngestAsync(pool, src.ID, policy)
			}
		}
		if src == nil {
			continue
		}
		switch src.Status {
		case StatusReady:
			score := 0
			if opts.Query != "" && src.ExtractedText != nil {
				score = LexicalScore(opts.Query, *src.ExtractedText)
			}
			ready = append(ready, ranked{url: link, source: src, score: score, origin: origin})
		case StatusPending:
			pack.PendingSources = append(pack.PendingSources, PendingSource{URL: link, Status: StatusPending})
		default:
			reason := ""
			if src.Error != nil {
				reason = *src.Error
			}
			pack.PendingSources = append(pack.PendingSources, PendingSource{URL: link, Status: src.Status, Reason: reason})
		}
	}

	// Rank ready links and inject top-k passages (FR-10).
	for i := 0; i < len(ready); i++ {
		for j := i + 1; j < len(ready); j++ {
			if ready[j].score > ready[i].score {
				ready[i], ready[j] = ready[j], ready[i]
			}
		}
	}
	limit := topK
	if limit > len(ready) {
		limit = len(ready)
	}
	for _, r := range ready[:limit] {
		src := r.source
		title := r.url
		if src.Title != nil && *src.Title != "" {
			title = *src.Title
		}
		text := ""
		if src.ExtractedText != nil {
			text = *src.ExtractedText
		}
		// Prefer highest-scoring chunk when query present.
		if opts.Query != "" {
			chunks, _ := ctrepo.ListLinkChunks(ctx, pool, src.ID)
			bestScore, best := -1, text
			for _, ch := range chunks {
				sc := LexicalScore(opts.Query, ch.Text)
				if sc > bestScore {
					bestScore = sc
					best = ch.Text
				}
			}
			if best != "" {
				text = best
			}
		}
		lang := ""
		if src.Lang != nil {
			lang = *src.Lang
		}
		add(ContextSegment{
			Kind:   KindLink,
			ID:     src.ID.String(),
			Title:  title,
			URL:    src.URL,
			Lang:   lang,
			Text:   text,
		})
	}

	return pack, nil
}

func pinnedNotesFromConfig(configJSON json.RawMessage) string {
	if len(configJSON) == 0 {
		return ""
	}
	var generic map[string]any
	if err := json.Unmarshal(configJSON, &generic); err != nil {
		return ""
	}
	for _, key := range []string{"pinnedNotes", "instructorNotes", "notes"} {
		if v, ok := generic[key].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// FetchLink returns extracted passages for a URL already in the activity corpus (FR-9).
func FetchLink(ctx stdctx.Context, pool *pgxpool.Pool, orgID uuid.UUID, rawURL string, policy IngestPolicy) ([]ContextSegment, []Citation, error) {
	src, err := EnsureSource(ctx, pool, orgID, rawURL, policy)
	if err != nil {
		return nil, nil, err
	}
	if src.Status != StatusReady {
		src, err = IngestSource(ctx, pool, src.ID, policy)
		if err != nil {
			return nil, nil, err
		}
	}
	if src == nil || src.Status != StatusReady || src.ExtractedText == nil {
		return nil, nil, ErrUnsupportedType
	}
	title := src.URL
	if src.Title != nil && *src.Title != "" {
		title = *src.Title
	}
	chunks, err := ctrepo.ListLinkChunks(ctx, pool, src.ID)
	if err != nil {
		return nil, nil, err
	}
	segs := make([]ContextSegment, 0, len(chunks))
	cites := make([]Citation, 0, len(chunks))
	for _, ch := range chunks {
		id := src.ID.String()
		segs = append(segs, ContextSegment{
			Kind: KindLink, ID: id, Title: title, URL: src.URL, Text: ch.Text, Tokens: ch.TokenCount,
		})
		cites = append(cites, Citation{
			Kind: CiteLink, ID: id, Title: title, URL: src.URL, Loc: "chunk:" + itoa(ch.Ordinal),
		})
	}
	if len(segs) == 0 {
		segs = []ContextSegment{{
			Kind: KindLink, ID: src.ID.String(), Title: title, URL: src.URL,
			Text: *src.ExtractedText, Tokens: EstimateTokens(*src.ExtractedText),
		}}
		cites = []Citation{{Kind: CiteLink, ID: src.ID.String(), Title: title, URL: src.URL}}
	}
	return segs, cites, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var neg bool
	if n < 0 {
		neg = true
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
