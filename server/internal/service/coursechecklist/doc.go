// Package coursechecklist is the Course Checklist rule registry and evaluation
// engine (CC.1).
//
// Checklist items are declarative ItemDescriptor values registered in code — no
// migration, table, or route is required to add a rule. Evaluate runs against an
// in-memory CourseSnapshot loaded once per evaluation; individual evaluators are
// pure functions of that snapshot (plus optional LazyLoader results) and MUST NOT
// write to the database.
//
// Stable ItemID values are a persisted contract for dismissals (CC.2), telemetry
// (CC.10), and mobile clients (CC.9). Retire an ID via RETIRED_ITEM_IDS; rename
// via ITEM_ID_ALIASES.
package coursechecklist
