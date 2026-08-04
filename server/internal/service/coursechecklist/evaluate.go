package coursechecklist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lextures/lextures/server/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
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
