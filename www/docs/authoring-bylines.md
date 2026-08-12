# Author bylines & registry (SEO.3)

Editorial pages use **named human authors**, not “Lextures Team”. Each author is a registry entry with consent, and blog/docs front-matter references them by **slug**.

## Registry

Source of truth: `www/src/lib/authors.ts` (plus optional markdown under `www/src/content/authors/` for bio drafts).

| Field | Purpose |
|---|---|
| `slug` | Front-matter value (`author: chase-willden`) |
| `name`, `jobTitle`, `bio` | Visible byline + `Person` schema |
| `knowsAbout`, `sameAs`, `credentials` | Schema only if true and consented |
| `consentRecordedAt` | ISO date consent was recorded (S04) |
| `status` | `active` \| `retired` |

## Onboard an author

1. Obtain **written consent** to publish name, role, bio, photo (if any), and profile links. Record the date.
2. Add an entry to `AUTHORS` in `authors.ts` with `status: 'active'`.
3. Optionally add `www/src/content/authors/<slug>.md` for editorial notes.
4. Create `/authors/<slug>` automatically via `enumerate()` — no separate route edit.
5. Use the slug in blog/docs front-matter:

```yaml
---
title: "…"
date: "2026-09-14"
description: "…"
author: chase-willden
reviewedBy: chase-willden   # optional
updated: 2026-09-20         # optional; feeds dateModified + sitemap lastmod
citations: ["https://example.org/source"]  # blog: primary sources for Article.citation
---
```

6. Build must succeed. An unknown slug **fails the build** (modules call `requireAuthor`).

## Retire an author (erasure / leave)

1. Set `status: 'retired'` on the registry entry.
2. Do **not** delete the slug — historical bylines stay as **plain text** (no link).
3. `/authors/<slug>` stops enumerating (404 / not found UI).
4. `Person` nodes are omitted from all graphs (`buildPerson` returns null).
5. No build failure for posts that still reference the slug.

## Visible UI

- `Byline` component (`www/src/components/byline.tsx`) on blog and docs posts.
- Blog index shows the author display name.
- Footer links to `/about` and `/authors` (entity home within one click).

## Do not

- Invent credentials, ORCID, or LinkedIn URLs without consent.
- Publish home location or personal email.
- Use free-text `author: "Marketing Team"` — always a registry slug.
