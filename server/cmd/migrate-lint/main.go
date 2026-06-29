// Command migrate-lint validates SQL migration conventions for CI and local development.
package main

import (
	"fmt"
	"os"

	"github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/migrate"
)

func main() {
	res, err := migrate.LintFS(serverdata.Migrations, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate lint: %v\n", err)
		os.Exit(2)
	}

	report := migrate.FormatLintReport(res)
	if report != "" {
		fmt.Print(report)
	}

	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "::warning file=server/migrations::%s\n", w)
	}

	if len(res.Errors) > 0 {
		os.Exit(1)
	}
}
