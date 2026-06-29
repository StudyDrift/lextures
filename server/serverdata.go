// Package serverdata holds server-wide embedded assets. It lives at the module root
// so go:embed can see the top-level ./migrations directory.
package serverdata

import "embed"

// Migrations is the SQL migration tree (server/migrations).
//
//go:embed all:migrations
var Migrations embed.FS

// ContentFilterAllowlistJSON is the published URL allowlist for K-12 web-content filters (plan 13.14).
//
//go:embed config/content-filter-allowlist.json
var ContentFilterAllowlistJSON []byte
