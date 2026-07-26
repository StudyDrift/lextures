// Package openapi serves the API description and Swagger UI.
//
// The OpenAPI 3.0.3 document lives in openapi.json (embedded at build time).
// Edit that file for API documentation changes — do not put the spec in a Go string.
// See docs/ARCHITECTURE_CONVENTIONS.md and TD.3.
package openapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var specBytes []byte

const docHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>StudyDrift API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin="anonymous"></script>
<script>
  window.onload = function () {
    window.ui = SwaggerUIBundle({ url: '/api/openapi.json', dom_id: '#swagger-ui' });
  };
</script>
</body>
</html>
`

// ServeOpenAPI returns the OpenAPI JSON document.
func ServeOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(specBytes)
}

// SpecBytes returns the embedded OpenAPI document (for tests and tooling).
func SpecBytes() []byte {
	return specBytes
}

// ServeDocs returns HTML that loads Swagger UI against /api/openapi.json.
func ServeDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docHTML))
}
