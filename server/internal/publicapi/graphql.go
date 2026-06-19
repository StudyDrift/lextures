package publicapi

import (
	"net/http"
)

const graphqlSchema = `# Lextures public GraphQL API (query-only stub — plan 16.1)
# Mutations are not supported in v1. Use the REST API for writes.

type Query {
  """API health metadata"""
  apiVersion: String!
  """Whether GraphQL query execution is available (always false in v1 stub)"""
  graphqlEnabled: Boolean!
}
`

// ServeGraphQL returns the read-only GraphQL schema stub.
func ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		WriteProblem(w, Problem{
			Type:   problemBaseType + "method-not-allowed",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: "Only GET is supported for the GraphQL schema stub.",
		})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(graphqlSchema))
}
