package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// canvasModuleItemCache holds Canvas API payloads prefetched before module DB writes.
type canvasModuleItemCache struct {
	assignments   map[int64]map[string]any
	quizzes       map[int64]map[string]any
	quizQuestions map[int64][]coursemodulequiz.QuizQuestion
	pages         map[string]map[string]any
}

func canvasPrefetchModuleItemData(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	modules []map[string]any,
	include canvasImportInclude,
) (*canvasModuleItemCache, error) {
	cache := &canvasModuleItemCache{
		assignments:   make(map[int64]map[string]any),
		quizzes:       make(map[int64]map[string]any),
		quizQuestions: make(map[int64][]coursemodulequiz.QuizQuestion),
		pages:         make(map[string]map[string]any),
	}

	assignmentIDs := make(map[int64]struct{})
	quizIDs := make(map[int64]struct{})
	pageSlugs := make(map[string]struct{})

	for _, m := range modules {
		for _, it := range arrAt(m, "items") {
			kind, _ := mapCanvasTypeToKind(strAt(it, "type", ""))
			switch kind {
			case "content_page":
				slug := strings.ToLower(strings.TrimSpace(strAt(it, "page_url", "")))
				if slug != "" {
					pageSlugs[slug] = struct{}{}
				}
			case "assignment":
				if include.Assignments {
					if cid := int64At(it, "content_id"); cid > 0 {
						assignmentIDs[cid] = struct{}{}
					}
				}
			case "quiz":
				if include.Quizzes {
					if cid := int64At(it, "content_id"); cid > 0 {
						quizIDs[cid] = struct{}{}
					}
				}
			}
		}
	}

	assignKeys := make([]int64, 0, len(assignmentIDs))
	for id := range assignmentIDs {
		assignKeys = append(assignKeys, id)
	}
	quizKeys := make([]int64, 0, len(quizIDs))
	for id := range quizIDs {
		quizKeys = append(quizKeys, id)
	}
	slugKeys := make([]string, 0, len(pageSlugs))
	for slug := range pageSlugs {
		slugKeys = append(slugKeys, slug)
	}

	g, gctx := canvasImportParallelGroup(ctx, len(assignKeys)+len(quizKeys)+len(slugKeys)+len(quizKeys))
	var cacheMu sync.Mutex

	for _, cid := range assignKeys {
		cid := cid
		g.Go(func() error {
			obj, err := canvasGetObject(gctx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/assignments/%d", canvasCourseID, cid), nil)
			if err != nil {
				return fmt.Errorf("prefetch assignment %d: %w", cid, err)
			}
			if obj != nil {
				_ = canvasEnrichAssignmentWithRubric(gctx, client, canvasBase, accessToken, canvasCourseID, obj)
				cacheMu.Lock()
				cache.assignments[cid] = obj
				cacheMu.Unlock()
			}
			return nil
		})
	}

	for _, cid := range quizKeys {
		cid := cid
		g.Go(func() error {
			obj, err := canvasGetObject(gctx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/quizzes/%d", canvasCourseID, cid), nil)
			if err != nil {
				return fmt.Errorf("prefetch quiz %d: %w", cid, err)
			}
			if obj != nil {
				cacheMu.Lock()
				cache.quizzes[cid] = obj
				cacheMu.Unlock()
			}
			return nil
		})
	}

	for _, cid := range quizKeys {
		cid := cid
		g.Go(func() error {
			qq, err := canvasImportQuizQuestions(gctx, client, canvasBase, accessToken, canvasCourseID, cid)
			if err != nil {
				return fmt.Errorf("prefetch quiz %d questions: %w", cid, err)
			}
			cacheMu.Lock()
			cache.quizQuestions[cid] = qq
			cacheMu.Unlock()
			return nil
		})
	}

	for _, slug := range slugKeys {
		slug := slug
		g.Go(func() error {
			page, err := canvasGetObject(gctx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/pages/%s", canvasCourseID, url.PathEscape(slug)), nil)
			if err != nil {
				return nil
			}
			if page != nil {
				cacheMu.Lock()
				cache.pages[slug] = page
				cacheMu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return cache, nil
}
