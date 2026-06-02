package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log"
	"path/filepath"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
)

// handleCourseImportCanvasWS is GET /api/v1/courses/{course_code}/import/canvas/ws.
func (d Deps) handleCourseImportCanvasWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil || d.Pool == nil {
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
		var m struct {
			AuthToken string `json:"authToken"`
		}
		if err := json.Unmarshal(b, &m); err != nil || m.AuthToken == "" {
			return
		}
		u, err := d.JWTSigner.Verify(r.Context(), m.AuthToken)
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

		var req canvasImportWSFirstMessage
		if err := json.Unmarshal(b, &req); err != nil {
			_ = wsWriteJSON(r.Context(), c, map[string]any{
				"type":    "error",
				"message": "Invalid JSON in the first message. Send authToken plus the former Canvas import POST body fields.",
			})
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
		emit := func(msg string) bool {
			return wsWriteJSON(r.Context(), c, map[string]any{"type": "progress", "message": msg}) == nil
		}
		if !emit("Connecting to Canvas...") {
			return
		}
		err = d.runCanvasImport(r.Context(), uid, courseCode, req.Mode, req.CanvasBaseURL, req.CanvasCourseID, req.AccessToken, include, emit)
		if err != nil {
			_ = wsWriteJSON(r.Context(), c, map[string]any{"type": "error", "message": err.Error()})
			return
		}
		_ = wsWriteJSON(r.Context(), c, map[string]any{"type": "complete"})
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

	course, err := canvasGetObject(ctx, client, canvasBase, accessToken, fmt.Sprintf("courses/%d", canvasCourseID), url.Values{"include[]": []string{"syllabus_body"}})
	if err != nil {
		return err
	}
	if !progress("Loaded course details from Canvas.") {
		return context.Canceled
	}
	modules := []map[string]any{}
	if include.Modules {
		if !progress("Loading modules from Canvas...") {
			return context.Canceled
		}
		modules, err = canvasGetArrayPaginated(ctx, client, canvasBase, accessToken, fmt.Sprintf("courses/%d/modules", canvasCourseID), url.Values{"include[]": []string{"items"}})
		if err != nil {
			return err
		}
	}
	enrollmentRows := []map[string]any{}
	rosterEmailByCanvasUID := make(map[int64]string)
	needEnrollmentRows := include.Enrollments || include.Grades
	if needEnrollmentRows {
		if !progress("Loading Canvas enrollments...") {
			return context.Canceled
		}
		enrollmentRows, err = canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/enrollments", canvasCourseID), canvasEnrollmentListQuery())
		if err != nil {
			return err
		}
		rosterEmailByCanvasUID, err = canvasRosterEmailsByCanvasUserID(ctx, client, canvasBase, accessToken, canvasCourseID)
		if err != nil {
			return err
		}
	}

	var canvasUserToLocal map[int64]uuid.UUID
	if include.Grades {
		canvasUserToLocal = buildCanvasUserIDToLexturesUserID(ctx, d.Pool, client, canvasBase, accessToken, canvasCourseID, enrollmentRows, rosterEmailByCanvasUID)
	}

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return errors.New("Failed to start import transaction.")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var courseID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM course.courses WHERE course_code = $1`, courseCode).Scan(&courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("Course not found or you do not have access.")
	}
	if err != nil {
		return errors.New("Failed to load course.")
	}
	var orgID uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID); err != nil {
		return errors.New("Failed to load course organization.")
	}

	if include.Settings {
		title := strAt(course, "name", "Imported Canvas course")
		// Avoid stuffing the syllabus (or HTML public description) into the short course
		// blurb—the syllabus still lands on the dedicated syllabus record below.
		desc := title
		published := strAt(course, "workflow_state", "available") == "available"
		_, err = tx.Exec(ctx, `UPDATE course.courses SET title = $1, description = $2, published = $3, updated_at = NOW() WHERE id = $4`, title, desc, published, courseID)
		if err != nil {
			return errors.New("Failed to update course settings.")
		}
		syllabusHTML := strAt(course, "syllabus_body", "")
		if syllabusHTML != "" {
			sections, _ := json.Marshal([]map[string]string{{
				"id":       "canvas-syllabus",
				"heading":  "Syllabus",
				"markdown": markdownFromHTML(syllabusHTML),
			}})
			_, err = tx.Exec(ctx, `
				INSERT INTO course.course_syllabus (course_id, sections, require_syllabus_acceptance, updated_at)
				VALUES ($1, $2, false, NOW())
				ON CONFLICT (course_id) DO UPDATE SET sections = EXCLUDED.sections, updated_at = NOW()
			`, courseID, sections)
			if err != nil {
				return errors.New("Failed to update syllabus.")
			}
		}
	}

	if include.Modules && (mode == "erase" || mode == "overwrite") {
		if !progress("Clearing existing course modules...") {
			return context.Canceled
		}
		if _, err = tx.Exec(ctx, `DELETE FROM course.course_structure_items WHERE course_id = $1`, courseID); err != nil {
			return errors.New("Failed to clear existing module structure.")
		}
	}

	nextSort := 0
	_ = tx.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order), -1) + 1 FROM course.course_structure_items WHERE course_id = $1`, courseID).Scan(&nextSort)
	canvasAssignToItem := make(map[int64]uuid.UUID)
	canvasQuizToItem := make(map[int64]uuid.UUID)
	if include.Modules {
		if !progress("Importing modules and items...") {
			return context.Canceled
		}
		for _, m := range modules {
			moduleID := uuid.New()
			title := strAt(m, "name", "Module")
			published := boolAt(m, "published", true)
			if _, err = tx.Exec(ctx, `
				INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
				VALUES ($1, $2, $3, 'module', $4, NULL, $5, false)
			`, moduleID, courseID, nextSort, title, published); err != nil {
				return errors.New("Failed to insert module item.")
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
				if _, err = tx.Exec(ctx, `
					INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
					VALUES ($1, $2, $3, $4, $5, $6, $7, false)
				`, itemID, courseID, nextSort, kind, itemTitle, moduleID, itemPublished); err != nil {
					return errors.New("Failed to insert module child item.")
				}
				nextSort++
				switch bodyTable {
				case "content":
					md := ""
					if kind == "content_page" {
						pageURL := strAt(it, "page_url", "")
						if pageURL != "" {
							page, e := canvasGetObject(ctx, client, canvasBase, accessToken, fmt.Sprintf("courses/%d/pages/%s", canvasCourseID, url.PathEscape(pageURL)), nil)
							if e == nil {
								md = markdownFromHTML(strAt(page, "body", ""))
							}
						}
					} else {
						link := strAt(it, "html_url", "")
						if link != "" {
							md = fmt.Sprintf("**%s**\n\n[Open in Canvas](%s)", itemTitle, link)
						}
					}
					if _, err = tx.Exec(ctx, `INSERT INTO course.module_content_pages (structure_item_id, markdown) VALUES ($1, $2)`, itemID, md); err != nil {
						return errors.New("Failed to save imported page content.")
					}
				case "assignment":
					markdown := ""
					var pointsWorth *int
					if cid := int64At(it, "content_id"); cid > 0 {
						canvasAssignToItem[cid] = itemID
						obj, e := canvasGetObject(ctx, client, canvasBase, accessToken, fmt.Sprintf("courses/%d/assignments/%d", canvasCourseID, cid), nil)
						if e == nil && obj != nil {
							markdown = markdownFromHTML(strAt(obj, "description", ""))
							pointsWorth = optionalPointsWorthFromCanvas(obj, "points_possible")
						}
					}
					if _, err = tx.Exec(ctx, `INSERT INTO course.module_assignments (structure_item_id, markdown, points_worth) VALUES ($1, $2, $3)`, itemID, markdown, pointsWorth); err != nil {
						return errors.New("Failed to save imported assignment.")
					}
				case "quiz":
					markdown := ""
					var questions []coursemodulequiz.QuizQuestion
					var pointsWorth *int
					if cid := int64At(it, "content_id"); cid > 0 {
						canvasQuizToItem[cid] = itemID
						obj, e := canvasGetObject(ctx, client, canvasBase, accessToken, fmt.Sprintf("courses/%d/quizzes/%d", canvasCourseID, cid), nil)
						if e == nil && obj != nil {
							markdown = markdownFromHTML(strAt(obj, "description", ""))
							pointsWorth = optionalPointsWorthFromCanvas(obj, "points_possible")
						}
						qq, qe := canvasImportQuizQuestions(ctx, client, canvasBase, accessToken, canvasCourseID, cid)
						if qe != nil {
							return fmt.Errorf("Failed to load quiz questions from Canvas (quiz id %d): %w", cid, qe)
						}
						questions = qq
					}
					qJSON, mj := json.Marshal(questions)
					if mj != nil {
						return errors.New("Failed to encode imported quiz questions.")
					}
					if _, err = tx.Exec(ctx, `INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json, points_worth) VALUES ($1, $2, $3, $4)`, itemID, markdown, qJSON, pointsWorth); err != nil {
						return errors.New("Failed to save imported quiz.")
					}
				case "external":
					raw := strAt(it, "external_url", "")
					if raw == "" {
						raw = strAt(it, "html_url", "")
					}
					if _, err = tx.Exec(ctx, `INSERT INTO course.module_external_links (structure_item_id, url) VALUES ($1, $2)`, itemID, raw); err != nil {
						return errors.New("Failed to save imported external link.")
					}
				}
			}
		}
	}

	if include.Enrollments {
		if !progress("Applying enrollments from Canvas...") {
			return context.Canceled
		}
		var enrollStats canvasEnrollmentImportStats
		for _, e := range enrollmentRows {
			u := objAt(e, "user")
			canvasUID := int64At(u, "id")
			email := rosterEmailByCanvasUID[canvasUID]
			if email == "" {
				email = normalizedLexturesEmailGuessFromCanvasUserMap(u)
			}
			userID, err := canvasResolveLexturesUserForEnrollment(ctx, d.Pool, tx, orgID, email, u, &enrollStats)
			if err != nil {
				return err
			}
			if userID == uuid.Nil {
				continue
			}
			role := canvasEnrollmentTypeToRole(strAt(e, "type", ""))
			if include.Grades && canvasUID > 0 {
				canvasUserToLocal[canvasUID] = userID
			}
			if err := canvasApplyEnrollment(ctx, tx, courseID, courseCode, userID, role, &enrollStats); err != nil {
				return errors.New("Failed to apply enrollment from Canvas.")
			}
		}
		msg := fmt.Sprintf("Applied %d enrollment(s) from Canvas.", enrollStats.Enrolled)
		if enrollStats.AccountsCreated > 0 {
			msg += fmt.Sprintf(" Created %d new Lextures account(s) from Canvas emails.", enrollStats.AccountsCreated)
		}
		if enrollStats.SkippedNoEmail > 0 {
			msg += fmt.Sprintf(" Skipped %d without an email in Canvas.", enrollStats.SkippedNoEmail)
		}
		if !progress(msg) {
			return context.Canceled
		}
	}

	if include.Grades {
		if canvasUserToLocal == nil {
			canvasUserToLocal = make(map[int64]uuid.UUID)
		}
		var gradeUserStats canvasEnrollmentImportStats
		if err := canvasFillGradeUserMap(ctx, d.Pool, tx, orgID, enrollmentRows, rosterEmailByCanvasUID, canvasUserToLocal, &gradeUserStats); err != nil {
			return err
		}
		if !include.Enrollments && gradeUserStats.AccountsCreated > 0 {
			if !progress(fmt.Sprintf("Created %d Lextures account(s) to match Canvas grades.", gradeUserStats.AccountsCreated)) {
				return context.Canceled
			}
		}
		if !progress("Importing assignment and quiz grades from Canvas...") {
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
		if err := canvasImportAllCanvasGrades(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasAssignToItem, canvasQuizToItem, canvasUserToLocal); err != nil {
			return err
		}
	}

	if !progress("Saving imported content into your course...") {
		return context.Canceled
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.New("Something went wrong while saving the import.")
	}

	if include.Files {
		if !progress("Importing course files from Canvas...") {
			return context.Canceled
		}
		fileCount, fileErr := d.importCanvasFiles(ctx, client, canvasBase, accessToken, canvasCourseID, courseID, courseCode, &importerUserID, progress)
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

	courseTitle := strAt(course, "name", "")
	if courseTitle == "" {
		_ = d.Pool.QueryRow(ctx, `SELECT title FROM course.courses WHERE id = $1`, courseID).Scan(&courseTitle)
	}
	if courseTitle == "" {
		courseTitle = courseCode
	}
	d.pushNotificationService().EnqueueCanvasCourseImported(ctx, importerUserID, courseTitle, courseCode)

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

func markdownFromHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
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

// canvasEnrollmentTypeToRole converts a Canvas enrollment type string (e.g.
// "TeacherEnrollment", "TaEnrollment") to the Lextures course role it maps to.
func canvasEnrollmentTypeToRole(canvasType string) string {
	t := strings.ToLower(canvasType)
	if strings.Contains(t, "teacher") || strings.Contains(t, "ta") {
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
	case "ExternalUrl", "ExternalTool", "File":
		return "external_link", "external"
	case "Discussion":
		return "content_page", "content"
	default:
		return "", ""
	}
}

// importCanvasFiles downloads all files from a Canvas course and stores them in the
// course's file manager (course.file_folders + course.file_items). Returns the count
// of files successfully imported.
func (d Deps) importCanvasFiles(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	courseCode string,
	importerUserID *uuid.UUID,
	progress func(string) bool,
) (int, error) {
	// Fetch all folders
	folderRows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/folders", canvasCourseID), nil)
	if err != nil {
		return 0, fmt.Errorf("fetching Canvas folders: %w", err)
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
		return 0, fmt.Errorf("fetching Canvas files: %w", err)
	}

	cfg := d.effectiveConfig()
	imported := 0
	for _, f := range fileRows {
		canvasFileID := int64At(f, "id")
		if canvasFileID == 0 {
			continue
		}
		displayName := strAt(f, "display_name", strAt(f, "filename", "file"))
		filename := strAt(f, "filename", displayName)
		mimeType := strAt(f, "content-type", "application/octet-stream")
		fileSize := int64At(f, "size")
		downloadURL := strAt(f, "url", "")
		if downloadURL == "" {
			continue
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

		// Download from Canvas
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if reqErr != nil {
			log.Printf("canvas-import-files: build request file=%d err=%v", canvasFileID, reqErr)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, dlErr := client.Do(req)
		if dlErr != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			log.Printf("canvas-import-files: download file=%d status=%v err=%v", canvasFileID, resp, dlErr)
			continue
		}

		// Store blob
		if d.Storage != nil {
			storeErr := d.Storage.PutObject(ctx, objectKey, resp.Body, fileSize, mimeType)
			_ = resp.Body.Close()
			if storeErr != nil {
				log.Printf("canvas-import-files: store file=%d key=%q err=%v", canvasFileID, objectKey, storeErr)
				continue
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
				continue
			}
			_ = resp.Body.Close()
		}

		// Register metadata
		_, dbErr := filemanager.CreateFileItemWithCanvas(
			ctx, d.Pool, courseID, localFolderID,
			objectKey, filename, displayName, mimeType, fileSize, importerUserID, canvasFileID,
		)
		if dbErr != nil {
			log.Printf("canvas-import-files: db insert file=%d err=%v", canvasFileID, dbErr)
			continue
		}
		imported++

		if !progress(fmt.Sprintf("Importing files… (%d so far)", imported)) {
			return imported, context.Canceled
		}
	}
	return imported, nil
}
