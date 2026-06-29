package migrate

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// LintResult holds migration lint findings for CI and local checks.
type LintResult struct {
	MissingDown []string
	Warnings    []string
	Errors      []string
}

// LintFS validates migration conventions under dir (e.g. "migrations").
func LintFS(fsys fs.FS, dir string) (LintResult, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return LintResult{}, fmt.Errorf("migrate lint: readdir: %w", err)
	}

	var list []migrationFile
	for _, e := range entries {
		if e.IsDir() || !isUpMigration(e.Name()) {
			continue
		}
		p := dir + "/" + e.Name()
		mf, perr := parseMigrationName(p)
		if perr != nil {
			return LintResult{}, perr
		}
		list = append(list, mf)
	}
	sortMigrations(list)

	var res LintResult
	secretPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)postgres(ql)?://[^\s'"]+`),
		regexp.MustCompile(`(?i)(api[_-]?key|secret|password)\s*=\s*['"][^'"]+['"]`),
	}
	destructivePatterns := []struct {
		re      *regexp.Regexp
		message string
	}{
		{regexp.MustCompile(`(?i)\bDROP\s+COLUMN\b`), "DROP COLUMN — use expand/contract; remove old column only after all app instances use the new schema"},
		{regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`), "DROP TABLE — prefer soft-delete or rename; contract only after traffic is on the new schema"},
		{regexp.MustCompile(`(?i)\bRENAME\s+COLUMN\b`), "RENAME COLUMN — add a new column and dual-write instead of renaming in place"},
		{regexp.MustCompile(`(?i)\bALTER\s+COLUMN\b[^;]*\bTYPE\b`), "ALTER COLUMN TYPE — use expand/contract with a new column and batched backfill"},
	}

	for _, mf := range list {
		upPath := dir + "/" + mf.Name
		body, rerr := fs.ReadFile(fsys, upPath)
		if rerr != nil {
			return LintResult{}, fmt.Errorf("migrate lint: read %q: %w", upPath, rerr)
		}
		text := string(body)

		downPath := downMigrationPath(upPath)
		if _, derr := fs.Stat(fsys, downPath); derr != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("missing companion down migration: %s", filepath.Base(downPath)))
		}

		for _, pat := range secretPatterns {
			if pat.MatchString(text) {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: possible inline secret — migrations must not contain credentials", mf.Name))
				break
			}
		}

		for _, dp := range destructivePatterns {
			if dp.re.MatchString(text) {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %s", mf.Name, dp.message))
			}
		}

		if isUnbatchedFullTableUpdate(text) {
			res.Warnings = append(res.Warnings, fmt.Sprintf(
				"%s: full-table UPDATE without batching — use batched UPDATE (1,000 rows/transaction) to avoid long locks",
				mf.Name,
			))
		}
	}

	return res, nil
}

func isUnbatchedFullTableUpdate(sql string) bool {
	upper := strings.ToUpper(sql)
	if !strings.Contains(upper, "UPDATE ") {
		return false
	}
	if strings.Contains(upper, "LIMIT ") || strings.Contains(upper, "BATCH") {
		return false
	}
	// Heuristic: UPDATE ... SET without WHERE or with WHERE true is a full-table scan.
	if strings.Contains(upper, "WHERE") {
		return false
	}
	return true
}

// FormatLintReport renders lint output for humans and CI annotation consumers.
func FormatLintReport(res LintResult) string {
	var b strings.Builder
	for _, e := range res.Errors {
		fmt.Fprintf(&b, "ERROR: %s\n", e)
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(&b, "WARNING: %s\n", w)
	}
	return b.String()
}
