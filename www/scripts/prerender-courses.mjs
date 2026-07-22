#!/usr/bin/env node
/**
 * Build-time prerender for www marketplace SEO (plan MKT10) and the
 * /self-learner → /homeschool redirect stub (plan HS.2).
 *
 * After `vite build`, fetches the public marketplace catalog and writes:
 *   - dist/courses/index.html
 *   - dist/courses/<slug>/index.html  (with per-course meta + JSON-LD)
 *   - dist/self-learner/index.html    (static meta-refresh + canonical redirect)
 *   - dist/sitemap.xml
 *   - dist/robots.txt
 *
 * Env:
 *   API_BASE / VITE_API_BASE_URL — API origin (default https://self.lextures.com)
 *   SITE_ORIGIN — public site origin (default https://lextures.com)
 *   SKIP_COURSE_PRERENDER=1 — skip course pages; still write /courses + sitemap shell + redirect stub
 */

import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const DIST = path.join(ROOT, 'dist')

const API_BASE = (
  process.env.API_BASE ||
  process.env.VITE_API_BASE_URL ||
  'https://self.lextures.com'
).replace(/\/$/, '')
const SITE_ORIGIN = (process.env.SITE_ORIGIN || 'https://lextures.com').replace(/\/$/, '')
const SKIP = process.env.SKIP_COURSE_PRERENDER === '1'
const DEFAULT_OG_IMAGE = `${SITE_ORIGIN}/assets/lextures-mark.svg`
const CONCURRENCY = 6

/** Turn API-relative asset paths into absolute URLs on the homeschool app origin. */
function resolveApiAssetUrl(url, apiBase = API_BASE) {
  if (!url) return null
  const trimmed = String(url).trim()
  if (!trimmed) return null
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) return trimmed
  if (trimmed.startsWith('/')) return `${apiBase}${trimmed}`
  return trimmed
}

const STATIC_ROUTES = [
  { loc: '/', priority: '1.0' },
  { loc: '/pricing', priority: '0.8' },
  { loc: '/pricing/calculator', priority: '0.7' },
  { loc: '/docs', priority: '0.7' },
  { loc: '/blog', priority: '0.7' },
  { loc: '/courses', priority: '0.9' },
  { loc: '/get-started', priority: '0.8' },
  { loc: '/higher-ed', priority: '0.6' },
  { loc: '/k-12', priority: '0.6' },
  { loc: '/parents', priority: '0.6' },
  { loc: '/homeschool', priority: '0.6' },
  { loc: '/privacy', priority: '0.3' },
  { loc: '/terms', priority: '0.3' },
  { loc: '/security', priority: '0.3' },
  { loc: '/accessibility', priority: '0.3' },
]

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function truncateMeta(text, maxLen = 160) {
  const cleaned = String(text || '')
    .replace(/\s+/g, ' ')
    .trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

/**
 * Static HTML for /self-learner — GitHub Pages cannot issue a 301, so crawlers and
 * no-JS clients follow via meta refresh + canonical + visible fallback link (HS.2 FR-3).
 */
function buildLegacyAudienceRedirectHtml(siteOrigin = SITE_ORIGIN) {
  const canonical = `${String(siteOrigin).replace(/\/$/, '')}/homeschool`
  const escapedCanonical = escapeHtml(canonical)
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta http-equiv="refresh" content="0; url=/homeschool" />
  <link rel="canonical" href="${escapedCanonical}" />
  <title>Moved to Homeschool — Lextures</title>
</head>
<body>
  <p>This page has moved to <a href="/homeschool">Homeschool</a>.</p>
</body>
</html>
`
}

function buildHeadTags({ title, description, canonical, image, jsonLd }) {
  const t = escapeHtml(title)
  const d = escapeHtml(description)
  const c = escapeHtml(canonical)
  const img = escapeHtml(image || DEFAULT_OG_IMAGE)
  const lines = [
    `<title>${t}</title>`,
    `<meta name="description" content="${d}" />`,
    `<link rel="canonical" href="${c}" />`,
    `<meta property="og:title" content="${t}" />`,
    `<meta property="og:description" content="${d}" />`,
    `<meta property="og:image" content="${img}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:url" content="${c}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${t}" />`,
    `<meta name="twitter:description" content="${d}" />`,
    `<meta name="twitter:image" content="${img}" />`,
  ]
  if (jsonLd) {
    lines.push(
      `<script type="application/ld+json" id="course-json-ld">${JSON.stringify(jsonLd)}</script>`,
    )
  }
  return lines.join('\n    ')
}

function injectHead(shellHtml, headTags) {
  let html = shellHtml
  // Replace existing title
  html = html.replace(/<title>[^<]*<\/title>/i, () => {
    const m = headTags.match(/<title>[\s\S]*?<\/title>/i)
    return m ? m[0] : '<title>Lextures</title>'
  })
  // Replace or insert description
  if (/<meta\s+name=["']description["']/i.test(html)) {
    html = html.replace(
      /<meta\s+name=["']description["'][^>]*>/i,
      () => {
        const m = headTags.match(/<meta name="description"[^>]*>/i)
        return m ? m[0] : ''
      },
    )
  }
  // Strip prior OG/Twitter/canonical/json-ld we manage, then inject before </head>
  html = html.replace(/<link\s+rel=["']canonical["'][^>]*>\s*/gi, '')
  html = html.replace(/<meta\s+property=["']og:[^"']+["'][^>]*>\s*/gi, '')
  html = html.replace(/<meta\s+name=["']twitter:[^"']+["'][^>]*>\s*/gi, '')
  html = html.replace(
    /<script\s+type=["']application\/ld\+json["'][^>]*>[\s\S]*?<\/script>\s*/gi,
    '',
  )
  // Re-inject full managed head block (title already replaced; skip duplicate title/description)
  const inject = headTags
    .split('\n')
    .map(l => l.trim())
    .filter(l => l && !/^<title>/i.test(l) && !/^<meta name="description"/i.test(l))
    .join('\n    ')
  html = html.replace(/<\/head>/i, `    ${inject}\n  </head>`)
  return html
}

async function fetchJSON(url) {
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`GET ${url} → ${res.status} ${body.slice(0, 200)}`)
  }
  return res.json()
}

async function fetchAllCourses() {
  const courses = []
  let cursor = ''
  for (;;) {
    const qs = new URLSearchParams({ limit: '50', sort: 'newest' })
    if (cursor) qs.set('cursor', cursor)
    const data = await fetchJSON(`${API_BASE}/api/v1/public/marketplace/courses?${qs}`)
    courses.push(...(data.courses || []))
    cursor = data.nextCursor || ''
    if (!cursor) break
  }
  return courses
}

async function mapPool(items, concurrency, fn) {
  const results = new Array(items.length)
  let i = 0
  async function worker() {
    while (i < items.length) {
      const idx = i++
      results[idx] = await fn(items[idx], idx)
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, () => worker()))
  return results
}

function buildSitemap(courseEntries) {
  const today = new Date().toISOString().slice(0, 10)
  const urls = [
    ...STATIC_ROUTES.map(
      r => `  <url>
    <loc>${SITE_ORIGIN}${r.loc === '/' ? '/' : r.loc}</loc>
    <lastmod>${today}</lastmod>
    <priority>${r.priority}</priority>
  </url>`,
    ),
    ...courseEntries.map(
      c => `  <url>
    <loc>${SITE_ORIGIN}/courses/${encodeURIComponent(c.slug)}</loc>
    <lastmod>${(c.lastmod || today).slice(0, 10)}</lastmod>
    <priority>0.8</priority>
  </url>`,
    ),
  ]
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.join('\n')}
</urlset>
`
}

function buildRobots() {
  return `User-agent: *
Allow: /

Allow: /courses
Allow: /courses/

Sitemap: ${SITE_ORIGIN}/sitemap.xml
`
}

async function main() {
  const shellPath = path.join(DIST, 'index.html')
  let shell
  try {
    shell = await readFile(shellPath, 'utf8')
  } catch {
    console.error('[prerender] dist/index.html missing — run vite build first')
    process.exit(1)
  }

  let courses = []
  if (SKIP) {
    console.warn('[prerender] SKIP_COURSE_PRERENDER=1 — writing /courses index only')
  } else {
    try {
      courses = await fetchAllCourses()
    } catch (err) {
      console.error('[prerender] Failed to fetch marketplace courses:', err.message || err)
      console.error('[prerender] Set SKIP_COURSE_PRERENDER=1 to deploy without course pages.')
      process.exit(1)
    }
  }

  // /courses index
  const coursesIndexHead = buildHeadTags({
    title: 'Courses — Lextures',
    description:
      'Browse free and paid courses on Lextures. Learn at your own pace with adaptive quizzes and spaced review.',
    canonical: `${SITE_ORIGIN}/courses`,
  })
  await mkdir(path.join(DIST, 'courses'), { recursive: true })
  await writeFile(path.join(DIST, 'courses', 'index.html'), injectHead(shell, coursesIndexHead), 'utf8')

  const sitemapCourses = []
  let prerendered = 1 // /courses

  if (!SKIP && courses.length > 0) {
    await mapPool(courses, CONCURRENCY, async course => {
      const slug = course.slug || course.courseCode
      if (!slug) return
      let detail
      try {
        detail = await fetchJSON(
          `${API_BASE}/api/v1/public/marketplace/courses/${encodeURIComponent(slug)}`,
        )
      } catch (err) {
        console.warn(`[prerender] skip ${slug}: ${err.message || err}`)
        return
      }
      const c = detail.course || course
      const head = buildHeadTags({
        title: `${c.title} — Lextures`,
        description: truncateMeta(c.description || c.title),
        canonical: `${SITE_ORIGIN}/courses/${encodeURIComponent(slug)}`,
        image: resolveApiAssetUrl(c.heroImageUrl) || DEFAULT_OG_IMAGE,
        jsonLd: detail.jsonLd || null,
      })
      const dir = path.join(DIST, 'courses', slug)
      await mkdir(dir, { recursive: true })
      await writeFile(path.join(dir, 'index.html'), injectHead(shell, head), 'utf8')
      sitemapCourses.push({ slug, lastmod: c.createdAt })
      prerendered++
    })
  }

  // /self-learner → /homeschool redirect stub (must survive indefinitely; HS.2 FR-3)
  await mkdir(path.join(DIST, 'self-learner'), { recursive: true })
  await writeFile(
    path.join(DIST, 'self-learner', 'index.html'),
    buildLegacyAudienceRedirectHtml(SITE_ORIGIN),
    'utf8',
  )

  await writeFile(path.join(DIST, 'sitemap.xml'), buildSitemap(sitemapCourses), 'utf8')
  await writeFile(path.join(DIST, 'robots.txt'), buildRobots(), 'utf8')

  console.log(
    `[prerender] Done: ${prerendered} HTML page(s), ${sitemapCourses.length} course URL(s) in sitemap, /self-learner redirect stub (${SITE_ORIGIN})`,
  )
}

// Exported for unit tests
export {
  escapeHtml,
  truncateMeta,
  buildHeadTags,
  injectHead,
  buildSitemap,
  buildRobots,
  buildLegacyAudienceRedirectHtml,
  resolveApiAssetUrl,
}

const isMain =
  process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isMain) {
  main().catch(err => {
    console.error(err)
    process.exit(1)
  })
}
