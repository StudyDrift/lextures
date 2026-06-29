// Command migrate provides operational helpers for database migrations (plan 17.10).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "rollback":
		rollbackCmd(os.Args[2:])
	case "lint":
		lintCmd()
	default:
		usage()
		os.Exit(2)
	}
}

func rollbackCmd(args []string) {
	fs := flag.NewFlagSet("rollback", flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("DATABASE_URL"), "Postgres connection string")
	_ = fs.Parse(args)

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "migrate rollback: set -dsn or DATABASE_URL")
		os.Exit(2)
	}
	err := migrate.RollbackLatest(context.Background(), serverdata.Migrations, *dsn)
	if err != nil {
		if errors.Is(err, migrate.ErrRollbackNotSupported) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("rollback complete")
}

func lintCmd() {
	res, err := migrate.LintFS(serverdata.Migrations, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate lint: %v\n", err)
		os.Exit(2)
	}
	fmt.Print(migrate.FormatLintReport(res))
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "::warning file=server/migrations::%s\n", w)
	}
	if len(res.Errors) > 0 {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  migrate rollback [-dsn URL]   Roll back the most recent migration using its down.sql
  migrate lint                  Validate migration conventions (down.sql, secrets, destructive patterns)

Environment:
  DATABASE_URL   Default connection string for rollback
`)
}
