# Answer-first content contract

The normative Markdown/directive and validation specification is
[`docs/guides/marketing-content-dialect.md`](../../docs/guides/marketing-content-dialect.md).
The same deterministic contract now runs on API saves and publish transitions as well as builds.

Editorial pages use safe Markdown directives. Content is compiled to static HTML; imports, JSX, raw HTML, and unknown directives fail lint for new `.mdx` content.

Every new page must provide the validated front matter below, open with three to five complete conclusions, answer its `primaryQuestion` in 40–60 words, cite numeric claims inline, include descriptive internal links, and close with three to six FAQ entries. Question headings, 120–180-word self-contained passages, lists/tables/steps, and primary citations determine the extractability score. New pages must score 8.0; 6.0–7.9 warns and scores below 6.0 fail. Older Markdown remains listed as grandfathered in the quality report until refreshed.

```markdown
---
title: "How do you design an effective rubric?"
description: "A concise 120–160 character summary that gives readers a distinct reason to open and use this practical guide."
published: 2026-08-11
updated: 2026-08-11
author: chase-willden
cluster: assessment
primaryQuestion: "How do you design an effective rubric?"
keywords: [rubric design, assessment]
relatedTo: [/guides, /platform/assessment, /blog]
---

:::key-takeaways
- Effective rubrics describe observable thinking instead of cosmetic output.
- Students use criteria better when they see them before starting work.
- Primary evidence should support every numeric or research claim.
:::

:::answer
Write a complete 40–60-word answer here. Name the subject directly, include the essential qualification, and avoid pronouns that depend on the title or surrounding paragraph for meaning. A reader should understand the answer after seeing only this block and the primary question.
:::

## How do effective rubric criteria work?

Write a self-contained 120–180-word passage and add descriptive links to [assessment resources](/guides), the [assessment platform](/platform/assessment), and [related research](/blog).

:::faq
### What belongs in a rubric criterion?

Write a 40–80-word answer.

### When should students receive the rubric?

Write a 40–80-word answer.

### How often should a rubric be reviewed?

Write a 40–80-word answer.
:::
```

Run `npm run content:lint`, `npm run content:score -- src/blog/example.mdx`, or `npm run content:report` from `www/`.
