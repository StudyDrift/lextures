# Utility page policy

Programmatic pages are indexable only when they offer an action, contain at least 150 words of page-specific prose with less than 60% five-gram similarity to siblings, cite a primary source, and have at least three inbound and three outbound internal links. Reviewed glossary content also needs a named reviewer.

`scripts/generate/utility-check.mjs` returns `noindex,follow` for failures. Generators must use that result for manifest eligibility and omit failures from sitemaps and curated `llms.txt`. A family launches only when at least 80% of its pages pass. Overrides must record a reason, owner, and expiry and remain `noindex` until review.
