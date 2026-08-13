package marketingcontentimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	mc "github.com/lextures/lextures/server/internal/service/marketingcontent"
	validator "github.com/lextures/lextures/server/internal/service/marketingcontent/validate"
	"github.com/lextures/lextures/server/internal/service/marketingmedia"
)

type Options struct {
	Root, Only, Slug                               string
	DryRun, FailValidation, AllowMissingGit, Force bool
	ReportPath                                     string
}
type ArticleReport struct {
	Path          string  `json:"path"`
	Slug          string  `json:"slug"`
	Kind          string  `json:"kind"`
	Action        string  `json:"action"`
	Lastmod       string  `json:"lastmod"`
	LastmodSource string  `json:"lastmodSource"`
	Score         float64 `json:"score"`
	Findings      int     `json:"findings"`
	Error         string  `json:"error,omitempty"`
}
type Report struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	DryRun      bool            `json:"dryRun"`
	Articles    []ArticleReport `json:"articles"`
	Summary     map[string]int  `json:"summary"`
}
type Runner struct {
	Pool    *pgxpool.Pool
	Content *mc.Service
	Media   *marketingmedia.Service
	Now     func() time.Time
}

func (r *Runner) Run(ctx context.Context, o Options) (Report, error) {
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	report := Report{GeneratedAt: now, DryRun: o.DryRun, Summary: map[string]int{}}
	authors, err := loadAuthors(o.Root)
	if err != nil {
		return report, err
	}
	knownAuthors := map[string]bool{}
	for _, a := range authors {
		knownAuthors[a.Slug] = true
		if !o.DryRun && o.Only != "media" {
			if _, err = r.Content.SaveAuthor(ctx, a, uuid.Nil); err != nil {
				return report, err
			}
		}
	}
	categories, err := loadCategories(o.Root)
	if err != nil {
		return report, err
	}
	categoryIDs := map[string]uuid.UUID{}
	for _, c := range categories {
		c.ID = uuid.New()
		if !o.DryRun && (o.Only == "" || o.Only == "taxonomy" || o.Only == "docs") {
			saved, e := r.Content.SaveCategory(ctx, c, uuid.Nil)
			if e != nil {
				return report, e
			}
			c.ID = saved.ID
		}
		categoryIDs[c.Slug] = c.ID
	}
	if o.Only == "taxonomy" {
		return report, r.writeReport(o.ReportPath, report)
	}
	articleOnly := o.Only
	if articleOnly == "media" {
		articleOnly = ""
	}
	articles, err := loadArticles(o.Root, articleOnly)
	if err != nil {
		return report, err
	}
	gitRoot, gitErr := gitRoot(o.Root)
	if gitErr != nil && !o.AllowMissingGit {
		return report, fmt.Errorf("git checkout required (use --allow-missing-git to override): %w", gitErr)
	}
	var failures []error
	for _, src := range articles {
		matched, e := filepath.Match(defaultGlob(o.Slug), src.Slug)
		if e != nil {
			return report, e
		}
		if !matched {
			continue
		}
		ar := ArticleReport{Path: src.File, Slug: src.Slug, Kind: src.Kind, Action: "dry-run"}
		in, lastSource, e := r.articleInput(ctx, src, categoryIDs, knownAuthors, gitRoot, o, now)
		if e != nil {
			ar.Action = "failed"
			ar.Error = e.Error()
			report.Articles = append(report.Articles, ar)
			report.Summary["failed"]++
			failures = append(failures, e)
			continue
		}
		ar.Lastmod = in.ContentUpdatedAt.Format(time.RFC3339)
		ar.LastmodSource = lastSource
		lint := validator.Article(validator.Input{Kind: in.Kind, BodyMD: in.BodyMD, Metadata: validator.Metadata{Title: in.Title, Description: in.Description, Updated: in.ContentUpdatedAt.Format("2006-01-02"), Author: in.AuthorSlug, Cluster: in.Cluster, PrimaryQuestion: in.PrimaryQuestion, Keywords: in.Keywords}, KnownPaths: map[string]struct{}{}})
		ar.Score = lint.Score
		ar.Findings = len(lint.Findings)
		if o.FailValidation && hasErrors(lint) {
			ar.Action = "failed"
			ar.Error = "content validation has error findings"
			failures = append(failures, errors.New(ar.Path+": "+ar.Error))
		} else if !o.DryRun && o.Only != "media" {
			res, e := r.Content.ImportArticle(ctx, in, sourceHash(in), o.Force)
			if e != nil {
				ar.Action = "failed"
				ar.Error = e.Error()
				failures = append(failures, fmt.Errorf("%s: %w", src.File, e))
			} else {
				ar.Action = res.Action
				ar.Score = valueOrZero(res.Article.QualityScore)
				var q struct {
					Findings []any `json:"findings"`
				}
				_ = json.Unmarshal(res.Article.QualityReport, &q)
				ar.Findings = len(q.Findings)
			}
		}
		report.Articles = append(report.Articles, ar)
		report.Summary[ar.Action]++
		fmt.Printf("slug=%s action=%s score=%.1f findings=%d lastmod=%s lastmodSource=%s\n", ar.Slug, ar.Action, ar.Score, ar.Findings, ar.Lastmod, ar.LastmodSource)
	}
	if e := r.writeReport(o.ReportPath, report); e != nil {
		return report, e
	}
	if len(failures) > 0 {
		return report, fmt.Errorf("import failed for %d article(s): %v", len(failures), failures[0])
	}
	return report, nil
}

func (r *Runner) articleInput(ctx context.Context, s sourceArticle, categories map[string]uuid.UUID, authors map[string]bool, gitRoot string, o Options, now time.Time) (repo.NewArticle, string, error) {
	m := s.Meta.Values
	author := m["author"]
	if author == "" {
		author = "chase-willden"
	}
	if !authors[author] {
		return repo.NewArticle{}, "", fmt.Errorf("%s: unknown author slug %q", s.File, author)
	}
	var reviewer *string
	if v := m["reviewedBy"]; v != "" {
		if !authors[v] {
			return repo.NewArticle{}, "", fmt.Errorf("%s: unknown reviewer slug %q", s.File, v)
		}
		reviewer = &v
	}
	published, e := date(m["date"])
	if e != nil {
		return repo.NewArticle{}, "", fmt.Errorf("%s date: %w", s.File, e)
	}
	updated, source, e := resolveLastmod(s.File, m["updated"], published, gitRoot, o.AllowMissingGit)
	if e != nil {
		return repo.NewArticle{}, "", e
	}
	var reviewed, due *time.Time
	if m["reviewedAt"] != "" {
		reviewed, e = date(m["reviewedAt"])
		if e != nil {
			return repo.NewArticle{}, "", e
		}
	}
	if m["reviewDue"] != "" {
		due, e = date(m["reviewDue"])
		if e != nil {
			return repo.NewArticle{}, "", e
		}
	}
	body := s.Meta.Body
	if !o.DryRun && r.Media != nil {
		body, e = r.localizeImages(ctx, o.Root, body)
		if e != nil {
			return repo.NewArticle{}, "", fmt.Errorf("%s: %w", s.File, e)
		}
	}
	extraMeta := map[string]any{}
	for _, k := range []string{"contentContract", "supportTicketThemes"} {
		if v := m[k]; v != "" {
			if list, ok := s.Meta.Lists[k]; ok {
				extraMeta[k] = list
			} else {
				extraMeta[k] = v
			}
		}
	}
	sha := ""
	if gitRoot != "" {
		sha, _ = gitOutput(gitRoot, "rev-parse", "HEAD")
	}
	provenance := map[string]any{"sourcePath": s.File, "gitSha": sha, "importedAt": now.Format(time.RFC3339), "lastmodSource": source, "metadata": extraMeta}
	extra, _ := json.Marshal(map[string]any{"import": provenance})
	in := repo.NewArticle{Kind: s.Kind, Slug: s.Slug, Locale: "en", Title: m["title"], Description: m["description"], BodyMD: body, Status: "published", AuthorSlug: author, ReviewerSlug: reviewer, PublishedAt: published, FirstPublishedAt: published, ContentUpdatedAt: updated, ReviewedAt: reviewed, ReviewDueOn: due, PrimaryQuestion: m["primaryQuestion"], Cluster: m["cluster"], Pillar: m["pillar"], BriefRef: m["briefRef"], VerifiedAgainst: m["verifiedAgainst"], Keywords: s.Meta.Lists["keywords"], RelatedTo: s.Meta.Lists["relatedTo"], Roles: s.Meta.Lists["roles"], Segments: s.Meta.Lists["segments"], Citations: s.Meta.Lists["citations"], Extra: extra, ChangeNote: "imported from " + s.File + "@" + sha}
	if s.Kind == "doc" {
		id, ok := categories[s.Category]
		if !ok {
			return in, "", fmt.Errorf("%s: unknown category %q", s.File, s.Category)
		}
		in.CategoryID = &id
	}
	h := sourceHash(in)
	provenance["sourceHash"] = h
	in.Extra, _ = json.Marshal(map[string]any{"import": provenance})
	return in, source, nil
}

var imageRE = regexp.MustCompile(`!\[([^\]]*)\]\((/[^ )]+)\)`)

func (r *Runner) localizeImages(ctx context.Context, root, body string) (string, error) {
	var first error
	out := imageRE.ReplaceAllStringFunc(body, func(match string) string {
		if first != nil {
			return match
		}
		m := imageRE.FindStringSubmatch(match)
		file := filepath.Join(filepath.Dir(root), "public", strings.TrimPrefix(m[2], "/"))
		data, e := os.ReadFile(file)
		if e != nil {
			first = e
			return match
		}
		decorative := strings.TrimSpace(m[1]) == ""
		asset, _, e := r.Media.Create(ctx, marketingmedia.Upload{Data: data, AltText: m[1], Decorative: decorative})
		if e != nil {
			first = e
			return match
		}
		return "![" + m[1] + "](/api/v1/public/content/media/" + asset.ID.String() + "/original." + mediaExt(asset.MIMEType) + ")"
	})
	return out, first
}
func mediaExt(m string) string {
	if m == "image/jpeg" {
		return "jpg"
	}
	return strings.TrimPrefix(m, "image/")
}
func resolveLastmod(file, updated string, published *time.Time, gitRoot string, allow bool) (*time.Time, string, error) {
	if updated != "" {
		v, e := date(updated)
		return v, "frontmatter", e
	}
	if gitRoot != "" {
		rel, e := filepath.Rel(gitRoot, file)
		if e == nil {
			v, e := gitOutput(gitRoot, "log", "-1", "--format=%cI", "--", rel)
			if e == nil && v != "" {
				t, e := time.Parse(time.RFC3339, v)
				if e == nil {
					t = t.UTC()
					return &t, "git", nil
				}
			}
		}
	}
	if published != nil {
		return published, "published", nil
	}
	if allow {
		return nil, "missing", nil
	}
	return nil, "", fmt.Errorf("%s: could not resolve lastmod", file)
}
func gitRoot(root string) (string, error) { return gitOutput(root, "rev-parse", "--show-toplevel") }
func gitOutput(dir string, args ...string) (string, error) {
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, e := c.Output()
	return strings.TrimSpace(string(b)), e
}
func date(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, e := time.Parse("2006-01-02", v)
	if e != nil {
		return nil, e
	}
	t = t.UTC()
	return &t, nil
}
func sourceHash(in repo.NewArticle) string {
	copy := in
	copy.Extra = nil
	copy.ActorID = uuid.Nil
	copy.ChangeNote = ""
	b, _ := json.Marshal(copy)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func defaultGlob(v string) string {
	if v == "" {
		return "*"
	}
	return v
}
func hasErrors(r validator.Report) bool {
	if r.ValidatorError {
		return true
	}
	for _, f := range r.Findings {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}
func valueOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
func (r *Runner) writeReport(path string, v Report) error {
	if path == "" {
		return nil
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
