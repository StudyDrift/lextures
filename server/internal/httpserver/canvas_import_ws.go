package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"log"
	"path/filepath"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
)

// handleCourseImportCanvasWS is GET /api/v1/courses/{course_code}/import/canvas/ws.
// Legacy endpoint: accepts the same first message as before, enqueues the import, and returns the job id.
func (d Deps) handleCourseImportCanvasWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil || d.Pool == nil || d.CanvasImportQueue == nil {
			http.Error(w, "server misconfiguration", http.StatusServiceUnavailable)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			http.Error(w, "missing course", http.StatusBadRequest)
			return
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}

		readAuthCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()
		typ, b, err := c.Read(readAuthCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}
		var req canvasImportWSFirstMessage
		if err := json.Unmarshal(b, &req); err != nil || req.AuthToken == "" {
			_ = wsWriteJSON(r.Context(), c, map[string]any{
				"type":    "error",
				"message": "Invalid JSON in the first message. Send authToken plus the former Canvas import POST body fields.",
			})
			return
		}
		u, err := d.JWTSigner.Verify(r.Context(), req.AuthToken)
		if err != nil {
			return
		}
		uid, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}
		hasAccess, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, uid)
		if err != nil || !hasAccess {
			return
		}
		canImport, err := courseroles.UserHasPermission(r.Context(), d.Pool, uid, "course:"+courseCode+":item:create")
		if err != nil || !canImport {
			return
		}
		if req.CanvasBaseURL == "" || req.CanvasCourseID == "" || req.AccessToken == "" {
			_ = wsWriteJSON(r.Context(), c, map[string]any{
				"type":    "error",
				"message": "Canvas base URL, course id, and access token are required.",
			})
			return
		}
		include := req.Include.withDefaults()

		jobID, err := canvasimportjobs.Insert(r.Context(), d.Pool, uid, courseCode, req.Mode, req.CanvasBaseURL, req.CanvasCourseID, includeToRepo(include))
		if err != nil {
			_ = wsWriteJSON(r.Context(), c, map[string]any{"type": "error", "message": "Failed to queue Canvas import."})
			return
		}
		msg := canvasimportjobs.QueueMessage{
			JobID:          jobID,
			UserID:         uid,
			CourseCode:     courseCode,
			Mode:           req.Mode,
			CanvasBaseURL:  req.CanvasBaseURL,
			CanvasCourseID: req.CanvasCourseID,
			AccessToken:    req.AccessToken,
			Include:        includeToRepo(include),
		}
		if err := d.CanvasImportQueue.Publish(r.Context(), msg); err != nil {
			_ = wsWriteJSON(r.Context(), c, map[string]any{"type": "error", "message": "Failed to enqueue Canvas import."})
			return
		}
		_ = wsWriteJSON(r.Context(), c, map[string]any{
			"type":    "queued",
			"jobId":   jobID.String(),
			"message": "Canvas import queued. You can leave this page and refresh later — we will notify you when it finishes.",
		})
	}
}

type canvasImportWSFirstMessage struct {
	AuthToken      string              `json:"authToken"`
	Mode           string              `json:"mode"`
	CanvasBaseURL  string              `json:"canvasBaseUrl"`
	CanvasCourseID string              `json:"canvasCourseId"`
	AccessToken    string              `json:"accessToken"`
	Include        canvasImportInclude `json:"include"`
}

type canvasImportInclude struct {
	Modules     bool `json:"modules"`
	Assignments bool `json:"assignments"`
	Quizzes     bool `json:"quizzes"`
	Enrollments bool `json:"enrollments"`
	Grades      bool `json:"grades"`
	Settings    bool `json:"settings"`
	Files       bool `json:"files"`
}

func (i canvasImportInclude) withDefaults() canvasImportInclude {
	if !i.Modules && !i.Assignments && !i.Quizzes && !i.Enrollments && !i.Grades && !i.Settings && !i.Files {
		return canvasImportInclude{Modules: true, Assignments: true, Quizzes: true, Enrollments: true, Grades: true, Settings: true, Files: true}
	}
	// Legacy clients send every category except files; treat that as "import everything".
	if i.Modules && i.Assignments && i.Quizzes && i.Enrollments && i.Grades && i.Settings && !i.Files {
		i.Files = true
	}
	return i
}

func wsWriteJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

func (d Deps) runCanvasImport(
	ctx context.Context,
	importerUserID uuid.UUID,
	courseCode, mode, canvasBaseURL, canvasCourseIDRaw, accessToken string,
	include canvasImportInclude,
	progress func(string) bool,
) error {
	if d.Pool == nil {
		return errors.New("server misconfiguration")
	}
	if mode != "erase" && mode != "mergeAdd" && mode != "overwrite" {
		return errors.New("Invalid import mode.")
	}
	canvasBase, err := normalizeCanvasBaseURL(canvasBaseURL, d.effectiveConfig().CanvasAllowedHostSuffixes)
	if err != nil {
		return err
	}
	canvasCourseID, err := strconv.ParseInt(strings.TrimSpace(canvasCourseIDRaw), 10, 64)
	if err != nil {
		return errors.New("Canvas course id must be a number (the id from the Canvas course URL).")
	}
	client := canvasHTTPClient()

	var course map[string]any
	modules := []map[string]any{}
	enrollmentRows := []map[string]any{}
	canvasSections := []map[string]any{}
	rosterEmailByCanvasUID := make(map[int64]string)
	needEnrollmentRows := include.Enrollments || include.Grades || include.Assignments
	needCanvasSections := needEnrollmentRows || include.Assignments || include.Quizzes || include.Modules

	if !progress("Loading course data from Canvas...") {
		return context.Canceled
	}
	prefetchTasks := 1
	if include.Modules {
		prefetchTasks++
	}
	if needEnrollmentRows {
		prefetchTasks += 2
	}
	if needCanvasSections {
		prefetchTasks++
	}
	prefetchGroup, prefetchCtx := canvasImportParallelGroup(ctx, prefetchTasks)
	prefetchGroup.Go(func() error {
		var loadErr error
		course, loadErr = canvasGetObject(prefetchCtx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d", canvasCourseID), url.Values{"include[]": []string{"syllabus_body", "course_image"}})
		return loadErr
	})
	if include.Modules {
		prefetchGroup.Go(func() error {
			var loadErr error
			modules, loadErr = canvasGetArrayPaginated(prefetchCtx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/modules", canvasCourseID), url.Values{"include[]": []string{"items"}})
			return loadErr
		})
	}
	if needEnrollmentRows {
		prefetchGroup.Go(func() error {
			var loadErr error
			enrollmentRows, loadErr = canvasFetchEnrollmentRowsForImport(prefetchCtx, client, canvasBase, accessToken, canvasCourseID)
			return loadErr
		})
		prefetchGroup.Go(func() error {
			var loadErr error
			rosterEmailByCanvasUID, loadErr = canvasRosterEmailsByCanvasUserID(prefetchCtx, client, canvasBase, accessToken, canvasCourseID)
			return loadErr
		})
	}
	if needCanvasSections {
		prefetchGroup.Go(func() error {
			var loadErr error
			canvasSections, loadErr = canvasFetchCourseSections(prefetchCtx, client, canvasBase, accessToken, canvasCourseID)
			return loadErr
		})
	}
	if err := prefetchGroup.Wait(); err != nil {
		return err
	}
	if needEnrollmentRows {
		if !progress(fmt.Sprintf("Loaded course data from Canvas (%d enrollment row(s) for import).", len(enrollmentRows))) {
			return context.Canceled
		}
	} else if !progress("Loaded course data from Canvas.") {
		return context.Canceled
	}

	var moduleItemCache *canvasModuleItemCache
	if include.Modules && len(modules) > 0 {
		if !progress("Prefetching module content from Canvas...") {
			return context.Canceled
		}
		moduleItemCache, err = canvasPrefetchModuleItemData(ctx, client, canvasBase, accessToken, canvasCourseID, modules, include)
		if err != nil {
			return err
		}
	}

	var canvasUserToLocal map[int64]uuid.UUID
	if include.Grades || include.Assignments {
		canvasUserToLocal = buildCanvasUserIDToLexturesUserID(ctx, d.Pool, client, canvasBase, accessToken, canvasCourseID, enrollmentRows, rosterEmailByCanvasUID)
	}

	var courseID uuid.UUID
	var orgID uuid.UUID
	err = d.Pool.QueryRow(ctx, `SELECT id, org_id FROM course.courses WHERE course_code = $1`, courseCode).Scan(&courseID, &orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("Course not found or you do not have access.")
	}
	if err != nil {
		return errors.New("Failed to load course.")
	}

	var canvasSectionMap map[int64]uuid.UUID
	if len(canvasSections) > 0 {
		if !progress(fmt.Sprintf("Importing %d Canvas section(s)...", len(canvasSections))) {
			return context.Canceled
		}
		var sectionStats *canvasSectionImportStats
		canvasSectionMap, sectionStats, err = canvasImportCourseSections(ctx, d.Pool, courseID, orgID, canvasSections)
		if err != nil {
			return err
		}
		if sectionStats != nil {
			msg := fmt.Sprintf("Imported %d course section(s) from Canvas.", sectionStats.SectionsCreated+sectionStats.SectionsUpdated)
			if sectionStats.CrossListLinked {
				msg += " Linked cross-listed sections for a combined gradebook."
			}
			if !progress(msg) {
				return context.Canceled
			}
		}
	}

	if include.Settings {
		settingsTx, settingsErr := d.Pool.Begin(ctx)
		if settingsErr != nil {
			return errors.New("Failed to start import transaction.")
		}
		title := strAt(course, "name", "Imported Canvas course")
		// Avoid stuffing the syllabus (or HTML public description) into the short course
		// blurb—the syllabus still lands on the dedicated syllabus record below.
		desc := title
		published := strAt(course, "workflow_state", "available") == "available"
		_, settingsErr = settingsTx.Exec(ctx, `UPDATE course.courses SET title = $1, description = $2, published = $3, updated_at = NOW() WHERE id = $4`, title, desc, published, courseID)
		if settingsErr != nil {
			_ = settingsTx.Rollback(ctx)
			return errors.New("Failed to update course settings.")
		}
		// Use the Canvas course banner as the course hero image (shown on the
		// course dashboard banner and the courses page cards). Only overwrite
		// when Canvas actually has an image so we don't clear an existing hero.
		if heroURL := strAt(course, "image_download_url", ""); heroURL != "" {
			_, settingsErr = settingsTx.Exec(ctx, `UPDATE course.courses SET hero_image_url = $1, updated_at = NOW() WHERE id = $2`, heroURL, courseID)
			if settingsErr != nil {
				_ = settingsTx.Rollback(ctx)
				return errors.New("Failed to update course banner.")
			}
		}
		syllabusHTML := strAt(course, "syllabus_body", "")
		if syllabusHTML != "" {
			sections, _ := json.Marshal([]map[string]string{{
				"id":       "canvas-syllabus",
				"heading":  "Syllabus",
				"markdown": markdownFromHTML(syllabusHTML),
			}})
			_, settingsErr = settingsTx.Exec(ctx, `
				INSERT INTO course.course_syllabus (course_id, sections, require_syllabus_acceptance, updated_at)
				VALUES ($1, $2, false, NOW())
				ON CONFLICT (course_id) DO UPDATE SET sections = EXCLUDED.sections, updated_at = NOW()
			`, courseID, sections)
			if settingsErr != nil {
				_ = settingsTx.Rollback(ctx)
				return errors.New("Failed to update syllabus.")
			}
		}
		if settingsErr = settingsTx.Commit(ctx); settingsErr != nil {
			return errors.New("Something went wrong while saving the import.")
		}
		broadcastStructureChanged(courseCode)
		d.notifyCourses(importerUserID)
	}

	if include.Modules && (mode == "erase" || mode == "overwrite") {
		if !progress("Clearing existing course modules...") {
			return context.Canceled
		}
		wipeTx, wipeErr := d.Pool.Begin(ctx)
		if wipeErr != nil {
			return errors.New("Failed to start import transaction.")
		}
		if _, wipeErr = wipeTx.Exec(ctx, `DELETE FROM course.course_structure_items WHERE course_id = $1`, courseID); wipeErr != nil {
			_ = wipeTx.Rollback(ctx)
			return errors.New("Failed to clear existing module structure.")
		}
		if wipeErr = wipeTx.Commit(ctx); wipeErr != nil {
			return errors.New("Something went wrong while saving the import.")
		}
		broadcastStructureChanged(courseCode)
	}

	nextSort := 0
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), -1) + 1 FROM course.course_structure_items WHERE course_id = $1`, courseID).Scan(&nextSort)
	canvasAssignToItem := make(map[int64]uuid.UUID)
	canvasQuizToItem := make(map[int64]uuid.UUID)
	canvasQuizToQuestions := make(map[int64][]coursemodulequiz.QuizQuestion)
	canvasQuizToAssignmentID := make(map[int64]int64)
	canvasPageSlugToItem := make(map[string]uuid.UUID)
	if include.Modules {
		if !progress("Importing modules and items...") {
			return context.Canceled
		}
		for _, m := range modules {
			moduleTx, moduleErr := d.Pool.Begin(ctx)
			if moduleErr != nil {
				return errors.New("Failed to start import transaction.")
			}
			moduleFailed := func(msg string) error {
				_ = moduleTx.Rollback(ctx)
				return errors.New(msg)
			}
			moduleID := uuid.New()
			title := strAt(m, "name", "Module")
			published := boolAt(m, "published", true)
			if _, moduleErr = moduleTx.Exec(ctx, `
				INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
				VALUES ($1, $2, $3, 'module', $4, NULL, $5, false)
			`, moduleID, courseID, nextSort, title, published); moduleErr != nil {
				return moduleFailed("Failed to insert module item.")
			}
			nextSort++
			items := arrAt(m, "items")
			for _, it := range items {
				kind, bodyTable := mapCanvasTypeToKind(strAt(it, "type", ""))
				if kind == "" {
					continue
				}
				if kind == "assignment" && !include.Assignments {
					continue
				}
				if kind == "quiz" && !include.Quizzes {
					continue
				}
				itemID := uuid.New()
				itemTitle := strAt(it, "title", "Item")
				itemPublished := boolAt(it, "published", published)
				if _, moduleErr = moduleTx.Exec(ctx, `
					INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
					VALUES ($1, $2, $3, $4, $5, $6, $7, false)
				`, itemID, courseID, nextSort, kind, itemTitle, moduleID, itemPublished); moduleErr != nil {
					return moduleFailed("Failed to insert module child item.")
				}
				nextSort++
				switch bodyTable {
				case "content":
					md := ""
					if strAt(it, "type", "") == "File" {
						if cid := int64At(it, "content_id"); cid > 0 {
							name := strAt(it, "title", itemTitle)
							md = fmt.Sprintf("[%s](/courses/%d/files/%d)", name, canvasCourseID, cid)
						}
					} else if kind == "content_page" {
						pageURL := strAt(it, "page_url", "")
						if pageURL != "" {
							slug := strings.ToLower(strings.TrimSpace(pageURL))
							canvasPageSlugToItem[slug] = itemID
							if moduleItemCache != nil {
								if page := moduleItemCache.pages[slug]; page != nil {
									md = markdownFromHTML(strAt(page, "body", ""))
								}
							}
						}
					} else {
						link := strAt(it, "html_url", "")
						if link != "" {
							md = fmt.Sprintf("**%s**\n\n[Open in Canvas](%s)", itemTitle, link)
						}
					}
					if _, moduleErr = moduleTx.Exec(ctx, `INSERT INTO course.module_content_pages (structure_item_id, markdown) VALUES ($1, $2)`, itemID, md); moduleErr != nil {
						return moduleFailed("Failed to save imported page content.")
					}
				case "assignment":
					markdown := ""
					var pointsWorth *int
					var dueAt, availFrom, availUntil *time.Time
					var rubricJSON []byte
					if cid := int64At(it, "content_id"); cid > 0 {
						canvasAssignToItem[cid] = itemID
						if _, moduleErr = moduleTx.Exec(ctx, `UPDATE course.course_structure_items SET canvas_assignment_id = $1 WHERE id = $2`, cid, itemID); moduleErr != nil {
							return moduleFailed("Failed to save Canvas assignment id.")
						}
						var obj map[string]any
						if moduleItemCache != nil {
							obj = moduleItemCache.assignments[cid]
						}
						if obj != nil {
							markdown = markdownFromHTML(strAt(obj, "description", ""))
							pointsWorth = optionalPointsWorthFromCanvas(obj, "points_possible")
							dueAt = canvasTimeAt(obj, "due_at")
							availFrom = canvasTimeAt(obj, "unlock_at")
							availUntil = canvasTimeAt(obj, "lock_at")
							if raw, rubErr := canvasOptionalRubricJSONFromAssignment(obj); rubErr == nil && len(raw) > 0 {
								rubricJSON = raw
							}
						}
					}
					if _, moduleErr = moduleTx.Exec(ctx, `INSERT INTO course.module_assignments (structure_item_id, markdown, points_worth, available_from, available_until, rubric_json) VALUES ($1, $2, $3, $4, $5, $6)`, itemID, markdown, pointsWorth, availFrom, availUntil, nullableJSONBytes(rubricJSON)); moduleErr != nil {
						return moduleFailed("Failed to save imported assignment.")
					}
					if dueAt != nil {
						if _, moduleErr = moduleTx.Exec(ctx, `UPDATE course.course_structure_items SET due_at = $1 WHERE id = $2`, dueAt, itemID); moduleErr != nil {
							return moduleFailed("Failed to save imported assignment due date.")
						}
					}
				case "quiz":
					markdown := ""
					var questions []coursemodulequiz.QuizQuestion
					var pointsWorth *int
					var dueAt, availFrom, availUntil *time.Time
					if cid := int64At(it, "content_id"); cid > 0 {
						canvasQuizToItem[cid] = itemID
						var canvasAssignmentID *int64
						if moduleItemCache != nil {
							if obj := moduleItemCache.quizzes[cid]; obj != nil {
								markdown = markdownFromHTML(strAt(obj, "description", ""))
								pointsWorth = optionalPointsWorthFromCanvas(obj, "points_possible")
								dueAt = canvasTimeAt(obj, "due_at")
								availFrom = canvasTimeAt(obj, "unlock_at")
								availUntil = canvasTimeAt(obj, "lock_at")
								if aid := int64At(obj, "assignment_id"); aid > 0 {
									canvasQuizToAssignmentID[cid] = aid
									canvasAssignmentID = &aid
								}
							}
							questions = moduleItemCache.quizQuestions[cid]
							canvasQuizToQuestions[cid] = questions
						}
						if _, moduleErr = moduleTx.Exec(ctx, `UPDATE course.course_structure_items SET canvas_quiz_id = $1, canvas_assignment_id = $2 WHERE id = $3`, cid, canvasAssignmentID, itemID); moduleErr != nil {
							return moduleFailed("Failed to save Canvas quiz id.")
						}
					}
					qJSON, mj := json.Marshal(questions)
					if mj != nil {
						return moduleFailed("Failed to encode imported quiz questions.")
					}
					if _, moduleErr = moduleTx.Exec(ctx, `INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json, points_worth, available_from, available_until) VALUES ($1, $2, $3, $4, $5, $6)`, itemID, markdown, qJSON, pointsWorth, availFrom, availUntil); moduleErr != nil {
						return moduleFailed("Failed to save imported quiz.")
					}
					if dueAt != nil {
						if _, moduleErr = moduleTx.Exec(ctx, `UPDATE course.course_structure_items SET due_at = $1 WHERE id = $2`, dueAt, itemID); moduleErr != nil {
							return moduleFailed("Failed to save imported quiz due date.")
						}
					}
				case "external":
					raw := strAt(it, "external_url", "")
					if raw == "" {
						raw = strAt(it, "html_url", "")
					}
					if _, moduleErr = moduleTx.Exec(ctx, `INSERT INTO course.module_external_links (structure_item_id, url) VALUES ($1, $2)`, itemID, raw); moduleErr != nil {
						return moduleFailed("Failed to save imported external link.")
					}
				}
			}
			if moduleErr = moduleTx.Commit(ctx); moduleErr != nil {
				return errors.New("Something went wrong while saving the import.")
			}
			broadcastStructureChanged(courseCode)
		}
	}

	if len(canvasSectionMap) > 0 && (include.Assignments || include.Quizzes) && len(canvasAssignToItem)+len(canvasQuizToItem) > 0 {
		if !progress("Importing per-section due dates from Canvas...") {
			return context.Canceled
		}
		overrideCount, overrideErr := canvasImportSectionAssignmentOverrides(
			ctx, d.Pool, client, canvasBase, accessToken, canvasCourseID, courseID,
			canvasAssignToItem, canvasQuizToItem, canvasSectionMap,
		)
		if overrideErr != nil {
			return overrideErr
		}
		if overrideCount > 0 {
			if !progress(fmt.Sprintf("Imported %d per-section due date override(s) from Canvas.", overrideCount)) {
				return context.Canceled
			}
		}
	}

	if include.Enrollments {
		if !progress("Applying enrollments from Canvas...") {
			return context.Canceled
		}
		enrollTx, enrollErr := d.Pool.Begin(ctx)
		if enrollErr != nil {
			return errors.New("Failed to start import transaction.")
		}
		enrollFailed := false
		defer func() {
			if enrollFailed {
				_ = enrollTx.Rollback(ctx)
			}
		}()
		var enrollStats canvasEnrollmentImportStats
		for _, e := range enrollmentRows {
			u := objAt(e, "user")
			canvasUID := canvasCanvasUserIDFromEnrollment(e, u)
			if canvasUID <= 0 {
				continue
			}
			userID, err := canvasResolveLexturesUserForEnrollment(ctx, d.Pool, enrollTx, orgID, canvasUID, rosterEmailByCanvasUID[canvasUID], u, &enrollStats)
			if err != nil {
				enrollFailed = true
				return err
			}
			if userID == uuid.Nil {
				continue
			}
			if err := canvasImportEnrollmentUserAvatar(ctx, enrollTx, client, accessToken, userID, e, u, &enrollStats); err != nil {
				enrollFailed = true
				return err
			}
			role := canvasEnrollmentTypeToRole(canvasEnrollmentTypeFromRow(e))
			if (include.Grades || include.Assignments) && canvasUID > 0 {
				if canvasUserToLocal == nil {
					canvasUserToLocal = make(map[int64]uuid.UUID)
				}
				canvasUserToLocal[canvasUID] = userID
			}
			var sectionID *uuid.UUID
			if canvasSectionMap != nil {
				if canvasSecID := canvasEnrollmentSectionID(e); canvasSecID > 0 {
					if sid, ok := canvasSectionMap[canvasSecID]; ok {
						sectionID = &sid
					}
				}
			}
			if err := canvasApplyEnrollment(ctx, enrollTx, courseID, courseCode, userID, role, sectionID, &enrollStats); err != nil {
				enrollFailed = true
				return errors.New("Failed to apply enrollment from Canvas.")
			}
			if sectionID != nil && (role == "teacher" || role == "instructor") {
				if err := canvasAssignSectionInstructor(ctx, enrollTx, canvasSectionMap, canvasEnrollmentSectionID(e), userID); err != nil {
					enrollFailed = true
					return errors.New("Failed to assign section instructor from Canvas.")
				}
			}
		}
		if !progress("Saving enrollments from Canvas...") {
			enrollFailed = true
			return context.Canceled
		}
		if err := enrollTx.Commit(ctx); err != nil {
			enrollFailed = true
			return errors.New("Something went wrong while saving enrollments.")
		}
		d.notifyEnrollmentsForCourse(ctx, courseCode)
		msg := fmt.Sprintf("Applied %d enrollment(s) from Canvas.", enrollStats.Enrolled)
		if enrollStats.AccountsCreated > 0 {
			msg += fmt.Sprintf(" Created %d new Lextures account(s) from Canvas emails.", enrollStats.AccountsCreated)
		}
		if enrollStats.AvatarsImported > 0 {
			msg += fmt.Sprintf(" Imported %d profile picture(s).", enrollStats.AvatarsImported)
		}
		if enrollStats.SkippedNoEmail > 0 {
			msg += fmt.Sprintf(" Skipped %d without an email in Canvas.", enrollStats.SkippedNoEmail)
		}
		if !progress(msg) {
			return context.Canceled
		}
	}

	needGradesTx := include.Grades || (include.Assignments && len(canvasAssignToItem) > 0)
	var tx pgx.Tx
	if needGradesTx {
		tx, err = d.Pool.Begin(ctx)
		if err != nil {
			return errors.New("Failed to start import transaction.")
		}
		defer func() { _ = tx.Rollback(ctx) }()
	}

	cfg := d.effectiveConfig()
	submissionDeps := &canvasAssignmentSubmissionImportDeps{
		CourseCode:     courseCode,
		ImporterUserID: importerUserID,
		FilesRoot:      cfg.CourseFilesRoot,
		Storage:        d.Storage,
	}
	if include.Grades {
		if canvasUserToLocal == nil {
			canvasUserToLocal = make(map[int64]uuid.UUID)
		}
		var gradeUserStats canvasEnrollmentImportStats
		if err := canvasFillGradeUserMap(ctx, d.Pool, tx, orgID, client, accessToken, enrollmentRows, rosterEmailByCanvasUID, canvasUserToLocal, &gradeUserStats); err != nil {
			return err
		}
		if !include.Enrollments && gradeUserStats.AccountsCreated > 0 {
			if !progress(fmt.Sprintf("Created %d Lextures account(s) to match Canvas grades.", gradeUserStats.AccountsCreated)) {
				return context.Canceled
			}
		}
		if !progress("Importing assignment grades and student submissions from Canvas...") {
			return context.Canceled
		}
		if !progress("Importing quiz attempt responses from Canvas...") {
			return context.Canceled
		}
		// #region agent log
		canvasAgentDebugLog("canvas-import", "H2", "canvas_import_ws.go:runCanvasImport", "invoking aggregated grade import (post-module maps)", map[string]any{
			"includeModules":     include.Modules,
			"includeAssignments": include.Assignments,
			"includeQuizzes":     include.Quizzes,
			"assignMapLen":       len(canvasAssignToItem),
			"quizMapLen":         len(canvasQuizToItem),
			"userMapLen":         len(canvasUserToLocal),
		})
		// #endregion agent log
		if err := canvasImportAllCanvasGrades(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasAssignToItem, canvasQuizToItem, canvasQuizToQuestions, canvasQuizToAssignmentID, canvasUserToLocal, submissionDeps); err != nil {
			return err
		}
	} else if include.Assignments && len(canvasAssignToItem) > 0 {
		if canvasUserToLocal == nil {
			canvasUserToLocal = make(map[int64]uuid.UUID)
		}
		if !progress("Importing student assignment submissions from Canvas...") {
			return context.Canceled
		}
		if err := canvasImportAssignmentGrades(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasAssignToItem, canvasUserToLocal, submissionDeps, false); err != nil {
			return err
		}
	}

	if needGradesTx {
		if !progress("Saving imported grades and submissions...") {
			return context.Canceled
		}
		if err = tx.Commit(ctx); err != nil {
			return errors.New("Something went wrong while saving the import.")
		}
	}

	var canvasFileIDs map[int64]uuid.UUID
	var canvasFileNames map[int64]string
	if include.Files {
		if !progress("Importing course files from Canvas...") {
			return context.Canceled
		}
		var fileCount int
		var fileErr error
		fileCount, canvasFileIDs, canvasFileNames, fileErr = d.importCanvasFiles(ctx, client, canvasBase, accessToken, canvasCourseID, courseID, courseCode, &importerUserID, progress)
		if fileErr != nil {
			// Non-fatal: log but let the import succeed overall
			log.Printf("canvas-import: file import failed course=%q err=%v", courseCode, fileErr)
			if !progress(fmt.Sprintf("Note: file import encountered an error: %v", fileErr)) {
				return context.Canceled
			}
		} else if fileCount > 0 {
			if !progress(fmt.Sprintf("Imported %d file(s) from Canvas.", fileCount)) {
				return context.Canceled
			}
		}
	}

	if include.Modules {
		if !progress("Updating internal links…") {
			return context.Canceled
		}
		enrichCanvasLinkMaps(ctx, client, canvasBase, accessToken, canvasCourseID, d.Pool, courseID,
			canvasAssignToItem, canvasQuizToItem, canvasPageSlugToItem)
		rc := &canvasLinkRewriteCtx{
			CanvasBase:            canvasBase,
			CanvasCourseID:        canvasCourseID,
			CourseCode:            courseCode,
			Assignments:           canvasAssignToItem,
			Quizzes:               canvasQuizToItem,
			PageSlugs:             canvasPageSlugToItem,
			FileIDs:               canvasFileIDs,
			FileNames:             canvasFileNames,
			AllowedHostSuffixes:   d.effectiveConfig().CanvasAllowedHostSuffixes,
		}
		if err := rewriteCanvasLinksInCourseMarkdown(ctx, d.Pool, courseID, rc); err != nil {
			log.Printf("canvas-import: link rewrite err=%v", err)
		}
	}

	return nil
}

func canvasHTTPClient() *http.Client {
	return &http.Client{Timeout: 180 * time.Second}
}

func normalizeCanvasBaseURL(raw string, allowedHostSuffixes []string) (string, error) {
	t := strings.TrimSpace(strings.TrimRight(raw, "/"))
	if t == "" {
		return "", errors.New("Canvas base URL is required.")
	}
	u, err := url.Parse(t)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("Canvas base URL must be a valid URL (https recommended).")
	}
	if u.Scheme != "https" {
		return "", errors.New("Canvas base URL must use https.")
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil {
		return "", errors.New("Canvas base URL must use a DNS hostname, not an IP address.")
	}
	ok := false
	for _, suffix := range allowedHostSuffixes {
		s := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(suffix), "*."), "."))
		if s != "" && (host == s || strings.HasSuffix(host, "."+s)) {
			ok = true
			break
		}
	}
	if !ok {
		return "", errors.New("Canvas base URL host is not allowed by server policy.")
	}
	return u.Scheme + "://" + u.Host, nil
}

func canvasGetArrayPaginated(ctx context.Context, client *http.Client, base, token, path string, q url.Values) ([]map[string]any, error) {
	out := make([]map[string]any, 0)
	for page := 1; ; page++ {
		qp := cloneQuery(q)
		qp.Set("per_page", "100")
		qp.Set("page", strconv.Itoa(page))
		arr, err := canvasGetArray(ctx, client, base, token, path, qp)
		if err != nil {
			return nil, err
		}
		if len(arr) == 0 {
			break
		}
		out = append(out, arr...)
		if len(arr) < 100 {
			break
		}
	}
	return out, nil
}

func canvasGetArray(ctx context.Context, client *http.Client, base, token, path string, q url.Values) ([]map[string]any, error) {
	v, err := canvasGetJSON(ctx, client, base, token, path, q)
	if err != nil {
		return nil, err
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, errors.New("Unexpected Canvas response (expected array).")
	}
	out := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func canvasGetObject(ctx context.Context, client *http.Client, base, token, path string, q url.Values) (map[string]any, error) {
	v, err := canvasGetJSON(ctx, client, base, token, path, q)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("Unexpected Canvas response (expected object).")
	}
	return m, nil
}

// canvasGetJSON is the only Canvas REST entry point. It must stay GET-only so imports never mutate Canvas.
func canvasGetJSON(ctx context.Context, client *http.Client, base, token, path string, q url.Values) (any, error) {
	u := fmt.Sprintf("%s/api/v1/%s", strings.TrimRight(base, "/"), strings.TrimLeft(path, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, errors.New("Failed to build Canvas request.")
	}
	if q != nil {
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Could not reach Canvas (network error). Check the base URL and try again.")
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("Canvas rejected the access token (401). Create a token with read access and try again.")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("Canvas returned 404 for this course or endpoint. Check the course id and token scope.")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Canvas API error HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, errors.New("Canvas returned invalid JSON.")
	}
	return out, nil
}

func cloneQuery(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}

func strAt(m map[string]any, k, def string) string {
	if m == nil {
		return def
	}
	if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func boolAt(m map[string]any, k string, def bool) bool {
	if m == nil {
		return def
	}
	if v, ok := m[k].(bool); ok {
		return v
	}
	return def
}

// canvasCanvasUserIDFromMap reads Canvas user id from submission/enrollment payloads that may
// nest the user object unless include[]=user was requested on the list endpoint.
func canvasCanvasUserIDFromMap(m map[string]any) int64 {
	if m == nil {
		return 0
	}
	if uid := int64At(m, "user_id"); uid > 0 {
		return uid
	}
	if u, ok := m["user"].(map[string]any); ok && u != nil {
		if uid := int64At(u, "id"); uid > 0 {
			return uid
		}
	}
	return 0
}

func int64At(m map[string]any, k string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

func canvasTimeAt(m map[string]any, k string) *time.Time {
	if m == nil {
		return nil
	}
	s, ok := m[k].(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &t
}

func objAt(m map[string]any, k string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[k].(map[string]any); ok {
		return v
	}
	return nil
}

func arrAt(m map[string]any, k string) []map[string]any {
	if m == nil {
		return nil
	}
	raw, ok := m[k].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, v := range raw {
		if mm, ok := v.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

var (
	canvasFileIframeRe = regexp.MustCompile(`(?is)<iframe\b[^>]*>`)
	canvasFileEmbedRe  = regexp.MustCompile(`(?is)<embed\b[^>]*>`)
	htmlMediaSrcRe     = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']([^"']+)["']`)
	htmlMediaTitleRe   = regexp.MustCompile(`(?i)\btitle\s*=\s*["']([^"']*)["']`)
)

// preprocessCanvasHTMLForMarkdown converts Canvas file embeds (iframes/embeds pointing at
// /courses/:id/files/:id) into anchor links so html-to-markdown preserves them.
func preprocessCanvasHTMLForMarkdown(s string) string {
	replaceMediaEmbed := func(tagRe *regexp.Regexp) {
		s = tagRe.ReplaceAllStringFunc(s, func(tag string) string {
			srcM := htmlMediaSrcRe.FindStringSubmatch(tag)
			if len(srcM) < 2 {
				return tag
			}
			href, title, ok := canvasEmbeddedFileLinkFromURL(srcM[1], tag)
			if !ok {
				return tag
			}
			return fmt.Sprintf(`<p><a href="%s">%s</a></p>`, html.EscapeString(href), html.EscapeString(title))
		})
	}
	replaceMediaEmbed(canvasFileIframeRe)
	replaceMediaEmbed(canvasFileEmbedRe)
	return s
}

func canvasEmbeddedFileLinkFromURL(rawURL, tag string) (href, title string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", false
	}
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}
	if !canvasFilePathRe.MatchString(path) {
		return "", "", false
	}
	href = path
	if u.RawQuery != "" {
		href += "?" + u.RawQuery
	}
	if u.IsAbs() {
		href = u.String()
	}
	titleM := htmlMediaTitleRe.FindStringSubmatch(tag)
	if len(titleM) >= 2 {
		title = strings.TrimSpace(titleM[1])
	}
	if title == "" {
		title = "Embedded file"
	}
	return href, title, true
}

func markdownFromHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = preprocessCanvasHTMLForMarkdown(s)
	converter := md.NewConverter("", true, nil)
	out, err := converter.ConvertString(s)
	if err == nil {
		out = strings.TrimSpace(out)
		if out != "" {
			return out
		}
	}
	return htmlToPlainText(s)
}

var (
	htmlBRTagRe  = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlPCloseRe = regexp.MustCompile(`(?i)</p\s*>`)
	htmlAnyTagRe = regexp.MustCompile(`<[^>]+>`)
)

func htmlToPlainText(html string) string {
	s := htmlBRTagRe.ReplaceAllString(html, "\n")
	s = htmlPCloseRe.ReplaceAllString(s, "\n\n")
	s = htmlAnyTagRe.ReplaceAllString(s, "")
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			if b.Len() == 0 || strings.HasSuffix(b.String(), "\n\n") {
				continue
			}
			b.WriteString("\n")
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
		b.WriteString(t)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// ── Canvas link rewriting ─────────────────────────────────────────────────────

// canvasLinkRewriteCtx holds ID mappings used to convert Canvas-internal URLs
// embedded in imported content into their Lextures equivalents.
type canvasLinkRewriteCtx struct {
	CanvasBase            string
	CanvasCourseID        int64
	CourseCode            string
	Assignments           map[int64]uuid.UUID // canvas assignment ID → lextures item UUID
	Quizzes               map[int64]uuid.UUID // canvas quiz ID → lextures item UUID
	PageSlugs             map[string]uuid.UUID // canvas page_url slug → lextures item UUID
	FileIDs               map[int64]uuid.UUID // canvas file ID → lextures file-item UUID
	FileNames             map[int64]string   // canvas file ID → display name
	AllowedHostSuffixes   []string
}

var (
	canvasFilePathRe    = regexp.MustCompile(`(?i)^/courses/(\d+)/files/(\d+)`)
	canvasAssignPathRe  = regexp.MustCompile(`(?i)^/courses/(\d+)/assignments/(\d+)`)
	canvasQuizPathRe    = regexp.MustCompile(`(?i)^/courses/(\d+)/quizzes/(\d+)`)
	canvasPagePathRe    = regexp.MustCompile(`(?i)^/courses/(\d+)/pages/([^/?#\s]+)`)
	canvasModulePathRe  = regexp.MustCompile(`(?i)^/courses/(\d+)/modules`)
	markdownLinkRe      = regexp.MustCompile(`(!?)\[([^\]]*)\]\(([^)]+)\)`)
	markdownAngleLinkRe = regexp.MustCompile(`<(https?://[^>\s]+)>`)
	htmlAnchorTagRe     = regexp.MustCompile(`(?i)<a\s([^>]*?)>`)
	htmlAnchorHrefRe    = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
)

func normalizeLinkMatchTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func canvasURLHostMatches(host, canvasBase string, allowedHostSuffixes []string) bool {
	if host == "" {
		return true
	}
	base, err := url.Parse(canvasBase)
	if err == nil && strings.EqualFold(host, base.Host) {
		return true
	}
	hostname := strings.ToLower(host)
	if i := strings.Index(hostname, ":"); i >= 0 {
		hostname = hostname[:i]
	}
	for _, suffix := range allowedHostSuffixes {
		s := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(suffix), "*."), "."))
		if s != "" && (hostname == s || strings.HasSuffix(hostname, "."+s)) {
			return true
		}
	}
	return false
}

// enrichCanvasLinkMaps fills Canvas→Lextures ID maps using the full Canvas course
// catalog matched to existing module items by title (and page slug). Module import
// only records content_id for module items; links often target other course resources.
func enrichCanvasLinkMaps(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	assignments, quizzes map[int64]uuid.UUID,
	pageSlugs map[string]uuid.UUID,
) {
	rows, err := pool.Query(ctx, `
		SELECT id, kind, title FROM course.course_structure_items
		WHERE course_id = $1 AND archived = false
		  AND kind IN ('assignment', 'quiz', 'content_page')`,
		courseID)
	if err != nil {
		log.Printf("canvas-link-rewrite: load local items: %v", err)
		return
	}
	defer rows.Close()
	byKindTitle := make(map[string]map[string][]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var kind, title string
		if err := rows.Scan(&id, &kind, &title); err != nil {
			continue
		}
		t := normalizeLinkMatchTitle(title)
		if t == "" {
			continue
		}
		if byKindTitle[kind] == nil {
			byKindTitle[kind] = make(map[string][]uuid.UUID)
		}
		byKindTitle[kind][t] = append(byKindTitle[kind][t], id)
	}
	uniqueByKindTitle := func(kind, title string) (uuid.UUID, bool) {
		ids := byKindTitle[kind][normalizeLinkMatchTitle(title)]
		if len(ids) != 1 {
			return uuid.Nil, false
		}
		return ids[0], true
	}

	var assignRows, quizRows, pageRows []map[string]any
	linkGroup, linkCtx := canvasImportParallelGroup(ctx, 3)
	linkGroup.Go(func() error {
		rows, listErr := canvasGetArrayPaginated(linkCtx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/assignments", canvasCourseID), nil)
		if listErr != nil {
			log.Printf("canvas-link-rewrite: list assignments: %v", listErr)
			return nil
		}
		assignRows = rows
		return nil
	})
	linkGroup.Go(func() error {
		rows, listErr := canvasGetArrayPaginated(linkCtx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/quizzes", canvasCourseID), nil)
		if listErr != nil {
			log.Printf("canvas-link-rewrite: list quizzes: %v", listErr)
			return nil
		}
		quizRows = rows
		return nil
	})
	linkGroup.Go(func() error {
		rows, listErr := canvasGetArrayPaginated(linkCtx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/pages", canvasCourseID), nil)
		if listErr != nil {
			log.Printf("canvas-link-rewrite: list pages: %v", listErr)
			return nil
		}
		pageRows = rows
		return nil
	})
	_ = linkGroup.Wait()

	for _, a := range assignRows {
		cid := int64At(a, "id")
		if cid <= 0 {
			continue
		}
		if _, ok := assignments[cid]; ok {
			continue
		}
		if id, ok := uniqueByKindTitle("assignment", strAt(a, "name", "")); ok {
			assignments[cid] = id
		}
	}
	for _, q := range quizRows {
		cid := int64At(q, "id")
		if cid <= 0 {
			continue
		}
		if _, ok := quizzes[cid]; ok {
			continue
		}
		if id, ok := uniqueByKindTitle("quiz", strAt(q, "title", "")); ok {
			quizzes[cid] = id
		}
	}
	for _, p := range pageRows {
		slug := strings.ToLower(strings.TrimSpace(strAt(p, "url", "")))
		if slug == "" {
			continue
		}
		if _, ok := pageSlugs[slug]; ok {
			continue
		}
		if id, ok := uniqueByKindTitle("content_page", strAt(p, "title", "")); ok {
			pageSlugs[slug] = id
			continue
		}
		if id, ok := uniqueByKindTitle("content_page", slug); ok {
			pageSlugs[slug] = id
		}
	}
}

// rewriteURL returns the Lextures equivalent of a Canvas-internal URL, or the
// original URL unchanged if it points to a different domain or resource type.
func (rc *canvasLinkRewriteCtx) rewriteURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// For absolute URLs, only process those pointing at an allowed Canvas host.
	if u.Host != "" && !canvasURLHostMatches(u.Host, rc.CanvasBase, rc.AllowedHostSuffixes) {
		return rawURL
	}
	path := u.Path

	if m := canvasFilePathRe.FindStringSubmatch(path); m != nil {
		cid, _ := strconv.ParseInt(m[1], 10, 64)
		if cid != rc.CanvasCourseID {
			return rawURL
		}
		fid, _ := strconv.ParseInt(m[2], 10, 64)
		if localID, ok := rc.FileIDs[fid]; ok {
			target := fmt.Sprintf("/api/v1/courses/%s/files/items/%s/content",
				url.PathEscape(rc.CourseCode), localID)
			if name := rc.FileNames[fid]; name != "" {
				target += "?name=" + url.QueryEscape(name)
			}
			return target
		}
		return rawURL
	}

	if m := canvasAssignPathRe.FindStringSubmatch(path); m != nil {
		cid, _ := strconv.ParseInt(m[1], 10, 64)
		if cid != rc.CanvasCourseID {
			return rawURL
		}
		aid, _ := strconv.ParseInt(m[2], 10, 64)
		if localID, ok := rc.Assignments[aid]; ok {
			return fmt.Sprintf("/courses/%s/modules/assignment/%s",
				url.PathEscape(rc.CourseCode), localID)
		}
		return rawURL
	}

	if m := canvasQuizPathRe.FindStringSubmatch(path); m != nil {
		cid, _ := strconv.ParseInt(m[1], 10, 64)
		if cid != rc.CanvasCourseID {
			return rawURL
		}
		qid, _ := strconv.ParseInt(m[2], 10, 64)
		if localID, ok := rc.Quizzes[qid]; ok {
			return fmt.Sprintf("/courses/%s/modules/quiz/%s",
				url.PathEscape(rc.CourseCode), localID)
		}
		return rawURL
	}

	if m := canvasPagePathRe.FindStringSubmatch(path); m != nil {
		cid, _ := strconv.ParseInt(m[1], 10, 64)
		if cid != rc.CanvasCourseID {
			return rawURL
		}
		slug, _ := url.PathUnescape(m[2])
		slug = strings.ToLower(strings.TrimSpace(slug))
		if localID, ok := rc.PageSlugs[slug]; ok {
			return fmt.Sprintf("/courses/%s/modules/content/%s",
				url.PathEscape(rc.CourseCode), localID)
		}
		return rawURL
	}

	if m := canvasModulePathRe.FindStringSubmatch(path); m != nil {
		cid, _ := strconv.ParseInt(m[1], 10, 64)
		if cid == rc.CanvasCourseID {
			return fmt.Sprintf("/courses/%s/modules", url.PathEscape(rc.CourseCode))
		}
	}

	return rawURL
}

// rewriteMarkdown rewrites Markdown links, autolinks, and any leftover HTML
// anchors that point into the Canvas course to their Lextures equivalents.
func (rc *canvasLinkRewriteCtx) rewriteMarkdown(markdown string) string {
	s := markdownLinkRe.ReplaceAllStringFunc(markdown, func(match string) string {
		subs := markdownLinkRe.FindStringSubmatch(match)
		if len(subs) < 4 {
			return match
		}
		bang, text, rawURL := subs[1], subs[2], subs[3]
		newURL := rc.rewriteURL(rawURL)
		if newURL == rawURL {
			return match
		}
		return fmt.Sprintf("%s[%s](%s)", bang, text, newURL)
	})
	s = markdownAngleLinkRe.ReplaceAllStringFunc(s, func(match string) string {
		subs := markdownAngleLinkRe.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		newURL := rc.rewriteURL(subs[1])
		if newURL == subs[1] {
			return match
		}
		return "<" + newURL + ">"
	})
	return htmlAnchorTagRe.ReplaceAllStringFunc(s, func(tag string) string {
		return htmlAnchorHrefRe.ReplaceAllStringFunc(tag, func(hrefPart string) string {
			subs := htmlAnchorHrefRe.FindStringSubmatch(hrefPart)
			if len(subs) < 2 {
				return hrefPart
			}
			newURL := rc.rewriteURL(subs[1])
			if newURL == subs[1] {
				return hrefPart
			}
			return `href="` + newURL + `"`
		})
	})
}

// rewriteCanvasLinksInCourseMarkdown updates all stored markdown bodies for the
// given course, rewriting Canvas-internal URLs to Lextures equivalents.
// Individual row failures are logged but do not abort the operation.
func rewriteCanvasLinksInCourseMarkdown(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	rc *canvasLinkRewriteCtx,
) error {
	if len(rc.Assignments) == 0 && len(rc.Quizzes) == 0 &&
		len(rc.PageSlugs) == 0 && len(rc.FileIDs) == 0 {
		return nil
	}

	type rowUpdate struct {
		id uuid.UUID
		md string
	}

	rewriteTable := func(table, idCol, mdCol string) {
		query := fmt.Sprintf(
			`SELECT t.%s, t.%s FROM %s t
			 JOIN course.course_structure_items csi ON csi.id = t.%s
			 WHERE csi.course_id = $1`, idCol, mdCol, table, idCol)
		rows, err := pool.Query(ctx, query, courseID)
		if err != nil {
			log.Printf("canvas-link-rewrite: query %s: %v", table, err)
			return
		}
		var updates []rowUpdate
		for rows.Next() {
			var id uuid.UUID
			var md string
			if err := rows.Scan(&id, &md); err != nil {
				continue
			}
			if newMD := rc.rewriteMarkdown(md); newMD != md {
				updates = append(updates, rowUpdate{id, newMD})
			}
		}
		rows.Close()
		upd := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`, table, mdCol, idCol)
		for _, u := range updates {
			if _, err := pool.Exec(ctx, upd, u.md, u.id); err != nil {
				log.Printf("canvas-link-rewrite: update %s id=%s: %v", table, u.id, err)
			}
		}
	}

	rewriteTable("course.module_content_pages", "structure_item_id", "markdown")
	rewriteTable("course.module_assignments", "structure_item_id", "markdown")
	rewriteTable("course.module_quizzes", "structure_item_id", "markdown")
	rewriteSyllabusMarkdown(ctx, pool, courseID, rc)
	return nil
}

// rewriteSyllabusMarkdown updates the markdown fields inside the JSON sections
// blob stored in course.course_syllabus.
func rewriteSyllabusMarkdown(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, rc *canvasLinkRewriteCtx) {
	var sectionsRaw []byte
	if err := pool.QueryRow(ctx, `SELECT sections FROM course.course_syllabus WHERE course_id = $1`, courseID).Scan(&sectionsRaw); err != nil || len(sectionsRaw) == 0 {
		return
	}
	var sections []map[string]any
	if err := json.Unmarshal(sectionsRaw, &sections); err != nil {
		return
	}
	changed := false
	for _, sec := range sections {
		if md, ok := sec["markdown"].(string); ok {
			if newMD := rc.rewriteMarkdown(md); newMD != md {
				sec["markdown"] = newMD
				changed = true
			}
		}
	}
	if !changed {
		return
	}
	newJSON, err := json.Marshal(sections)
	if err != nil {
		return
	}
	if _, err := pool.Exec(ctx, `UPDATE course.course_syllabus SET sections = $1 WHERE course_id = $2`, newJSON, courseID); err != nil {
		log.Printf("canvas-link-rewrite: update syllabus course=%s: %v", courseID, err)
	}
}

// ── End Canvas link rewriting ─────────────────────────────────────────────────

// canvasEnrollmentTypeToRole converts a Canvas enrollment type string (e.g.
// "TeacherEnrollment", "TaEnrollment") to the Lextures course role it maps to.
func canvasEnrollmentTypeToRole(canvasType string) string {
	t := strings.ToLower(canvasType)
	if strings.Contains(t, "teacher") {
		return "teacher"
	}
	if strings.Contains(t, "ta") {
		return "instructor"
	}
	return "student"
}

func mapCanvasTypeToKind(t string) (kind string, bodyTable string) {
	switch t {
	case "SubHeader":
		return "heading", ""
	case "Page":
		return "content_page", "content"
	case "Assignment":
		return "assignment", "assignment"
	case "Quiz":
		return "quiz", "quiz"
	case "File":
		return "content_page", "content"
	case "ExternalUrl", "ExternalTool":
		return "external_link", "external"
	case "Discussion":
		return "content_page", "content"
	default:
		return "", ""
	}
}

// importCanvasFiles downloads all files from a Canvas course and stores them in the
// course's file manager (course.file_folders + course.file_items). Returns the count
// of files successfully imported, a mapping of canvas file ID → local file-item UUID,
// and a mapping of canvas file ID → display name (for link rewriting).
func (d Deps) importCanvasFiles(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	courseCode string,
	importerUserID *uuid.UUID,
	progress func(string) bool,
) (int, map[int64]uuid.UUID, map[int64]string, error) {
	// Fetch all folders
	folderRows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/folders", canvasCourseID), nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("fetching Canvas folders: %w", err)
	}

	// Map canvas folder ID -> our UUID; skip the root "course files" folder
	canvasFolderToLocal := make(map[int64]uuid.UUID)
	// Sort by full_name length so parents are created before children
	for i := 0; i < len(folderRows); i++ {
		for j := i + 1; j < len(folderRows); j++ {
			if len(strAt(folderRows[i], "full_name", "")) > len(strAt(folderRows[j], "full_name", "")) {
				folderRows[i], folderRows[j] = folderRows[j], folderRows[i]
			}
		}
	}
	for _, f := range folderRows {
		fullName := strAt(f, "full_name", "")
		name := strAt(f, "name", "folder")
		canvasID := int64At(f, "id")
		if canvasID == 0 {
			continue
		}
		// The root canvas folder ("course files") maps to our virtual root (nil parent)
		if strings.EqualFold(fullName, "course files") || strings.EqualFold(name, "course files") {
			canvasFolderToLocal[canvasID] = uuid.Nil // sentinel: this is the root
			continue
		}
		parentCanvasID := int64At(f, "parent_folder_id")
		var localParentID *uuid.UUID
		if parentCanvasID != 0 {
			if pid, ok := canvasFolderToLocal[parentCanvasID]; ok && pid != uuid.Nil {
				localParentID = &pid
			}
			// If parent is root (uuid.Nil sentinel), localParentID stays nil → top-level folder
		}
		folder, createErr := filemanager.CreateFolder(ctx, d.Pool, courseID, localParentID, name, importerUserID)
		if createErr != nil {
			log.Printf("canvas-import-files: create folder %q err=%v", name, createErr)
			continue
		}
		canvasFolderToLocal[canvasID] = folder.ID
	}

	// Fetch all files
	fileRows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/files", canvasCourseID), url.Values{"include[]": []string{"user"}})
	if err != nil {
		return 0, nil, nil, fmt.Errorf("fetching Canvas files: %w", err)
	}

	cfg := d.effectiveConfig()
	imported := 0
	canvasFileIDs := make(map[int64]uuid.UUID)
	canvasFileNames := make(map[int64]string)
	var fileMapsMu sync.Mutex
	var importedMu sync.Mutex

	fileGroup, fileCtx := canvasImportParallelGroup(ctx, len(fileRows))
	for _, f := range fileRows {
		f := f
		fileGroup.Go(func() error {
			canvasFileID := int64At(f, "id")
			if canvasFileID == 0 {
				return nil
			}
			displayName := strAt(f, "display_name", strAt(f, "filename", "file"))
			filename := strAt(f, "filename", displayName)
			mimeType := strAt(f, "content-type", "application/octet-stream")
			fileSize := int64At(f, "size")
			downloadURL := strAt(f, "url", "")
			if downloadURL == "" {
				return nil
			}
			canvasFolderID := int64At(f, "folder_id")
			var localFolderID *uuid.UUID
			if canvasFolderID != 0 {
				if fid, ok := canvasFolderToLocal[canvasFolderID]; ok && fid != uuid.Nil {
					localFolderID = &fid
				}
			}
			ext := filepath.Ext(filename)
			objectKey := fmt.Sprintf("managed-files/%s/%s%s", courseCode, uuid.New().String(), ext)

			req, reqErr := http.NewRequestWithContext(fileCtx, http.MethodGet, downloadURL, nil)
			if reqErr != nil {
				log.Printf("canvas-import-files: build request file=%d err=%v", canvasFileID, reqErr)
				return nil
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)
			resp, dlErr := client.Do(req)
			if dlErr != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
				if resp != nil {
					_ = resp.Body.Close()
				}
				log.Printf("canvas-import-files: download file=%d status=%v err=%v", canvasFileID, resp, dlErr)
				return nil
			}

			if d.Storage != nil {
				storeErr := d.Storage.PutObject(fileCtx, objectKey, resp.Body, fileSize, mimeType)
				_ = resp.Body.Close()
				if storeErr != nil {
					log.Printf("canvas-import-files: store file=%d key=%q err=%v", canvasFileID, objectKey, storeErr)
					return nil
				}
			} else {
				root := strings.TrimSpace(cfg.CourseFilesRoot)
				if root == "" {
					root = "data/course-files"
				}
				p := root + "/" + courseCode + "/" + objectKey
				if writeErr := writeLocalFile(p, resp.Body); writeErr != nil {
					_ = resp.Body.Close()
					log.Printf("canvas-import-files: write file=%d path=%q err=%v", canvasFileID, p, writeErr)
					return nil
				}
				_ = resp.Body.Close()
			}

			fi, dbErr := filemanager.CreateFileItemWithCanvas(
				fileCtx, d.Pool, courseID, localFolderID,
				objectKey, filename, displayName, mimeType, fileSize, importerUserID, canvasFileID,
			)
			if dbErr != nil {
				log.Printf("canvas-import-files: db insert file=%d err=%v", canvasFileID, dbErr)
				return nil
			}

			fileMapsMu.Lock()
			canvasFileIDs[canvasFileID] = fi.ID
			canvasFileNames[canvasFileID] = displayName
			fileMapsMu.Unlock()
			broadcastFilesChanged(courseCode)

			n := 0
			importedMu.Lock()
			imported++
			n = imported
			importedMu.Unlock()
			if n%5 == 0 {
				if !progress(fmt.Sprintf("Importing files… (%d so far)", n)) {
					return context.Canceled
				}
			}
			return nil
		})
	}
	if err := fileGroup.Wait(); err != nil {
		return imported, canvasFileIDs, canvasFileNames, err
	}
	if imported > 0 {
		if !progress(fmt.Sprintf("Importing files… (%d so far)", imported)) {
			return imported, canvasFileIDs, canvasFileNames, context.Canceled
		}
	}
	return imported, canvasFileIDs, canvasFileNames, nil
}
