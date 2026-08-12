# MC.5 — Marketing Media Library & Image Pipeline

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans;
> SEO.4 FR-12/FR-13 image policy in [www/scripts/optimize-images.mjs](../../../www/scripts/optimize-images.mjs).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.5 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — article images are PNG files committed to `www/public/`, converted to AVIF/WebP at build time; a non-engineer cannot add one |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Server platform + Web platform |
| **Depends on** | MC.1 |
| **Unblocks** | MC.6, MC.7, MC.10, MC.12 |

---

## 1. Problem Statement

Every screenshot in the help center — `docs-course-interface.png`, `docs-create-course-step1.png`
and the rest — is a file committed to `www/public/`, hard-coded into a dimensions map in
`www/src/lib/markdown.ts`, and converted to AVIF/WebP by a build script. A content expert writing a
new help article in the workspace has no way to add an image, and if we ship the editor without one
they will either publish text-only walkthroughs or ask an engineer for a commit — recreating exactly
the bottleneck this program removes. The image path is also where accessibility is won or lost: alt
text must be structurally required, not culturally encouraged.

## 2. Goals

- Let permitted users upload, name, describe and reuse images from inside the Marketing Content
  workspace, with alt text required before the asset can be referenced.
- Serve those images through the existing storage driver (local disk or S3/CDN) with immutable,
  content-addressed URLs and pre-generated AVIF/WebP/PNG variants at known dimensions.
- Keep the static site's image performance contract intact: `width`/`height` always present,
  `loading="lazy"`, `decoding="async"`, modern formats first — the SEO.4 rules that
  `www/src/lib/markdown.ts` enforces today by hard-coded table.
- Make the build able to localise remote images into `dist/` so `lextures.com` never hot-links the
  app origin for a page image.
- Enforce the same safety controls as other uploads: size caps, MIME sniffing, AV scanning.

## 3. Non-Goals

- No video or audio hosting (out of scope; embedded third-party video stays as today).
- No image editing (crop/resize/annotate) in the UI beyond choosing a rendition.
- No replacement of `www/public/assets/*` marketing art (logos, OG images, illustrations) — those
  stay file-managed; this library is for **content** images.
- No DAM features: no folders-with-permissions, no versioning of a single asset, no usage approval
  workflow.
- No AI alt-text generation in this plan (`internal/service/alttextai` exists and may be offered as a
  *suggestion* later; the requirement to have human-approved alt text does not change).

## 4. Personas & User Stories

- **As a content expert**, I want to drop a screenshot into an article and have it appear correctly
  sized on the published page, so I can document a workflow without engineering help.
- **As a content expert**, I want to reuse an image I uploaded last month, so the same screenshot is
  not uploaded six times.
- **As an accessibility reviewer**, I want alt text to be mandatory at upload time, so no published
  image can be undescribed.
- **As an SRE**, I want image bytes served from object storage/CDN rather than the API process, so a
  popular help article is not a database load problem.
- **As a performance owner**, I want dimensions and modern formats on every content image, so Core
  Web Vitals do not regress when the content team scales up.

## 5. Functional Requirements

- **FR-1.** `POST /api/v1/admin/marketing/media` MUST accept `multipart/form-data` with `file`,
  `altText` (required, 1–300 chars), optional `title` and `credit`, and MUST reject a request whose
  `altText` is empty or whitespace with `422 alt_text_required`.
- **FR-2.** Accepted types MUST be `image/png`, `image/jpeg`, `image/webp`, `image/avif`, `image/gif`
  (still) and `image/svg+xml`; type MUST be determined by content sniffing, not the declared header
  or extension. Max size 10 MB (configurable).
- **FR-3.** SVG uploads MUST be sanitized (strip `<script>`, event handlers, external references) or
  rejected if sanitization fails.
- **FR-4.** Uploaded files MUST be scanned by `internal/clamav` when configured; a positive result
  MUST reject the upload and log a security event.
- **FR-5.** The service MUST derive `width`, `height` and a SHA-256 `checksum`, and MUST deduplicate:
  an upload whose checksum already exists returns the existing asset (with a `deduplicated: true`
  flag) rather than storing a second copy.
- **FR-6.** For raster uploads the service MUST generate renditions — original, `1600w`, `800w`,
  `400w` — each in AVIF and WebP plus the original format, all stored through
  `internal/service/filestorage` under `marketing/media/{id}/{rendition}.{ext}`.
- **FR-7.** Assets MUST be served at `GET /api/v1/public/content/media/{id}/{rendition}.{ext}`
  (anonymous, `Cache-Control: public, max-age=31536000, immutable`) or, when the storage driver has a
  CDN base configured, via a redirect/absolute CDN URL.
- **FR-8.** `GET /api/v1/admin/marketing/media` MUST list assets with filters (`q` over alt/title,
  `unusedOnly`, `mimeType`, cursor pagination) and MUST report `usageCount` — the number of published
  and draft articles referencing the asset.
- **FR-9.** `DELETE /api/v1/admin/marketing/media/{id}` MUST refuse (`409 media_in_use`) while any
  non-deleted article references the asset, and MUST soft-delete otherwise.
- **FR-10.** Markdown MUST reference assets with a stable URL form
  `/api/v1/public/content/media/{id}/original.{ext}`; the editor inserts this automatically. The
  renderer MUST accept the form and MUST NOT require the author to know it.
- **FR-11.** The public content API (MC.3) MUST include, for every asset referenced by an article, a
  `media[]` block with `id`, `alt`, `width`, `height`, `renditions[]` and `checksum`, so the build can
  emit a correct `<picture>` element without extra requests.
- **FR-12.** `www`'s build MUST download referenced assets into `dist/assets/content/{checksum}/…`
  and rewrite URLs to those local paths, so published pages never depend on the app origin at runtime
  ([MC.7](MC.7-www-build-time-content-integration.md) implements the download; this plan specifies
  the contract).
- **FR-13.** The renderer's hard-coded `LOCAL_IMAGE_DIMENSIONS` map MUST be superseded for DB-sourced
  content: dimensions come from the `media[]` block. The map remains for legacy file-based pages
  until MC.15.
- **FR-14.** Hero images MUST be supported: `content_articles.hero_media_id` renders as the OG image
  and the article's lead image, with a documented minimum size (1200×630) enforced as a `warn` at
  save and an `error` at publish when used as an OG image.
- **FR-15.** All media writes MUST be permission-gated: upload/update `…:author`, delete `…:admin`,
  list/read `…:view`.
- **FR-16.** Uploads MUST be recorded in the admin audit log with actor, asset id, checksum and size.

## 6. Non-Functional Requirements

- **Performance** — Upload + rendition generation for a 3 MB PNG completes in < 4 s p95 (rendition
  work runs synchronously for ≤ 5 MB; larger files enqueue a background job and the asset is usable
  at `original` immediately). Public asset reads are served by storage/CDN, not the API, whenever a
  CDN base is configured.
- **Security** — Content-sniffed MIME, size caps, AV scan, SVG sanitization, no user-controlled file
  paths (storage key derives from a server-generated UUID + rendition name), no `Content-Disposition`
  reflection of user filenames without sanitization. Public read path exposes only assets referenced
  by published or draft content; there is no directory listing.
- **Privacy & Compliance** — Screenshots may contain personal data (names in a demo course). The
  upload UI must warn, and MC.11's review checklist includes "no real learner data in screenshots".
  Assets are public once referenced by a published article — treat every upload as public.
- **Accessibility** — Alt text is structurally required (FR-1); the media picker shows alt text in the
  list; decorative images are expressed with an explicit "decorative" checkbox that stores `alt=""`
  plus `role="presentation"`, never an empty required field.
- **Scalability** — Content-addressed dedupe bounds growth; renditions are ~5× the original per
  asset. Expected volume: hundreds of assets, single-digit GB.
- **Reliability** — Storage write happens before the DB row commits; an orphaned object (write
  succeeded, commit failed) is cleaned by a weekly reconciliation job. A missing rendition falls back
  to `original` rather than 404ing.
- **Observability** — `marketing_media_uploads_total{result}`, `…_bytes_total`,
  `…_rendition_duration_seconds`, `…_av_rejections_total`; log fields `media_id`, `checksum`,
  `mime`, `bytes`.
- **Maintainability** — New package `internal/service/marketingmedia`; reuses
  `internal/service/filestorage` and `internal/clamav`; no new storage abstraction.
- **Internationalization** — Alt text is per-asset today. MC.14 open question: localized alt text
  keyed by locale (proposed: `alt_text_i18n JSONB` reserved now, unused).
- **Backward compatibility** — Existing `www/public/*.png` images and their build-time conversion are
  untouched; DB-sourced articles simply use the new path. Both coexist through MC.15.

## 7. Acceptance Criteria

- **AC-1.** *Given* an upload without `altText`, *when* posted, *then* the response is `422` with
  code `alt_text_required` and nothing is stored.
- **AC-2.** *Given* a PNG renamed to `.jpg` with a PNG magic number, *when* uploaded, *then* it is
  accepted and recorded as `image/png` (sniffed, not trusted).
- **AC-3.** *Given* a file whose declared type is `image/png` but whose content is a PDF, *when*
  uploaded, *then* it is rejected with `415`.
- **AC-4.** *Given* an SVG containing `<script>`, *when* uploaded, *then* the stored file contains no
  script element and no `on*` attribute.
- **AC-5.** *Given* the same file uploaded twice, *when* the second upload completes, *then* the
  response references the first asset id with `deduplicated: true` and storage holds one copy.
- **AC-6.** *Given* a 2400×1600 PNG, *when* uploaded, *then* renditions `1600w`, `800w`, `400w` exist
  in AVIF, WebP and PNG, each with recorded dimensions.
- **AC-7.** *Given* an asset referenced by a published article, *when* `DELETE` is called, *then* the
  response is `409 media_in_use` with the referencing paths listed.
- **AC-8.** *Given* an article referencing an asset, *when* the MC.3 detail endpoint is called,
  *then* the payload's `media[]` contains that asset with `alt`, `width`, `height` and all
  renditions.
- **AC-9.** *Given* a public media URL, *when* fetched twice, *then* the second response is served
  from cache/CDN with `immutable` and the API process is not in the path (when CDN configured).
- **AC-10.** *Given* an 11 MB upload with a 10 MB cap, *when* posted, *then* the response is `413`
  and no partial object remains in storage.
- **AC-11.** *Given* ClamAV is configured and returns a positive, *when* an upload is scanned, *then*
  the upload is rejected, the object is deleted, and a security log entry exists.
- **AC-12.** *Given* a decorative image, *when* inserted with the decorative checkbox, *then* the
  rendered HTML has `alt=""` and `role="presentation"` and the a11y lint rule does not fire.

## 8. Data Model

Migration `480_marketing_content_media.sql` (indicative number):

```sql
CREATE TABLE marketing.content_media (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checksum     TEXT NOT NULL UNIQUE,                -- sha256 hex of original bytes
    mime_type    TEXT NOT NULL,
    byte_size    BIGINT NOT NULL,
    width        INTEGER,
    height       INTEGER,
    alt_text     TEXT NOT NULL,                        -- '' only when decorative = TRUE
    decorative   BOOLEAN NOT NULL DEFAULT FALSE,
    title        TEXT NOT NULL DEFAULT '',
    credit       TEXT NOT NULL DEFAULT '',
    storage_key  TEXT NOT NULL,                        -- marketing/media/{id}/original.{ext}
    renditions   JSONB NOT NULL DEFAULT '[]'::jsonb,   -- [{name,ext,mime,width,height,key,bytes}]
    uploaded_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CONSTRAINT alt_or_decorative CHECK (decorative OR length(btrim(alt_text)) > 0)
);
CREATE INDEX idx_mc_media_created ON marketing.content_media (created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE marketing.content_article_media (
    article_id UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    media_id   UUID NOT NULL REFERENCES marketing.content_media (id),
    usage      TEXT NOT NULL CHECK (usage IN ('body', 'hero')),
    PRIMARY KEY (article_id, media_id, usage)
);

ALTER TABLE marketing.content_articles
    ADD CONSTRAINT fk_mc_articles_hero_media
    FOREIGN KEY (hero_media_id) REFERENCES marketing.content_media (id);
ALTER TABLE marketing.content_authors
    ADD CONSTRAINT fk_mc_authors_image_media
    FOREIGN KEY (image_media_id) REFERENCES marketing.content_media (id);
```

`content_article_media` is maintained by the service on every article save by parsing media URLs out
of `body_md` — it is a derived index that makes `usageCount` and FR-11 a join rather than a scan.

**Backfill:** MC.6 uploads the eight existing `docs-*.png` files (and their known dimensions from
`LOCAL_IMAGE_DIMENSIONS`) so imported help articles keep working images.

## 9. API Surface

| Verb | Path | Auth | Notes |
|---|---|---|---|
| POST | `/api/v1/admin/marketing/media` | `…:author` | multipart; `file`, `altText`, `decorative`, `title`, `credit` |
| GET | `/api/v1/admin/marketing/media` | `…:view` | `q`, `mimeType`, `unusedOnly`, cursor |
| GET | `/api/v1/admin/marketing/media/{id}` | `…:view` | detail incl. `usedBy[]` |
| PATCH | `/api/v1/admin/marketing/media/{id}` | `…:author` | alt/title/credit/decorative only |
| DELETE | `/api/v1/admin/marketing/media/{id}` | `…:admin` | 409 when in use |
| GET | `/api/v1/public/content/media/{id}/{rendition}.{ext}` | anonymous | immutable cache; CDN redirect when configured |

```ts
type MediaAsset = {
  id: string; checksum: string; mimeType: string; byteSize: number
  width: number | null; height: number | null
  altText: string; decorative: boolean; title: string; credit: string
  url: string                       // original
  renditions: Array<{ name: '1600w'|'800w'|'400w'|'original'; ext: 'avif'|'webp'|'png'|'jpg'
                      mime: string; width: number; height: number; url: string; bytes: number }>
  usageCount: number; usedBy?: Array<{ articleId: string; path: string; usage: 'body'|'hero' }>
  createdAt: string; uploadedBy: { id: string; name: string } | null
  deduplicated?: boolean
}
```

- **Rate limits:** 30 uploads/min/user; 10 MB body cap; `413` on exceed.
- **OpenAPI:** all routes documented; multipart schema declared.
- **Events:** none (article save recomputes usage).

## 10. UI / UX

Delivered inside the MC.10 editor; specified here:

1. **Insert image** in the editor toolbar opens a media dialog with two tabs: **Upload** and
   **Library**.
2. Upload tab: drag-and-drop or file picker → preview → **required** alt-text field with a
   "decorative image" checkbox that disables and clears it → optional title/credit → Upload.
3. Library tab: searchable grid, alt text shown beneath each thumbnail, filter "not used anywhere",
   selecting inserts the markdown reference at the cursor.
4. Hero image is chosen from the same dialog in the metadata panel, with an inline warning when the
   asset is smaller than 1200×630.
5. States: empty library ("No images yet — upload your first screenshot"), uploading (progress with
   percentage and cancel), rejected (specific reason: too large / wrong type / infected / alt text
   missing), offline (queue disabled with explanation).
6. Mobile/responsive: dialog becomes full-screen below `sm`; grid collapses to two columns.
7. Accessibility: dialog uses `OverlaySurface` with focus trap and restore; grid is a
   `role="listbox"` with roving tabindex; upload progress announced via `aria-live="polite"`; the
   alt-text field is programmatically required (`aria-required`, error linked by `aria-describedby`).
8. Copy/i18n: `marketingContent.media.*` (`upload.title`, `alt.label`, `alt.help`,
   `alt.decorativeLabel`, `error.tooLarge`, `error.unsupportedType`, `error.infected`,
   `library.empty`, `hero.tooSmall`).

## 11. AI / ML Considerations

None in scope. `internal/service/alttextai` exists and could later *suggest* alt text in the upload
dialog; if added, the suggestion must be editable, must be labelled as AI-generated in the UI, and
must not satisfy FR-1 without a human keystroke or explicit accept action — recorded through
`internal/aidisclosure`.

## 12. Integration Points

- **Internal modules:** `internal/service/marketingmedia` (new), `internal/service/filestorage`
  (local/S3/CDN drivers), `internal/clamav`, `internal/repos/marketingcontent`,
  `internal/httpserver/marketing_media_http.go` (new), `internal/service/adminaudit`.
- **`www`:** `scripts/generate-site.mjs` (asset localisation), `src/lib/markdown.ts` (image rendering
  from `media[]`), `scripts/optimize-images.mjs` (unchanged; still handles file-based art).
- **Precedents:** `internal/httpserver/board_posts_http.go` (multipart + altText upload),
  `internal/httpserver/course_file_upload.go` (size caps, sniffing).
- **External services:** S3-compatible storage and CDN when configured; ClamAV daemon.

## 13. Dependencies & Sequencing

- Must ship after: MC.1 (schema), and alongside MC.2 (the media routes share the admin namespace).
- Must ship before: MC.6 (imports the eight existing screenshots), MC.7 (build asset localisation),
  MC.10 (editor insert flow), MC.12 (OG image from hero asset).
- Shared infra: object storage (already present), ClamAV (optional), `sharp`-equivalent image
  processing in Go — use `golang.org/x/image` + `libvips`/`bimg` if available, otherwise shell out to
  the existing image tooling; the decision is recorded in §18.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Go-side AVIF encoding is awkward | **H** | M | Renditions are generated best-effort: WebP + original always, AVIF when the encoder is available; `www`'s build (which already runs `sharp`) fills any missing AVIF during asset localisation, so the published page always has one |
| Screenshots leak real learner data | M | H | Upload warning copy, review checklist item (MC.11), and a policy that demo data comes from the intro course fixtures |
| Orphaned storage objects | M | L | Weekly reconciliation job comparing storage keys to `content_media`; report-only for the first month |
| SVG sanitizer bypass | L | H | Prefer rejecting SVG entirely if the sanitizer library is not vetted; decision in §18. If accepted, serve SVG with `Content-Security-Policy: sandbox` and `Content-Disposition: inline` only |
| Image bytes served by the API become a load problem | M | M | CDN base configured in production; `immutable` caching; the static site localises assets so public traffic does not hit the API at all |
| Alt text becomes a checkbox ritual ("image") | M | M | Lint rule flags alt text shorter than 8 characters or equal to the filename as `warn`; review checklist covers it |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content`. No separate flag.
- **Sequencing:** schema → upload/list/read service → public read route → editor integration (MC.10)
  → importer backfill (MC.6) → build localisation (MC.7).
- **Dogfood:** the content team re-uploads the eight existing help screenshots and confirms the
  rendered help pages are pixel-equivalent to today's build.
- **GA criteria:** all ACs; AV scanning verified; CDN path verified in staging; no image on a
  generated page missing `width`/`height`.
- **Rollback:** flag off. Uploaded assets remain in storage but are unreferenced; imported articles
  fall back to the file-based images that still exist in `www/public/` until MC.15 removes them.

## 16. Test Plan

- **Unit** — MIME sniffing matrix; size cap; checksum dedupe; rendition naming and dimension math;
  SVG sanitizer; alt/decorative constraint; usage extraction from markdown bodies.
- **Integration** — upload → storage object exists → DB row → public URL serves bytes with correct
  headers; delete-in-use conflict; reconciliation job finds a deliberately orphaned object; ClamAV
  positive path with a stubbed scanner.
- **End-to-end** — Playwright: upload an image in the editor, insert it, preview shows it, publish,
  and the generated page (MC.7 build in CI) contains a `<picture>` with AVIF/WebP sources and
  explicit dimensions.
- **Security** — polyglot file uploads (GIFAR-style), path traversal in filename, `Content-Type`
  spoofing, oversized multipart, zip-bomb-style decompression guard, SVG XSS corpus, unauthenticated
  access to admin media routes.
- **Accessibility** — axe on the media dialog; keyboard-only upload and selection; screen-reader
  script for the alt-text requirement and error announcement.
- **Performance / load** — 20 concurrent 5 MB uploads; assert p95 < 6 s and no memory blow-up;
  measure rendition CPU cost.
- **Manual exploratory** — upload each supported type, a corrupt file, a huge file, and a duplicate;
  verify library filters and usage counts.

## 17. Documentation & Training

- Help article (public, written in the new system as its own dogfood): "Adding images to help
  articles", covering alt text, sizing guidance and what not to screenshot.
- `www/docs/performance-budget.md` — note that DB-sourced images carry dimensions from the API.
- Internal runbook: storage layout, reconciliation job, how to purge an asset that must be removed
  for legal reasons (including CDN invalidation).

## 18. Open Questions

1. Do we accept SVG at all for content images? (Proposed: **no** for v1 — reject with a message
   pointing at PNG export; revisit if the content team needs diagrams.)
2. Which Go image pipeline for renditions — pure-Go (`x/image`, no AVIF) vs `libvips` binding (adds a
   system dependency to the server image)? (Proposed: pure-Go for WebP/PNG/JPEG, AVIF filled in by
   the `www` build, per the risk table.)
3. Should hero images be mandatory for blog posts (OG image quality) or optional with a branded
   fallback? (Proposed: optional with a generated branded fallback in MC.12.)
4. Do we need per-asset expiry/retention for time-limited campaign art? (Proposed: no; out of scope.)

## 19. References

- Files this work touches: `server/internal/service/marketingmedia/*`,
  `server/internal/httpserver/marketing_media_http.go`, `server/migrations/480_*`,
  `www/src/lib/markdown.ts`, `www/scripts/generate-site.mjs`.
- Precedents: `server/internal/httpserver/board_posts_http.go`,
  `server/internal/service/filestorage/{s3,local}.go`, `www/scripts/optimize-images.mjs`.
- Standards: WCAG 2.1 AA (1.1.1 Non-text Content), OWASP File Upload Cheat Sheet.
- Related plans: [MC.6](MC.6-markdown-to-database-migration.md),
  [MC.7](MC.7-www-build-time-content-integration.md), [MC.10](MC.10-article-editor.md),
  [MC.12](MC.12-seo-parity-from-database.md).
