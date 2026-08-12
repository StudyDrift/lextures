# Publishing original research

The public policy is at `/resources/research/methodology`. It explains participation, de-identification,
minimum cell sizes, complementary suppression, preregistration, publication, and corrections in
administrator-readable language.

Every report must have a manifest route and static HTML, full report-specific methods, limitations,
CSV and JSON data, a dictionary, CC BY 4.0 notice, accessible chart tables, `Report` + `Dataset`
JSON-LD, citation text, version history, errata, and a press kit. Use the reviewed React components in
`src/components/research/` and the schema builder in `src/lib/schema/research.ts`.

Never publish placeholder findings or run an extract while a participation setting or DPIA is
unresolved. See `/research/README.md` at the repository root for the internal gates.

