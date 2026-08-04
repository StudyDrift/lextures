package coursechecklist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/service/coursechecklist/contentdoc"
	"github.com/lextures/lextures/server/internal/service/coursechecklist/linkhealth"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Evaluate runs the checklist engine against an in-memory snapshot (FR-6–FR-13).
func Evaluate(ctx context.Context, snap CourseSnapshot, opt EvaluateOptions) Result {
	reg := opt.Registry
	if reg == nil {
		reg = MustDefault()
	}

	start := time.Now()
	mode := "full"
	if len(opt.Only) > 0 {
		mode = "single"
	}

	ctx, span := telemetry.Tracer("coursechecklist").Start(ctx, "coursechecklist.Evaluate")
	defer span.End()
	span.SetAttributes(
		attribute.String("course_id", snap.CourseID.String()),
		attribute.String("catalog_version", catalogVersionFor(reg)),
		attribute.String("mode", mode),
	)

	items := reg.ItemsForEvaluate(opt)
	runLazyLoaders(ctx, &snap, items, opt.LazyLoaders)
	// Shared authored-content parse for accessibility / UDL rules (CC.6 FR-22 / AC-13).
	if needsContentDoc(items) {
		EnsureContentDoc(&snap)
	}

	findings := make([]ItemResult, 0, len(items))
	for _, it := range items {
		findings = append(findings, evaluateOne(ctx, snap, it))
	}

	// Deterministic category-then-registry order (already registry-ordered; re-group for counts).
	res := Result{
		Findings:       findings,
		Counts:         aggregateCounts(findings),
		ByCategory:     aggregateByCategory(findings),
		CatalogVersion: catalogVersionFor(reg),
		EngineVersion:  EngineVersion(),
	}

	elapsed := time.Since(start).Seconds()
	observeEvaluateDuration(mode, elapsed)
	slog.Debug("coursechecklist evaluated",
		"course_id", snap.CourseID.String(),
		"catalog_version", res.CatalogVersion,
		"mode", mode,
		"findings", len(findings),
		"seconds", elapsed,
	)
	return res
}

func needsContentDoc(items []ItemDescriptor) bool {
	for _, it := range items {
		switch it.ID {
		case ItemA11yImageAltText, ItemA11yVideoCaptions, ItemA11yHeadingStructure,
			ItemA11yLinkText, ItemA11yTableHeaders, ItemA11yTablesForLayout,
			ItemA11yTextFormatting, ItemA11yMediaAlternatives, ItemA11yPlainLanguage,
			ItemUDLMultipleRepresentations, ItemLinksExternalHealth:
			return true
		}
	}
	return false
}

func evaluateOne(ctx context.Context, snap CourseSnapshot, it ItemDescriptor) ItemResult {
	start := time.Now()
	out := ItemResult{
		ID:            it.ID,
		Category:      it.Category,
		Tier:          it.Tier,
		TitleKey:      it.TitleKey,
		TitleDefault:  it.TitleDefault,
		WhyKey:        it.WhyKey,
		WhyDefault:    it.WhyDefault,
		HelpRef:       it.HelpRef,
		Sources:       append([]string(nil), it.Sources...),
		Target:        it.Target,
		EvidenceShape: it.EvidenceShape,
	}

	applies := true
	if it.Applies != nil {
		applies = it.Applies(snap)
	}
	if !applies {
		out.Finding = Finding{
			Status:        StatusNotApplicable,
			DetailKey:     fmt.Sprintf("coursechecklist.item.%s.detail.na", it.ID),
			DetailDefault: "Does not apply to this course.",
		}
		observeRuleDuration(it.ID, time.Since(start).Seconds())
		return out
	}

	// Lazy timeout: if any declared lazy need is missing, report unknown.
	for _, lid := range it.LazyNeeds {
		if snap.Lazy == nil {
			out.Finding = unknownFinding(it.ID, "Required data was unavailable.")
			incRuleError(it.ID, "timeout")
			observeRuleDuration(it.ID, time.Since(start).Seconds())
			return out
		}
		if _, ok := snap.Lazy[lid]; !ok {
			out.Finding = unknownFinding(it.ID, "Required data was unavailable.")
			incRuleError(it.ID, "timeout")
			observeRuleDuration(it.ID, time.Since(start).Seconds())
			return out
		}
	}

	finding, err := safeEvaluate(ctx, it, snap)
	if err != nil {
		slog.Warn("coursechecklist rule error", "item_id", it.ID, "err", err)
		incRuleError(it.ID, "error")
		out.Finding = unknownFinding(it.ID, "Couldn't check this item.")
		observeRuleDuration(it.ID, time.Since(start).Seconds())
		return out
	}
	out.Finding = truncateEvidence(finding)
	observeRuleDuration(it.ID, time.Since(start).Seconds())
	return out
}

func safeEvaluate(ctx context.Context, it ItemDescriptor, snap CourseSnapshot) (finding Finding, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("coursechecklist rule panic", "item_id", it.ID, "panic", rec)
			incRuleError(it.ID, "panic")
			finding = unknownFinding(it.ID, "Couldn't check this item.")
			err = nil
		}
	}()
	return it.Evaluate(ctx, snap)
}

func unknownFinding(id ItemID, detail string) Finding {
	return Finding{
		Status:        StatusUnknown,
		DetailKey:     fmt.Sprintf("coursechecklist.item.%s.detail.unknown", id),
		DetailDefault: detail,
	}
}

func truncateEvidence(f Finding) Finding {
	if len(f.Evidence) <= MaxEvidenceRows {
		return f
	}
	f.Evidence = f.Evidence[:MaxEvidenceRows]
	n := MaxEvidenceRows
	f.TruncatedAt = &n
	return f
}

func aggregateCounts(findings []ItemResult) Counts {
	var c Counts
	for _, f := range findings {
		switch f.Finding.Status {
		case StatusDone:
			c.Done++
			c.Total++
		case StatusTodo:
			c.Todo++
			c.Total++
			if f.Tier == TierEssential {
				c.OutstandingEssential++
			}
		case StatusInProgress:
			c.InProgress++
			c.Total++
			if f.Tier == TierEssential {
				c.OutstandingEssential++
			}
		case StatusNotApplicable:
			c.NotApplicable++
		case StatusUnknown:
			c.Unknown++
		}
	}
	return c
}

func aggregateByCategory(findings []ItemResult) []CategoryCounts {
	buckets := make(map[CategoryID][]ItemResult, len(CategoryOrder))
	for _, f := range findings {
		buckets[f.Category] = append(buckets[f.Category], f)
	}
	out := make([]CategoryCounts, 0, len(CategoryOrder))
	for _, cat := range CategoryOrder {
		items := buckets[cat]
		if len(items) == 0 {
			continue
		}
		out = append(out, CategoryCounts{
			Category: cat,
			Counts:   aggregateCounts(items),
		})
	}
	// Any categories not in CategoryOrder (shouldn't happen) append sorted by name.
	for cat, items := range buckets {
		known := false
		for _, c := range CategoryOrder {
			if c == cat {
				known = true
				break
			}
		}
		if known {
			continue
		}
		out = append(out, CategoryCounts{Category: cat, Counts: aggregateCounts(items)})
	}
	return out
}

func runLazyLoaders(ctx context.Context, snap *CourseSnapshot, items []ItemDescriptor, loaders map[LazyLoaderID]LazyLoader) {
	if snap.Lazy == nil {
		snap.Lazy = make(map[LazyLoaderID]any)
	}
	needed := make(map[LazyLoaderID]struct{})
	for _, it := range items {
		applies := true
		if it.Applies != nil {
			applies = it.Applies(*snap)
		}
		if !applies {
			continue
		}
		for _, lid := range it.LazyNeeds {
			needed[lid] = struct{}{}
		}
	}
	for lid := range needed {
		loader, ok := loaders[lid]
		if !ok || loader == nil {
			// Missing loader → leave unset so evaluateOne marks unknown/timeout.
			continue
		}
		lctx, span := telemetry.Tracer("coursechecklist").Start(ctx, "coursechecklist.LazyLoader")
		span.SetAttributes(attribute.String("lazy_loader_id", string(lid)))
		lctx, cancel := context.WithTimeout(lctx, LazyLoaderBudget)
		err := loader.Load(lctx, snap)
		cancel()
		if err != nil {
			slog.Warn("coursechecklist lazy loader failed", "lazy_loader_id", lid, "err", err)
			span.End()
			continue
		}
		if _, present := snap.Lazy[lid]; !present {
			// Loader succeeded but did not publish — treat as loaded with nil marker.
			snap.Lazy[lid] = struct{}{}
		}
		span.End()
	}
}

// SerializeResultJSON returns a deterministic JSON encoding of Result for AC-7.
func SerializeResultJSON(r Result) ([]byte, error) {
	return json.Marshal(r)
}

// ContentDoc aliases the shared authored-content model (CC.6 FR-22).
type ContentDoc = contentdoc.Doc

// EnsureContentDoc returns snap.ContentDoc, parsing once when missing.
func EnsureContentDoc(snap *CourseSnapshot) *ContentDoc {
	if snap == nil {
		return &ContentDoc{}
	}
	if snap.ContentDoc != nil {
		return snap.ContentDoc
	}
	doc := parseContentDoc(*snap)
	snap.ContentDoc = doc
	return doc
}

func contentDocFor(snap CourseSnapshot) *ContentDoc {
	if snap.ContentDoc != nil {
		return snap.ContentDoc
	}
	return parseContentDoc(snap)
}

func parseContentDoc(snap CourseSnapshot) *ContentDoc {
	moduleTitle := map[uuid.UUID]string{}
	for _, it := range snap.StructureItems {
		if it.Kind == "module" {
			moduleTitle[it.ID] = it.Title
		}
	}
	var sources []contentdoc.Source
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		meta, ok := snap.ItemMeta[it.ID]
		if !ok || meta.BodyMarkdown == "" {
			continue
		}
		mod := ""
		if it.ParentID != nil {
			mod = moduleTitle[*it.ParentID]
		}
		sources = append(sources, contentdoc.Source{
			ItemID: it.ID, Kind: it.Kind, Title: it.Title, ModuleTitle: mod,
			Route: contentdoc.PageRoute(it.Kind), Markdown: meta.BodyMarkdown,
		})
	}
	for _, sec := range snap.SyllabusSections {
		if sec.Markdown == "" {
			continue
		}
		sources = append(sources, contentdoc.Source{
			Kind: "syllabus", Title: sec.Title,
			Route: "/courses/{courseCode}/syllabus", Markdown: sec.Markdown,
		})
	}
	return contentdoc.Parse(sources)
}

func contentPageRoute(kind string, id uuid.UUID) string {
	_ = id
	return contentdoc.PageRoute(kind)
}

type ContentPage = contentdoc.Page
type ContentImage = contentdoc.Image
type ContentHeading = contentdoc.Heading
type ContentLink = contentdoc.Link
type ContentTable = contentdoc.Table
type ContentMedia = contentdoc.Media

// JobTypeChecklistLinkCheck is the background job that populates link-health cache.
const JobTypeChecklistLinkCheck = "checklist-linkcheck"

// LinkCheckEnabled reports whether outbound link checking is enabled.
// Default false until security review (CC.6 §15); set CHECKLIST_LINKCHECK_ENABLED=true to enable.
func LinkCheckEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CHECKLIST_LINKCHECK_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// NewLinkHealthLazyLoader returns a LazyLoader that serves cached results or enqueues a check.
func NewLinkHealthLazyLoader(pool *pgxpool.Pool, enqueue bool) LazyLoader {
	return LazyFunc{
		LoaderID: LazyLinkHealth,
		Fn: func(ctx context.Context, snap *CourseSnapshot) error {
			if snap == nil {
				return nil
			}
			now := time.Now().UTC()
			rows, err := linkhealth.ListForCourse(ctx, pool, snap.CourseID)
			if err != nil {
				// Table missing / error → pending unknown.
				snap.Lazy[LazyLinkHealth] = LinkHealthLazy{Pending: true}
				return nil
			}
			if linkhealth.CacheFresh(rows, now) {
				var newest *time.Time
				capped := false
				for _, r := range rows {
					if r.Reason == "cap" {
						capped = true
					}
					t := r.CheckedAt
					if newest == nil || t.After(*newest) {
						newest = &t
					}
				}
				snap.Lazy[LazyLinkHealth] = LinkHealthLazy{
					Pending: false, Rows: rows, Capped: capped, CheckedAt: newest,
				}
				return nil
			}
			snap.Lazy[LazyLinkHealth] = LinkHealthLazy{Pending: true}
			if !LinkCheckEnabled() || !enqueue || pool == nil {
				return nil
			}
			payload, _ := json.Marshal(map[string]string{"courseId": snap.CourseID.String()})
			_, err = jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
				JobType:   JobTypeChecklistLinkCheck,
				Payload:   payload,
				Priority:  6,
				UniqueKey: linkhealth.LastEnqueuedKey(snap.CourseID),
			})
			if err != nil {
				slog.Debug("coursechecklist linkcheck enqueue skipped", "err", err, "course_id", snap.CourseID)
			}
			return nil
		},
	}
}

// ExtractExternalURLs returns distinct external http(s) URLs from a snapshot's ContentDoc.
func ExtractExternalURLs(snap CourseSnapshot) []string {
	doc := contentDocFor(snap)
	seen := map[string]struct{}{}
	var out []string
	for _, p := range doc.Pages {
		for _, link := range p.Links {
			href := strings.TrimSpace(link.Href)
			if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
				continue
			}
			n := linkhealth.NormalizeURL(href)
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	return out
}

// RunLinkCheckJob executes one course link-health check (worker entrypoint).
func RunLinkCheckJob(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	if !LinkCheckEnabled() {
		return nil
	}
	start := time.Now()
	// Load a minimal snapshot for URL extraction.
	code, err := courseCodeForID(ctx, pool, courseID)
	if err != nil || code == "" {
		return err
	}
	snap, err := LoadSnapshot(ctx, pool, code, []DataNeed{
		DataNeedCourse, DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus,
	})
	if err != nil {
		return err
	}
	EnsureContentDoc(&snap)
	urls := ExtractExternalURLs(snap)
	checker := &linkhealth.Checker{}
	results := checker.CheckURLs(ctx, urls)
	if err := linkhealth.UpsertResults(ctx, pool, courseID, results); err != nil {
		return err
	}
	linkhealth.ObserveDuration(time.Since(start).Seconds())
	return nil
}

func courseCodeForID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var code string
	err := pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&code)
	return code, err
}

