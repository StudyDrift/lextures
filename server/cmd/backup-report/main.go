// Command backup-report records a backup heartbeat in compliance.backup_tier_status (plan 10.15).
// Intended for cron after WAL-G base backup or object-storage snapshot jobs.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lextures/lextures/server/internal/db"
	backupservice "github.com/lextures/lextures/server/internal/service/backup"
	repobackup "github.com/lextures/lextures/server/internal/repos/backup"
)

func main() {
	tierFlag := flag.String("tier", "postgres", "postgres or object_storage")
	success := flag.Bool("success", true, "whether the backup job succeeded")
	duration := flag.Int("duration-seconds", 0, "job duration in seconds")
	walLag := flag.Int("wal-lag-seconds", -1, "WAL lag in seconds (postgres tier only; omit if unknown)")
	next := flag.String("next-scheduled", "", "RFC3339 timestamp for next scheduled backup")
	errMsg := flag.String("error", "", "last error message when success=false")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	var tier repobackup.Tier
	switch *tierFlag {
	case "postgres":
		tier = repobackup.TierPostgres
	case "object_storage":
		tier = repobackup.TierObjectStorage
	default:
		fmt.Fprintln(os.Stderr, "tier must be postgres or object_storage")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	var lastSuccess *time.Time
	var lastErr *string
	if *success {
		now := time.Now().UTC()
		lastSuccess = &now
	} else if *errMsg != "" {
		lastErr = errMsg
	}

	var durationPtr *int
	if *duration > 0 {
		durationPtr = duration
	}
	var walPtr *int
	if *walLag >= 0 {
		walPtr = walLag
	}
	var nextPtr *time.Time
	if *next != "" {
		t, err := time.Parse(time.RFC3339, *next)
		if err != nil {
			fmt.Fprintf(os.Stderr, "next-scheduled: %v\n", err)
			os.Exit(1)
		}
		nextPtr = &t
	}

	if err := backupservice.ReportTierHeartbeat(ctx, pool, tier, lastSuccess, durationPtr, walPtr, nextPtr, lastErr); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("recorded backup heartbeat for tier=%s success=%v\n", tier, *success)
}
