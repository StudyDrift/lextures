package publicapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<title>Lextures API — Swagger UI</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin="anonymous"></script>
<script>
  window.onload = function () {
    window.ui = SwaggerUIBundle({ url: '/api/v1/openapi.json', dom_id: '#swagger-ui' });
  };
</script>
</body>
</html>`

const redocHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<title>Lextures API — ReDoc</title>
</head>
<body>
<redoc spec-url="/api/v1/openapi.json"></redoc>
<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`

// ServeOpenAPI returns the OpenAPI 3.1 document (cached at build time).
func ServeOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(openAPISpec)
}

// SpecBytes returns the embedded OpenAPI document bytes.
func SpecBytes() []byte {
	return openAPISpec
}

// ServeSwaggerUI returns Swagger UI HTML.
func ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

// ServeReDoc returns ReDoc HTML.
func ServeReDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(redocHTML))
}
