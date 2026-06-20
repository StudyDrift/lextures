package publicapi

// GraphQLSchemaStub is a read-only GraphQL schema placeholder (plan 16.1 FR-8).
const GraphQLSchemaStub = `# Lextures public GraphQL API (query-only stub)
schema {
  query: Query
}

type Query {
  """API health metadata for integrators."""
  apiInfo: APIInfo!
}

type APIInfo {
  version: String!
  restBasePath: String!
  note: String!
}
`
