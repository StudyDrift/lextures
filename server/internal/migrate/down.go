package migrate

import (
	"strings"
)

// downMigrationPath returns the companion rollback file for an up migration path.
// Example: migrations/001_users.sql → migrations/001_users.down.sql
func downMigrationPath(upPath string) string {
	if strings.HasSuffix(upPath, ".sql") {
		return upPath[:len(upPath)-4] + ".down.sql"
	}
	return upPath + ".down.sql"
}

// isDownMigration reports whether filename is a rollback companion (not an up migration).
func isDownMigration(name string) bool {
	base := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		base = name[i+1:]
	}
	return strings.HasSuffix(base, ".down.sql")
}

// isUpMigration reports whether filename is a forward migration SQL file.
func isUpMigration(name string) bool {
	base := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		base = name[i+1:]
	}
	return strings.HasSuffix(base, ".sql") && !strings.HasSuffix(base, ".down.sql")
}

// rollbackSupported reports whether down SQL contains executable rollback statements.
// Stubs that are comment-only or contain "rollback not supported" return false.
func rollbackSupported(body []byte) bool {
	stripped := stripSQLComments(string(body))
	stripped = strings.TrimSpace(stripped)
	if stripped == "" {
		return false
	}
	lower := strings.ToLower(stripped)
	if strings.Contains(lower, "rollback not supported") {
		return false
	}
	return true
}

// stripSQLComments removes -- line comments and /* */ block comments.
func stripSQLComments(sql string) string {
	var b strings.Builder
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	out := b.String()
	for {
		start := strings.Index(out, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(out[start:], "*/")
		if end < 0 {
			out = out[:start]
			break
		}
		out = out[:start] + out[start+end+2:]
	}
	return out
}
