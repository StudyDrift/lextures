// Command marketing-content-import idempotently shadows file-backed marketing content into PostgreSQL.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	mc "github.com/lextures/lextures/server/internal/service/marketingcontent"
	importer "github.com/lextures/lextures/server/internal/service/marketingcontentimport"
	"github.com/lextures/lextures/server/internal/service/marketingmedia"
)

func main() {
	var o importer.Options
	flag.StringVar(&o.Root, "content-root", "../www/src", "content source root")
	flag.StringVar(&o.Only, "only", "", "blog, docs, media, or taxonomy")
	flag.StringVar(&o.Slug, "slug", "*", "article slug glob")
	flag.BoolVar(&o.DryRun, "dry-run", false, "validate and report without database writes")
	flag.BoolVar(&o.FailValidation, "fail-on-validation-error", false, "fail articles with error findings")
	flag.BoolVar(&o.AllowMissingGit, "allow-missing-git", false, "allow published date fallback outside git")
	flag.BoolVar(&o.Force, "force", false, "overwrite articles with post-import revisions")
	confirmProduction := flag.Bool("confirm-production", false, "confirm an import against a non-local database")
	report := flag.String("report", "../docs/plan/marketing-content/import-report.json", "JSON audit report path")
	databaseURL := flag.String("database-url", "", "PostgreSQL URL (defaults to DATABASE_URL)")
	storageRoot := flag.String("media-root", "data/course-files", "local media storage root")
	flag.Parse()
	o.ReportPath = *report
	if o.Only != "" && o.Only != "blog" && o.Only != "docs" && o.Only != "media" && o.Only != "taxonomy" {
		log.Fatalf("invalid --only %q", o.Only)
	}
	dsn := strings.TrimSpace(*databaseURL)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" && !o.DryRun {
		log.Fatal("DATABASE_URL or --database-url is required")
	}
	if dsn != "" && !*confirmProduction && !isSafeHost(dsn) {
		log.Fatal("refusing non-local/non-staging database; pass --confirm-production")
	}
	abs, e := filepath.Abs(o.Root)
	if e != nil {
		log.Fatal(e)
	}
	o.Root = abs
	ctx := context.Background()
	runner := &importer.Runner{}
	if dsn != "" {
		pool, e := db.NewPool(ctx, dsn)
		if e != nil {
			log.Fatalf("db: %v", e)
		}
		defer pool.Close()
		storage := &filestorage.LocalDriver{Root: *storageRoot}
		runner.Pool = pool
		runner.Content = &mc.Service{Pool: pool}
		runner.Media = &marketingmedia.Service{Pool: pool, Storage: storage}
	}
	if o.DryRun {
		runner.Content = &mc.Service{}
	}
	r, e := runner.Run(ctx, o)
	if e != nil {
		log.Fatal(e)
	}
	fmt.Printf("summary created=%d updated=%d unchanged=%d dry-run=%d failed=%d\n", r.Summary["created"], r.Summary["updated"], r.Summary["unchanged"], r.Summary["dry-run"], r.Summary["failed"])
}

func isSafeHost(dsn string) bool {
	u, e := url.Parse(dsn)
	if e != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || strings.Contains(h, "staging")
}
