// Command provision-marketplace-courses idempotently provisions official marketplace courses (MC0).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	mcservice "github.com/lextures/lextures/server/internal/service/marketplacecourses"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	runMig := flag.Bool("migrate", false, "apply embedded SQL migrations before provisioning (uses DATABASE_URL)")
	only := flag.String("only", "", "provision only this course catalog slug (or content directory name)")
	deploy := flag.Bool("deploy", false, "provision official catalog courses only (skips harness-smoke)")
	flag.Parse()

	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	if *runMig {
		if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	cfg := config.Load()
	svc := mcservice.New(pool)
	var courses []mcservice.Course
	if slug := strings.TrimSpace(*only); slug != "" {
		c, err := svc.EnsureProvisioned(ctx, cfg, slug)
		if err != nil {
			log.Fatalf("provision %s: %v", slug, err)
		}
		courses = []mcservice.Course{c}
	} else if *deploy {
		courses, err = svc.EnsureDeployProvisioned(ctx, cfg)
		if err != nil {
			log.Fatalf("provision: %v", err)
		}
	} else {
		courses, err = svc.EnsureAllProvisioned(ctx, cfg)
		if err != nil {
			log.Fatalf("provision: %v", err)
		}
	}

	for _, c := range courses {
		status := "reconciled"
		if c.Created {
			status = "created"
		} else if c.Report.Skipped {
			status = "noop"
		}
		fmt.Printf("%s marketplace course %s (course_id=%s catalog_slug=%s) %s\n",
			status, c.CourseCode, c.ID, c.CatalogSlug,
			mcservice.FormatProvisionSummary(c.CatalogSlug, boolToInt(c.Created),
				c.Report.Modules, c.Report.Pages, c.Report.Assignments, c.Report.Quizzes, boolToInt(c.Report.Skipped)))
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
