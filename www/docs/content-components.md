# Content components

The live, noindex examples are at `/internal/content-kit`. Author directives use paired `:::` fences:

- `key-takeaways`: three to five Markdown bullets.
- `answer`: a standalone 40–60-word response.
- `definition term="…"`: a reusable term definition harvested into `.definitions.json`.
- `comparison-table summary="…"`: a summary followed by a Markdown table with a caption supplied in content.
- `steps`: a Markdown ordered list for procedural content.
- `faq`: three to six `### Question?` entries with 40–80-word answers; output is expanded and feeds matching FAQ schema.
- `callout note|warning|tip`: a text-labelled aside.
- `stat source="…"`: a highlighted statistic and source label.
- `sources`: the numbered primary-source list with access dates.

All components are server rendered. Tables scroll within their container, steps remain ordered lists, and FAQ entries use native keyboard-operable disclosures that are open by default.
