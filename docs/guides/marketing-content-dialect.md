# Marketing content dialect

Marketing articles use CommonMark with GFM tables, strikethrough, autolinks, lists, blockquotes,
code blocks, images, and headings. Raw HTML, JSX, imports, scripts, and executable URLs are never
accepted. The public TypeScript renderer and the API's sanitized Go renderer are pinned to the
shared corpus in `tests/fixtures/content-render`.

## Content contract

Every article supplies `title`, `description`, `updated`, `author`, `cluster`, `primaryQuestion`,
and at least one keyword. A publishable article opens with three to five key takeaways, includes a
40–60 word direct answer, uses question-oriented level-two headings and self-contained passages,
cites numeric claims, links to relevant internal pages, and closes with three to six FAQ questions.
Scores below 6.0 fail, scores from 6.0 through 7.9 warn, and scores of 8.0 or above pass. Safety,
unknown directive, missing metadata, unresolved internal-link, and missing image-alt findings are
always errors.

Author profile links are stored as consented JSON in the author registry:

```json
{ "sameAs": ["https://github.com/example"], "website": "https://example.com" }
```

Active authors use these links on schema.org `Person` nodes. Retired authors keep plain-text
bylines, but do not emit a `Person` node or author page.

## Directives

Directives start and end on their own lines. They cannot be nested.

| Directive | Purpose | Optional argument |
|---|---|---|
| `key-takeaways` | Three to five conclusions | none |
| `answer` | Direct answer to the primary question | none |
| `definition` | Defined term | `term="…"` |
| `comparison-table` | Responsive GFM table | `summary="…"` |
| `steps` | Ordered procedure / HowTo content | none |
| `faq` | Three to six `### Question?` entries | none |
| `callout` | Editorial note | `note`, `warning`, or `tip` |
| `stat` | Quoted statistic | source/caption text |
| `sources` | Source-definition section | none |

Unknown or malformed directives remain escaped text in rendered output and block publication.

## Citations, links, headings, and images

Use `[^1]` for a citation and define it with `[^1]: https://…`. Internal links must resolve to a
published article or a route uploaded from the generated SEO manifest. Link labels must describe
their destination. Level-two through level-four headings receive a lowercase, punctuation-stripped
ID and `tabindex="-1"`. Images use `![useful alternative text](url)`; empty alt text is an error.
External links receive `target="_blank" rel="noopener noreferrer"`.

## Rule reference

Stable rule families are `fm.*`, `struct.*`, `passage.*`, `cite.*`, `link.*`, `directive.*`,
`safety.*`, `a11y.image-alt`, and `extractability.score`. Findings include line and column when the
source location is known, or `path` for metadata fields. The lint endpoint does not persist content;
article create and update operations persist its score and full report on the article revision.
