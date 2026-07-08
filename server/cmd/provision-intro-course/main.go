// Command provision-intro-course idempotently provisions the canonical intro course (IC01).
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
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	runMig := flag.Bool("migrate", false, "apply embedded SQL migrations before provisioning (uses DATABASE_URL)")
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
	dbPlatform, err := platformconfig.Get(ctx, pool)
	if err != nil {
		log.Fatalf("platform settings: %v", err)
	}
	merged := platformconfig.Merge(cfg, dbPlatform)

	svc := introcourseservice.New(pool)
	course, err := svc.EnsureProvisioned(ctx, merged)
	if err != nil {
		log.Fatalf("provision: %v", err)
	}
	if course.ID.String() == "00000000-0000-0000-0000-000000000000" {
		log.Fatal("intro course is disabled and has not been provisioned")
	}
	status := "reconciled"
	if course.Created {
		status = "created"
	}
	fmt.Printf("%s intro course %s (course_id=%s short_code=%s)\n", status, course.CourseCode, course.ID, course.ShortCode)
}