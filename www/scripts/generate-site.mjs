#!/usr/bin/env node
/**
 * Static site generator for the www marketing site (SEO.1 + SEO.2).
 *
 * After `vite build`, uses Vite SSR + react-dom/server renderToString to write
 * dist/<path>/index.html for every route in the manifest, plus:
 *   - dist/.seo-manifest.json
 *   - dist/_redirects / _headers
 *   - dist/sitemap.xml (index) + dist/sitemaps/*.xml
 *   - dist/robots.txt (from crawler-policy.ts)
 *   - dist/llms.txt + dist/llms-full.txt
 *   - dist content .md siblings for blog/docs pages
 *   - dist/{indexnow-key}.txt
 *   - dist/404.html (real noindex 404)
 *   - redirect stubs (e.g. /self-learner)
 *   - course pages with marketplace API + previous-deploy fallback
 *
 * Env:
 *   API_BASE / VITE_API_BASE_URL — marketplace API (default https://self.lextures.com)
 *   SITE_ORIGIN — canonical origin (default https://lextures.com)
 *   COURSE_CACHE_URL — optional URL of previous deploy course HTML base
 *   GENERATE_CONCURRENCY — bounded pool size (default 8)
 *   ROBOTS_DISALLOW_ALL — if "1", emit staging robots (Disallow: /)
 *
 * Marketing articles are always loaded from the public content API. The
 * previous-deploy cache is the only availability fallback after MC.15.
 */

import { createServer as createViteServer } from 'vite'
import { mkdir, readFile, writeFile, access } from 'node:fs/promises'
import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'
import sharp from 'sharp'
import { fetchWithRetry, loadApiContent, mergeRedirects, contentHreflangAlternates } from './content-source.mjs'
import { buildFeeds } from './feeds.mjs'
import { renderOgCard } from './og-card/render.mjs'
import {
  assertSitemapManifestParity,
  buildLlmsFullTxt,
  buildImageSitemap,
  buildSitemapArtifacts,
  buildUrlset,
  markdownSiblingPath,
  normalizeLastmod,
  resolveLastmod,
  shouldEmitMarkdownSibling,
  sitemapSectionForPath,
} from './seo-artifacts.mjs'
import {
  buildCourseCache,
  courseSeoTitle,
  evaluateCourseQuality,
} from './marketplace-seo.mjs'

const execFileAsync = promisify(execFile)

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const DIST = path.join(ROOT, 'dist')
const PACKAGE_VERSION = createRequire(import.meta.url)(path.join(ROOT, 'package.json')).version || '0'

const API_BASE = (
  process.env.API_BASE ||
  process.env.VITE_API_BASE_URL ||
  'https://self.lextures.com'
).replace(/\/$/, '')
const SITE_ORIGIN = (process.env.SITE_ORIGIN || 'https://lextures.com').replace(/\/$/, '')
const LIVE_ORIGIN = (process.env.COURSE_CACHE_URL || SITE_ORIGIN).replace(/\/$/, '')
const CONCURRENCY = Math.max(1, Number(process.env.GENERATE_CONCURRENCY || 8))
const ROBOTS_DISALLOW_ALL =
  process.env.ROBOTS_DISALLOW_ALL === '1' ||
  process.env.ROBOTS_DISALLOW_ALL === 'true' ||
  (SITE_ORIGIN !== 'https://lextures.com' && process.env.ROBOTS_DISALLOW_ALL !== '0')
const PRERENDER_UA = `lextures-www-prerender/${PACKAGE_VERSION}`
const CONTENT_SOURCE = 'api'
const CONTENT_API_BASE = (process.env.CONTENT_API_BASE || API_BASE).replace(/\/$/, '')
const CONTENT_CACHE_DIR = path.resolve(ROOT, process.env.CONTENT_CACHE_DIR || '.content-cache')
const FORCE_COURSE_REBUILD = process.env.FORCE_COURSE_REBUILD === '1'
const MAX_RETRIES = 3
const MAX_DESC = 160

/** Map static routes → primary source files for git lastmod (SEO.2 FR-7). */
const STATIC_SOURCE_FILES = {
  '/': ['src/pages/home-page.tsx', 'src/lib/route-manifest.tsx'],
  '/about': ['src/pages/about-page.tsx', 'src/lib/schema/entity.ts'],
  '/authors': ['src/pages/authors-index-page.tsx', 'src/lib/authors.ts'],
  '/get-started': ['src/pages/get-started-page.tsx'],
  '/parents': ['src/pages/parents-page.tsx'],
  '/higher-ed': ['src/pages/higher-ed-page.tsx'],
  '/k12': ['src/pages/k12-page.tsx'],
  '/homeschool': ['src/pages/homeschool-page.tsx'],
  '/pricing': ['src/pages/pricing-page.tsx', 'src/lib/institution-pricing.ts'],
  '/pricing/calculator': ['src/pages/pricing-calculator-page.tsx'],
  '/courses': ['src/pages/courses-page.tsx'],
  '/request-information': ['src/pages/request-information-page.tsx'],
  '/blog': ['src/pages/blog-index.tsx'],
  '/docs': ['src/pages/docs-index.tsx'],
  '/privacy': ['src/pages/legal-pages.tsx', 'src/content/legal/privacy-policy.md'],
  '/privacy/history': ['src/pages/legal-pages.tsx', 'src/content/legal/privacy-history.md'],
  '/terms': ['src/pages/legal-pages.tsx', 'src/content/legal/terms-of-service.md'],
  '/terms/history': ['src/pages/legal-pages.tsx', 'src/content/legal/terms-history.md'],
  '/security': ['src/pages/security-page.tsx'],
  '/accessibility': ['src/pages/accessibility-conformance-page.tsx'],
  '/accessibility/vpat': ['src/pages/vpat-page.tsx', 'src/lib/vpat-data.ts'],
  '/privacy-rights/california': ['src/pages/california-privacy-rights-page.tsx'],
  '/404': ['src/pages/not-found-page.tsx'],
}

/** @type {Map<string, string | null>} */
const gitDateCache = new Map()

// ── pure helpers (also unit-tested) ──────────────────────────────────────────

export function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function truncateMeta(text, maxLen = 160) {
  const cleaned = String(text || '')
    .replace(/\s+/g, ' ')
    .trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

/** Soft cap for SEO titles — keep in sync with `document-head.ts`. */
export function truncateTitle(text, maxLen = 60) {
  const cleaned = String(text || '')
    .replace(/\s+/g, ' ')
    .trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 20 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

function decodeBasicEntities(value) {
  return String(value || '')
    .replace(/&quot;/g, '"')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
}

/**
 * Normalize HTML copied from a previous deploy during content-API fallback:
 * enforce the 60-char title budget and expose the og:image URL so the build
 * can materialize the raster into `dist/`.
 */
export function normalizeFallbackContentHtml(html) {
  const source = String(html || '')
  const titleRaw = source.match(/<title>([^<]+)<\/title>/i)?.[1] || ''
  const title = truncateTitle(decodeBasicEntities(titleRaw))
  const escapedTitle = escapeHtml(title)
  let next = source.replace(/<title>[^<]*<\/title>/i, `<title>${escapedTitle}</title>`)
  for (const property of ['og:title', 'twitter:title']) {
    const reAttrFirst = new RegExp(
      `(<meta\\b[^>]*property=["']${property}["'][^>]*content=["'])[^"']*(["'])`,
      'i',
    )
    const reContentFirst = new RegExp(
      `(<meta\\b[^>]*content=["'])[^"']*(["'][^>]*property=["']${property}["'])`,
      'i',
    )
    if (reAttrFirst.test(next)) next = next.replace(reAttrFirst, `$1${escapedTitle}$2`)
    else if (reContentFirst.test(next)) next = next.replace(reContentFirst, `$1${escapedTitle}$2`)
  }
  const image =
    decodeBasicEntities(
      next.match(/<meta\b[^>]*property=["']og:image["'][^>]*content=["']([^"']+)/i)?.[1] ||
        next.match(/<meta\b[^>]*content=["']([^"']+)["'][^>]*property=["']og:image["']/i)?.[1] ||
        '',
    )
  return { html: next, title, image }
}

/** Always emit @graph envelope (SEO.3 FR-1) with script-safe escaping (FR-3). */
export function serializeJsonLd(nodes) {
  let graphNodes
  if (Array.isArray(nodes)) {
    if (nodes.length === 1 && nodes[0] && Array.isArray(nodes[0]['@graph'])) {
      graphNodes = nodes[0]['@graph']
    } else {
      graphNodes = nodes
    }
  } else if (nodes && Array.isArray(nodes['@graph'])) {
    graphNodes = nodes['@graph']
  } else {
    graphNodes = nodes ? [nodes] : []
  }
  const cleaned = graphNodes.map(n => {
    if (!n || typeof n !== 'object') return n
    const { ['@context']: _c, ...rest } = n
    return rest
  })
  const payload = { '@context': 'https://schema.org', '@graph': cleaned }
  return JSON.stringify(payload)
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026')
    .replace(/\u2028/g, '\\u2028')
    .replace(/\u2029/g, '\\u2029')
}

/**
 * Lightweight graph validation for the build (SEO.3 FR-5).
 * Full TypeScript validators live in src/lib/schema/graph.ts; this mirrors the critical checks.
 */
export function validateJsonLdGraph(nodes, path = '') {
  const errors = []
  if (!nodes) return errors
  let list = Array.isArray(nodes) ? nodes : [nodes]
  if (list.length === 1 && list[0] && Array.isArray(list[0]['@graph'])) {
    list = list[0]['@graph']
  }
  const defined = new Set()
  for (const node of list) {
    if (!node || typeof node !== 'object') continue
    const id = node['@id']
    if (typeof id !== 'string' || !id) {
      errors.push(`${path}: JSON-LD node missing @id (type=${node['@type'] || '?'})`)
      continue
    }
    if (!/^https?:\/\//i.test(id)) {
      errors.push(`${path}: JSON-LD @id is not absolute: ${id}`)
    }
    if (!node['@type']) {
      errors.push(`${path}: JSON-LD node ${id} missing @type`)
    }
    defined.add(id)
  }
  const collectRefs = value => {
    if (value == null) return
    if (Array.isArray(value)) {
      for (const v of value) collectRefs(v)
      return
    }
    if (typeof value !== 'object') return
    const keys = Object.keys(value)
    if (keys.length === 1 && keys[0] === '@id' && typeof value['@id'] === 'string') {
      if (!defined.has(value['@id'])) {
        errors.push(`${path}: dangling JSON-LD @id ref ${value['@id']}`)
      }
      return
    }
    for (const [k, v] of Object.entries(value)) {
      if (k === '@id' || k === '@type' || k === '@context') continue
      collectRefs(v)
    }
  }
  for (const node of list) collectRefs(node)

  const serialized = serializeJsonLd(list)
  if (Buffer.byteLength(serialized, 'utf8') > 12 * 1024) {
    errors.push(
      `${path}: JSON-LD payload exceeds 12 KB (${Buffer.byteLength(serialized, 'utf8')} bytes)`,
    )
  }
  return errors
}

export function collectSchemaTypes(jsonLd) {
  if (!jsonLd) return []
  const types = []
  const visit = node => {
    if (!node || typeof node !== 'object') return
    if (Array.isArray(node)) {
      for (const n of node) visit(n)
      return
    }
    if (node['@graph']) {
      visit(node['@graph'])
      return
    }
    if (node['@type']) {
      const t = node['@type']
      if (Array.isArray(t)) types.push(...t.map(String))
      else types.push(String(t))
    }
  }
  visit(jsonLd)
  return types
}

export function buildHeadTags({ title, description, canonical, image, imageAlt, jsonLd, robots, markdownAlternate, alternates = [], feeds = false }) {
  const t = escapeHtml(title)
  const d = escapeHtml(description)
  const c = escapeHtml(canonical)
  const img = escapeHtml(image || `${SITE_ORIGIN}/assets/og-default.png`)
  const imgAlt = escapeHtml(imageAlt || `${title} — Lextures`)
  const r = escapeHtml(robots || 'index,follow')
  const lines = [
    `<title>${t}</title>`,
    `<meta name="description" content="${d}" />`,
    `<meta name="robots" content="${r}" />`,
    `<link rel="canonical" href="${c}" />`,
    `<meta property="og:title" content="${t}" />`,
    `<meta property="og:description" content="${d}" />`,
    `<meta property="og:image" content="${img}" />`,
    `<meta property="og:image:width" content="1200" />`,
    `<meta property="og:image:height" content="630" />`,
    `<meta property="og:image:alt" content="${imgAlt}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:url" content="${c}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${t}" />`,
    `<meta name="twitter:description" content="${d}" />`,
    `<meta name="twitter:image" content="${img}" />`,
    `<meta name="twitter:image:alt" content="${imgAlt}" />`,
  ]
  if (markdownAlternate) {
    lines.push(
      `<link rel="alternate" type="text/markdown" href="${escapeHtml(markdownAlternate)}" />`,
    )
  }
  if (feeds) {
    lines.push(`<link rel="alternate" type="application/rss+xml" title="Lextures Blog" href="${SITE_ORIGIN}/blog/feed.xml" />`)
    lines.push(`<link rel="alternate" type="application/feed+json" title="Lextures Blog" href="${SITE_ORIGIN}/blog/feed.json" />`)
  }
  for (const alternate of alternates) {
    lines.push(`<link rel="alternate" hreflang="${escapeHtml(alternate.hreflang)}" href="${escapeHtml(alternate.href)}" />`)
  }
  if (jsonLd) {
    lines.push(
      `<script type="application/ld+json" id="site-json-ld">${serializeJsonLd(jsonLd)}</script>`,
    )
  }
  return lines.join('\n    ')
}

/**
 * Find the hashed static-island entry produced by Vite multi-entry build.
 * @param {string} shellHtml
 * @param {string} [distDir]
 */
export async function findStaticIslandSrc(shellHtml, distDir) {
  // Prefer an asset already referenced; else scan dist/assets for static-island-*.js
  const fromShell = shellHtml.match(/src=["']([^"']*static-island[^"']*\.js)["']/i)
  if (fromShell) return fromShell[1]
  try {
    const { readdir } = await import('node:fs/promises')
    const assetsDir = path.join(distDir || DIST, 'assets')
    const files = await readdir(assetsDir)
    const hit = files.find(f => f.startsWith('static-island') && f.endsWith('.js'))
    if (hit) return `/assets/${hit}`
  } catch {
    /* ignore */
  }
  return null
}

export function injectDocument(
  shellHtml,
  { headTags, bodyHtml, ssrData, interactive = true, staticIslandSrc = null, locale = 'en', dir = 'ltr' },
) {
  let html = shellHtml

  // Drop SPA-restore and homepage-specific meta; generator owns head for each page.
  html = html.replace(/<!--\s*SPA-RESTORE[\s\S]*?-->/i, '')
  html = html.replace(/<script>\s*\/\/ Restore clean URLs[\s\S]*?<\/script>\s*/i, '')
  html = html.replace(/<script>\s*;\(function \(l\) \{[\s\S]*?\}\)\(window\.location\)\s*<\/script>\s*/i, '')

  // SEO.4 FR-8 — never leave Google Fonts on any page
  html = html.replace(/<link[^>]+fonts\.googleapis\.com[^>]*>\s*/gi, '')
  html = html.replace(/<link[^>]+fonts\.gstatic\.com[^>]*>\s*/gi, '')
  html = html.replace(/<link[^>]+preconnect[^>]+fonts\.(googleapis|gstatic)\.com[^>]*>\s*/gi, '')

  // Mark interactivity for the client entry (SEO.4 FR-4)
  html = html.replace(/<html\b([^>]*)>/i, (_m, attrs) => {
    const cleaned = String(attrs || '')
      .replace(/\s*data-interactive=(["']).*?\1/i, '')
      .replace(/\s*lang=(["']).*?\1/i, '')
      .replace(/\s*dir=(["']).*?\1/i, '')
    return `<html${cleaned} lang="${escapeHtml(locale)}" dir="${dir === 'rtl' ? 'rtl' : 'ltr'}" data-interactive="${interactive ? 'true' : 'false'}">`
  })

  // Replace title
  if (/<title>[^<]*<\/title>/i.test(html)) {
    html = html.replace(/<title>[^<]*<\/title>/i, () => {
      const m = headTags.match(/<title>[\s\S]*?<\/title>/i)
      return m ? m[0] : '<title>Lextures</title>'
    })
  } else {
    html = html.replace(/<\/head>/i, `    <title>Lextures</title>\n  </head>`)
  }

  // Replace or insert description. The Vite shell often has no description meta
  // (generator owns per-route head); without an insert path here, the tag is
  // filtered out of the inject block below and never appears in dist HTML.
  const descriptionTag = (headTags.match(/<meta name="description"[^>]*>/i) || [])[0]
  if (descriptionTag) {
    if (/<meta\s+name=["']description["']/i.test(html)) {
      html = html.replace(/<meta\s+name=["']description["'][^>]*>/i, descriptionTag)
    } else if (/<title>[^<]*<\/title>/i.test(html)) {
      html = html.replace(/<title>[^<]*<\/title>/i, (title) => `${title}\n    ${descriptionTag}`)
    } else {
      html = html.replace(/<\/head>/i, `    ${descriptionTag}\n  </head>`)
    }
  }

  // Strip managed tags then inject full block (skip title/description already handled)
  html = html.replace(/<link\s+rel=["']canonical["'][^>]*>\s*/gi, '')
  html = html.replace(/<link\s+rel=["']alternate["'][^>]*type=["']text\/markdown["'][^>]*>\s*/gi, '')
  html = html.replace(/<meta\s+name=["']robots["'][^>]*>\s*/gi, '')
  html = html.replace(/<meta\s+property=["']og:[^"']+["'][^>]*>\s*/gi, '')
  html = html.replace(/<meta\s+name=["']twitter:[^"']+["'][^>]*>\s*/gi, '')
  html = html.replace(
    /<script\s+type=["']application\/ld\+json["'][^>]*>[\s\S]*?<\/script>\s*/gi,
    '',
  )
  html = html.replace(/<script>\s*window\.__LEXTURES_SSR__\s*=[\s\S]*?<\/script>\s*/gi, '')

  const inject = headTags
    .split('\n')
    .map(l => l.trim())
    .filter(l => l && !/^<title>/i.test(l) && !/^<meta name="description"/i.test(l))
    .join('\n    ')

  const ssrScript =
    interactive && ssrData && Object.keys(ssrData).length
      ? `\n    <script>window.__LEXTURES_SSR__=${JSON.stringify(ssrData).replace(/</g, '\\u003c')}</script>`
      : ''

  html = html.replace(/<\/head>/i, `    ${inject}${ssrScript}\n  </head>`)

  // Inject body into #root
  if (/<div id="root"><\/div>/i.test(html)) {
    html = html.replace(
      /<div id="root"><\/div>/i,
      `<div id="root">${bodyHtml}</div>`,
    )
  } else if (/<div id="root">[\s\S]*?<\/div>/i.test(html)) {
    html = html.replace(
      /<div id="root">[\s\S]*?<\/div>/i,
      `<div id="root">${bodyHtml}</div>`,
    )
  }

  // SEO.4 FR-4 — content pages: swap main React entry for the tiny static-island.
  if (!interactive) {
    const island = staticIslandSrc
    // Remove all module scripts (index/main entry + any island from shell).
    html = html.replace(
      /<script\s+type=["']module["'][^>]*src=["'][^"']+["'][^>]*>\s*<\/script>\s*/gi,
      '',
    )
    if (island) {
      html = html.replace(
        /<\/body>/i,
        `  <script type="module" src="${island}"></script>\n  </body>`,
      )
    }
  } else {
    // Ensure static-island is not also loaded on interactive pages from multi-entry shell.
    html = html.replace(
      /<script\s+type=["']module["'][^>]*src=["'][^"']*static-island[^"']*["'][^>]*>\s*<\/script>\s*/gi,
      '',
    )
  }

  return html
}

export function buildRedirectsFile(rules) {
  const lines = rules.map(r => `${r.from} ${r.to} ${r.status}`)
  lines.push('/*    /404.html   404')
  return `${lines.join('\n')}\n`
}

export function buildRedirectStubHtml(toPath, siteOrigin = SITE_ORIGIN) {
  const origin = String(siteOrigin).replace(/\/$/, '')
  const target = toPath.startsWith('/') ? toPath : `/${toPath}`
  const canonical = `${origin}${target === '/' ? '/' : target.replace(/\/+$/, '')}`
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta http-equiv="refresh" content="0; url=${escapeHtml(target)}" />
  <link rel="canonical" href="${escapeHtml(canonical)}" />
  <meta name="robots" content="noindex,follow" />
  <title>Moved — Lextures</title>
</head>
<body>
  <p>This page has moved to <a href="${escapeHtml(target)}">${escapeHtml(target)}</a>.</p>
</body>
</html>
`
}

/** @deprecated alias used by tests migrating from prerender-courses */
export function buildLegacyAudienceRedirectHtml(siteOrigin = SITE_ORIGIN) {
  return buildRedirectStubHtml('/homeschool', siteOrigin)
}

/**
 * Flat urlset builder (unit tests / single-file consumers).
 * Production uses buildSitemapArtifacts (sitemap index + sections).
 * lastmod is omitted when unknown — never fabricates build-date (SEO.2 FR-7).
 */
export function buildSitemap(entries, siteOrigin = SITE_ORIGIN) {
  return buildUrlset(
    entries
      .filter(e => e.sitemap !== false)
      .map(e => ({
        path: e.path || e.loc,
        lastmod: e.lastmod,
        priority: e.priority,
        changefreq: e.changefreq,
        alternates: e.alternates,
      })),
    siteOrigin,
  )
}

/** Fallback robots when crawler-policy cannot load (tests). Prefer policy module. */
export function buildRobots(siteOrigin = SITE_ORIGIN, opts = {}) {
  const origin = String(siteOrigin).replace(/\/$/, '')
  if (opts.disallowAll) {
    return `User-agent: *\nDisallow: /\n\nSitemap: ${origin}/sitemap.xml\n`
  }
  return `User-agent: *\nAllow: /\nDisallow: /404\nDisallow: /*?*\n\nSitemap: ${origin}/sitemap.xml\n`
}

/**
 * Git commit date (ISO) for a repo-relative path under www/, or null.
 * @param {string} relativeFile path relative to www/ root
 */
export async function gitLastmod(relativeFile) {
  if (!relativeFile) return null
  if (gitDateCache.has(relativeFile)) return gitDateCache.get(relativeFile)
  try {
    const { stdout } = await execFileAsync(
      'git',
      ['log', '-1', '--format=%cI', '--', relativeFile],
      { cwd: ROOT, maxBuffer: 64 * 1024 },
    )
    const iso = String(stdout || '').trim()
    const day = normalizeLastmod(iso)
    gitDateCache.set(relativeFile, day)
    return day
  } catch {
    gitDateCache.set(relativeFile, null)
    return null
  }
}

/**
 * Latest git date among several files.
 * @param {string[]} files
 */
export async function gitLastmodMany(files) {
  let best = null
  for (const f of files) {
    const d = await gitLastmod(f)
    if (d && (!best || d > best)) best = d
  }
  return best
}

/**
 * Resolve honest lastmod for a route (SEO.2 FR-7).
 * @param {{ path: string, lastmod?: string, descriptor?: { lastmodSource?: string } }} route
 * @param {{ courseUpdatedAt?: string, courseCreatedAt?: string, contentMeta?: Record<string, string> }} [extra]
 */
export async function resolveRouteLastmod(route, extra = {}) {
  const contentMeta = extra.contentMeta || {}
  let gitDate = null
  const p = route.path

  if ((p.startsWith('/blog/') && p !== '/blog') || (p.startsWith('/docs/') && p !== '/docs') || (p.startsWith('/courses/') && p !== '/courses')) {
    // Database/API content uses API timestamps only.
    gitDate = null
  } else {
    const files = STATIC_SOURCE_FILES[p] || ['src/lib/route-manifest.tsx']
    gitDate = await gitLastmodMany(files)
  }

  return resolveLastmod({
    contentUpdatedAt: extra.contentUpdatedAt,
    publishedAt: extra.publishedAt,
    frontmatterUpdated: contentMeta.updated || contentMeta.updatedAt,
    frontmatterDate: contentMeta.date || route.lastmod,
    gitDate,
    courseUpdatedAt: extra.courseUpdatedAt,
    courseCreatedAt: extra.courseCreatedAt || (p.startsWith('/courses/') ? route.lastmod : null),
  })
}

export function resolveApiAssetUrl(url, apiBase = API_BASE) {
  if (!url) return null
  const trimmed = String(url).trim()
  if (!trimmed) return null
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) return trimmed
  if (trimmed.startsWith('/')) return `${apiBase}${trimmed}`
  return trimmed
}

export function outputPathForRoute(routePath) {
  if (routePath === '/' || routePath === '') return path.join(DIST, 'index.html')
  const clean = routePath.replace(/^\//, '').replace(/\/+$/, '')
  return path.join(DIST, clean, 'index.html')
}

export function validateGeneratedPage({ path: routePath, head, bodyHtml }) {
  const errors = []
  if (!bodyHtml || !String(bodyHtml).trim()) {
    errors.push(`${routePath}: empty body`)
  }
  if (!head?.title || !String(head.title).trim()) {
    errors.push(`${routePath}: missing title`)
  }
  if (!head?.description || !String(head.description).trim()) {
    errors.push(`${routePath}: missing description`)
  } else if (String(head.description).length > MAX_DESC) {
    errors.push(`${routePath}: description > ${MAX_DESC} chars`)
  }
  if (!head?.canonical) {
    errors.push(`${routePath}: missing canonical`)
  }
  return errors
}

export function buildLinkGraph(pages) {
  const paths = new Set(pages.map(page => page.path))
  const edges = []
  for (const page of pages) {
    const anchorRe = /<a\b[^>]*\bhref=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi
    let match
    while ((match = anchorRe.exec(page.html || ''))) {
      let href = match[1].split('#')[0].split('?')[0].replace(/\/+$/, '') || '/'
      if (!href.startsWith('/') || !paths.has(href)) continue
      const anchor = match[2].replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
      edges.push({ from: page.path, to: href, anchor })
    }
  }
  const depth = new Map([['/', 0]])
  const queue = ['/']
  while (queue.length) {
    const from = queue.shift()
    for (const edge of edges) if (edge.from === from && !depth.has(edge.to)) {
      depth.set(edge.to, depth.get(from) + 1); queue.push(edge.to)
    }
  }
  const nodes = [...paths].sort().map(routePath => ({
    path: routePath, depth: depth.get(routePath) ?? null,
    inbound: edges.filter(edge => edge.to === routePath).length,
    outbound: edges.filter(edge => edge.from === routePath).length,
  }))
  return { nodes, edges }
}

export function validateLinkGraph(graph) {
  const errors = []
  for (const node of graph.nodes) {
    if (node.path !== '/' && node.inbound === 0) errors.push(`${node.path}: orphan page`)
    if (node.depth == null) errors.push(`${node.path}: unreachable from homepage`)
    else if (node.depth > 3) errors.push(`${node.path}: depth ${node.depth} exceeds 3`)
  }
  for (const edge of graph.edges) {
    if (/^(click here|read more|https?:\/\/)/i.test(edge.anchor)) errors.push(`${edge.from}: non-descriptive anchor "${edge.anchor}"`)
  }
  return errors
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
  await Promise.all(
    Array.from({ length: Math.min(concurrency, Math.max(items.length, 1)) }, () => worker()),
  )
  return results
}

async function sleep(ms) {
  return new Promise(r => setTimeout(r, ms))
}

async function fetchJSON(url, attempt = 1) {
  const res = await fetch(url, {
    headers: {
      Accept: 'application/json',
      'User-Agent': PRERENDER_UA,
    },
  })
  if (res.status === 429 || res.status >= 500) {
    if (attempt < MAX_RETRIES) {
      const retryAfter = Number(res.headers.get('Retry-After') || 0)
      const delay = retryAfter > 0 ? retryAfter * 1000 : 250 * 2 ** (attempt - 1)
      await sleep(delay)
      return fetchJSON(url, attempt + 1)
    }
  }
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

async function selfHostCourseImages(courses) {
  const outputDir = path.join(DIST, 'assets', 'course-images')
  await mkdir(outputDir, { recursive: true })
  await mapPool(courses, Math.min(CONCURRENCY, 4), async course => {
    if (!course.heroImageUrl) return
    try {
      const sourceUrl = new URL(course.heroImageUrl, API_BASE)
      const response = await fetch(sourceUrl, { headers: { 'User-Agent': PRERENDER_UA } })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const input = Buffer.from(await response.arrayBuffer())
      const slug = String(course.slug || course.courseCode || course.id).replace(/[^a-z0-9-]/gi, '-')
      const base = path.join(outputDir, slug)
      await Promise.all([
        sharp(input).resize({ width: 960, withoutEnlargement: true }).avif({ quality: 55 }).toFile(`${base}.avif`),
        sharp(input).resize({ width: 960, withoutEnlargement: true }).webp({ quality: 72 }).toFile(`${base}.webp`),
      ])
      course.heroImageUrl = `/assets/course-images/${slug}.avif`
    } catch (err) {
      console.warn(`[generate-site] course image fallback (${course.slug || course.id}): ${err.message || err}`)
    }
  })
}

async function fetchPreviousCourseHtml(slug) {
  const url = `${LIVE_ORIGIN}/courses/${encodeURIComponent(slug)}/`
  try {
    const res = await fetch(url, {
      headers: { 'User-Agent': PRERENDER_UA, Accept: 'text/html' },
    })
    if (!res.ok) return null
    const html = await res.text()
    if (!html || !html.includes('<div id="root">')) return null
    return html
  } catch {
    return null
  }
}

async function fetchPreviousCourseCache() {
  if (FORCE_COURSE_REBUILD) return null
  try {
    const res = await fetch(`${LIVE_ORIGIN}/.course-cache.json`, {
      headers: { 'User-Agent': PRERENDER_UA, Accept: 'application/json' },
    })
    if (!res.ok) return null
    const data = await res.json()
    return data && typeof data.courses === 'object' ? data : null
  } catch {
    return null
  }
}

async function fileExists(p) {
  try {
    await access(p)
    return true
  } catch {
    return false
  }
}

function extractCourseSlugsFromSitemap(xml, liveOrigin) {
  const origin = String(liveOrigin).replace(/\/$/, '')
  const re = new RegExp(
    `${origin.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/courses/([^<\\s/]+)`,
    'g',
  )
  const slugs = []
  let m
  while ((m = re.exec(xml)) !== null) {
    const slug = decodeURIComponent(m[1].replace(/\/$/, ''))
    if (slug && slug !== 'index.html') slugs.push(slug)
  }
  return slugs
}

async function discoverPreviousContentPaths() {
  const paths = []
  for (const section of ['blog', 'docs']) {
    try {
      const response = await fetch(`${LIVE_ORIGIN}/sitemaps/${section}.xml`, { headers: { 'User-Agent': PRERENDER_UA } })
      if (!response.ok) continue
      const xml = await response.text()
      for (const match of xml.matchAll(/<loc>([^<]+)<\/loc>/g)) {
        const pathname = new URL(match[1]).pathname.replace(/\/$/, '') || '/'
        if ((section === 'blog' && /^\/blog\/[^/]+$/.test(pathname)) || (section === 'docs' && /^\/docs\/[^/]+\/[^/]+$/.test(pathname))) paths.push(pathname)
      }
    } catch { /* previous deploy is best effort */ }
  }
  return [...new Set(paths)]
}

// ── main ─────────────────────────────────────────────────────────────────────

async function main() {
  const started = Date.now()
  const shellPath = path.join(DIST, 'index.html')
  let shell
  try {
    shell = await readFile(shellPath, 'utf8')
  } catch {
    console.error('[generate-site] dist/index.html missing — run vite build first')
    process.exit(1)
  }

  // Vite SSR server to load TSX modules (no HMR / watch — we only need ssrLoadModule)
  const vite = await createViteServer({
    root: ROOT,
    server: { middlewareMode: true, watch: null, hmr: false },
    appType: 'custom',
    // Don't re-write dist during SSR
    build: { emptyOutDir: false },
  })

  const contentSnapshot = await loadApiContent({ apiBase: CONTENT_API_BASE, cacheDir: CONTENT_CACHE_DIR, concurrency: CONCURRENCY, userAgent: PRERENDER_UA })
  await localizeContentMedia(contentSnapshot.articles)
  globalThis.__LEXTURES_BUILD_CONTENT__ = contentSnapshot

  let renderPath
  let enumerateConcreteRoutes
  let validateLocaleManifest
  let REDIRECTS
  let flattenAndValidateRedirects
  let renderRobotsTxt
  let renderLlmsTxt
  let INDEXNOW_KEY
  let INDEXNOW_KEY_FILENAME
  try {
    const entry = await vite.ssrLoadModule('/src/entry-server.tsx')
    renderPath = entry.renderPath
    enumerateConcreteRoutes = entry.enumerateConcreteRoutes
    validateLocaleManifest = entry.validateLocaleManifest
    const redirectsMod = await vite.ssrLoadModule('/src/lib/redirects.ts')
    REDIRECTS = redirectsMod.REDIRECTS
    flattenAndValidateRedirects = redirectsMod.flattenAndValidateRedirects
    const crawlerMod = await vite.ssrLoadModule('/src/lib/crawler-policy.ts')
    renderRobotsTxt = crawlerMod.renderRobotsTxt
    const llmsMod = await vite.ssrLoadModule('/src/lib/llms-catalog.ts')
    renderLlmsTxt = llmsMod.renderLlmsTxt
    const indexNowMod = await vite.ssrLoadModule('/src/lib/indexnow-key.ts')
    INDEXNOW_KEY = indexNowMod.INDEXNOW_KEY
    INDEXNOW_KEY_FILENAME = indexNowMod.INDEXNOW_KEY_FILENAME
  } catch (err) {
    console.error('[generate-site] Failed to load SSR modules:', err)
    await vite.close()
    process.exit(1)
  }
  REDIRECTS = mergeRedirects(REDIRECTS, contentSnapshot.redirects || [])
  console.log(`[generate-site] Content: source=${CONTENT_SOURCE} articles=${contentSnapshot.articles.length} fetched=${contentSnapshot.fetched || 0} cacheHits=${contentSnapshot.cacheHits || 0} fallbackUsed=${Boolean(contentSnapshot.fallbackUsed)}`)

  // Courses: fetch listing; degrade on failure (FR-7)
  let courses = []
  let courseApiFailed = false
  let previousCourseCache = null
  try {
    courses = await fetchAllCourses()
    previousCourseCache = await fetchPreviousCourseCache()
    await selfHostCourseImages(courses)
    console.log(`[generate-site] Marketplace API: ${courses.length} course(s)`)
  } catch (err) {
    courseApiFailed = true
    console.warn(
      `[generate-site] WARN: marketplace API unreachable (${err.message || err}); reusing previous-deploy course HTML when available`,
    )
  }

  const coursePaths = []
  const courseDetails = new Map() // path → detail
  const courseQuality = new Map() // path → quality report
  const catalogHubData = new Map() // path → coursesIndex payload

  if (!courseApiFailed && courses.length > 0) {
    await mapPool(courses, CONCURRENCY, async course => {
      const slug = course.slug || course.courseCode
      if (!slug) return
      try {
        const detail = await fetchJSON(
          `${API_BASE}/api/v1/public/marketplace/courses/${encodeURIComponent(slug)}`,
        )
        const c = detail.course || course
        if (course.heroImageUrl?.startsWith('/assets/course-images/')) {
          c.heroImageUrl = course.heroImageUrl
        }
        const routePath = `/courses/${slug}`
        const quality = evaluateCourseQuality(detail, courses)
        const courseLastmod = resolveLastmod({
          courseUpdatedAt: c.updatedAt,
          courseCreatedAt: c.createdAt,
        })
        coursePaths.push({
          path: routePath,
          title: truncateMeta(courseSeoTitle(c), 60),
          description: truncateMeta(c.description || c.title),
          lastmod: courseLastmod || undefined,
          priority: '0.8',
          courseUpdatedAt: c.updatedAt || null,
          courseCreatedAt: c.createdAt || null,
        })
        courseDetails.set(routePath, detail)
        courseQuality.set(routePath, quality)
      } catch (err) {
        console.warn(`[generate-site] skip course ${slug}: ${err.message || err}`)
      }
    })
  }

  const hubSlug = value => String(value || '').trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  for (const [dimension, field] of [['subject', 'category'], ['level', 'level']]) {
    const groups = new Map()
    for (const course of courses) {
      const value = course[field]
      const slug = hubSlug(value)
      if (!slug) continue
      const group = groups.get(slug) || { value, courses: [] }
      group.courses.push(course)
      groups.set(slug, group)
    }
    for (const [slug, group] of groups) {
      if (group.courses.length < 3) continue
      const hubPath = `/courses/${dimension}/${slug}`
      coursePaths.push({
        path: hubPath,
        title: `${group.value} courses — Lextures`,
        description: truncateMeta(`Explore ${group.value} courses on Lextures, including course content, levels, prices, and enrollment options.`),
        priority: '0.7',
      })
      catalogHubData.set(hubPath, { courses: group.courses.slice(0, 24), total: group.courses.length })
    }
  }

  const apiContentMissing = CONTENT_SOURCE === 'api' && !(contentSnapshot.articles || []).length
  const previousContentPaths = CONTENT_SOURCE === 'api' && (contentSnapshot.fallbackUsed || apiContentMissing)
    ? await discoverPreviousContentPaths() : []
  const extraContentPaths = (contentSnapshot.articles || [])
    .filter(article => article.path && article.locale && article.locale !== 'en')
    .map(article => ({
      path: article.path,
      title: article.title ? `${article.title} — Lextures` : undefined,
      description: article.description,
    }))
  const concrete = enumerateConcreteRoutes([...coursePaths, ...extraContentPaths])
  const localeManifestErrors = validateLocaleManifest()
  if (localeManifestErrors.length) {
    for (const error of localeManifestErrors) console.error(`[generate-site] ${error}`)
    await vite.close()
    process.exit(1)
  }
  REDIRECTS = flattenAndValidateRedirects(
    [...concrete.filter(route => route.path !== '/404').map(route => route.path), ...previousContentPaths],
    REDIRECTS,
    { onMissingTarget: apiContentMissing ? 'omit' : 'throw' },
  )
  // Attach priority from manifest where present
  for (const route of concrete) {
    if (!route.priority && route.descriptor?.priority) {
      route.priority = route.descriptor.priority
    }
    if (route.descriptor?.sitemap === false) {
      route.sitemap = false
    } else {
      route.sitemap = route.descriptor?.sitemap !== false
    }
    const quality = courseQuality.get(route.path)
    if (quality && !quality.indexable) {
      route.robots = 'noindex,follow'
      route.sitemap = false
    }
  }

  const seoUrls = []
  const titles = new Map()
  const descriptions = new Map()
  const allErrors = []
  /** @type {Array<{ path: string, title: string, body: string, robots?: string, sourceRoot?: string }>} */
  const contentForLlmsFull = []
  let slowest = { path: '', ms: 0 }
  let reusedCourses = 0
  let generated = 0
  let markdownSiblings = 0
  const generatedPages = []
  const imageSitemapEntries = []

  async function loadContentMeta(routePath) {
    const article = contentSnapshot.articles.find(item => item.path === routePath)
    if (!article) return null
    return {
      meta: {
        title: article.title, description: article.description,
        author: article.author?.slug || 'chase-willden',
        updated: article.contentUpdatedAt || article.updatedAt,
        date: article.publishedAt,
      },
      body: article.bodyMd || '', sourceFile: null,
      sourceRoot: article.kind === 'blog' ? 'blog' : 'docs', article,
    }
  }

  const staticIslandSrc = await findStaticIslandSrc(shell, DIST)

  async function writeRoute(route, ssrData = {}) {
    const t0 = Date.now()
    const content = await loadContentMeta(route.path)
    const course = ssrData.courseDetail?.course
    const lastmod = await resolveRouteLastmod(route, {
      contentMeta: content?.meta,
      contentUpdatedAt: content?.article?.contentUpdatedAt,
      publishedAt: content?.article?.publishedAt,
      courseUpdatedAt: course?.updatedAt || route.courseUpdatedAt,
      courseCreatedAt: course?.createdAt || route.courseCreatedAt || route.lastmod,
    })

    const markdownAlternate = shouldEmitMarkdownSibling(route.path)
      ? `${SITE_ORIGIN}${route.path}.md`
      : null

    const result = await renderPath(route.path, ssrData)
    const ms = Date.now() - t0
    if (ms > slowest.ms) slowest = { path: route.path, ms }

    const head = {
      ...result.head,
      title: route.title || result.head.title,
      description: route.description || result.head.description,
      robots: route.robots || result.head.robots,
      canonical: route.canonical || result.head.canonical,
      markdownAlternate: markdownAlternate || undefined,
      feeds: route.path === '/blog' || /^\/(?:[a-z]{2}(?:-[a-z0-9]+)?\/)?blog\/[^/]+$/.test(route.path),
    }
    if (content?.article?.canonicalOverride) head.canonical = content.article.canonicalOverride
    if (content?.article?.noindex) head.robots = 'noindex,follow'
    if (content?.article?.locale) {
      head.locale = content.article.locale
      head.dir = /^(ar)(-|$)/i.test(content.article.locale) ? 'rtl' : 'ltr'
    }
    const contentAlts = contentHreflangAlternates(content?.article, SITE_ORIGIN)
    if (contentAlts.length) head.alternates = contentAlts
    // Prefer course-resolved head from renderPath when present
    if (ssrData.courseDetail) {
      Object.assign(head, result.head, {
        robots: route.robots || result.head.robots,
        canonical: route.canonical || result.head.canonical,
        markdownAlternate: markdownAlternate || undefined,
      })
    }

    // SEO.14: an explicit hand-made raster image wins; otherwise generate a
    // deterministic, content-addressed social card. Generation failures are
    // non-fatal and use the checked-in raster default.
    const hero = content?.article?.media?.find(media => media.id === content.article.heroMediaId || media.usage === 'hero')
    const heroRendition = hero?.renditions?.find(item => Number(item.width) === 1200 && Number(item.height) === 630) || hero?.renditions?.find(item => item.name === 'social')
    const cardOverride = heroRendition ? `${SITE_ORIGIN}${localizedMediaPath(hero, heroRendition)}` : route.descriptor?.ogImage
    if (cardOverride && !/\.(png|jpe?g)(?:[?#]|$)/i.test(cardOverride)) {
      allErrors.push(`${route.path}: ogImage override must be a raster PNG or JPEG`)
    }
    if (!cardOverride) {
      try {
        const card = await renderOgCard({ title: head.title, routePath: route.path, distDir: DIST })
        head.image = `${SITE_ORIGIN}/${card.relativePath}`
      } catch (error) {
        console.warn(`[generate-site] WARN: OG card failed for ${route.path}: ${error.message || error}`)
        head.image = `${SITE_ORIGIN}/assets/og-default.png`
      }
    } else {
      head.image = cardOverride.startsWith('http') ? cardOverride : `${SITE_ORIGIN}${cardOverride}`
    }
    head.imageAlt = `${head.title} — Lextures social preview`

    const errors = validateGeneratedPage({
      path: route.path,
      head,
      bodyHtml: result.bodyHtml,
    })
    if (/^\/(?:[a-z]{2}(?:-[a-z0-9]+)?\/)?blog\/[^/]+$/.test(route.path) || /^\/(?:[a-z]{2}(?:-[a-z0-9]+)?\/)?docs\/[^/]+\/[^/]+$/.test(route.path)) {
      const contextual = result.bodyHtml.match(/data-contextual-links[\s\S]*?<\/p>/)?.[0] || ''
      const contextualCount = (contextual.match(/<a\b/g) || []).length
      if (contextualCount < 3) errors.push(`${route.path}: needs at least 3 contextual internal links`)
      const related = result.bodyHtml.match(/data-related-content[\s\S]*?<\/aside>/)?.[0] || ''
      const relatedCount = (related.match(/<a\b/g) || []).length
      if (relatedCount < 3 || relatedCount > 6) errors.push(`${route.path}: Related module must contain 3–6 links`)
    }
    if (titles.has(head.title) && titles.get(head.title) !== route.path) {
      errors.push(`${route.path}: duplicate title "${head.title}" (also ${titles.get(head.title)})`)
    }
    if (descriptions.has(head.description) && descriptions.get(head.description) !== route.path) {
      // History/thin pages may share short descriptions — warn only for indexable
      if ((head.robots || 'index,follow').includes('index')) {
        errors.push(
          `${route.path}: duplicate description (also ${descriptions.get(head.description)})`,
        )
      }
    }
    titles.set(head.title, route.path)
    descriptions.set(head.description, route.path)
    allErrors.push(...errors)

    // SEO.3 FR-5 — graph shape + dangling refs + 12 KB budget
    if (head.jsonLd) {
      allErrors.push(...validateJsonLdGraph(head.jsonLd, route.path))
    }

    // SEO.3 FR-20 — author slug must resolve (front-matter already validated in TS modules;
    // re-check content meta when present)
    if (content?.meta?.author) {
      const slug = String(content.meta.author).trim()
      // Author registry is loaded via SSR modules; unknown slugs fail when vite loads blog.ts.
      // Keep a soft check for raw front-matter drift.
      if (!slug || slug.includes(' ')) {
        allErrors.push(
          `${route.path}: author must be a registry slug (got "${content.meta.author}")`,
        )
      }
    }

    const headTags = buildHeadTags(head)
    const interactive = result.interactive !== false && route.descriptor?.interactive !== false
    const html = injectDocument(shell, {
      headTags,
      bodyHtml: result.bodyHtml,
      ssrData: Object.keys(ssrData).length ? { ...ssrData, path: route.path } : { path: route.path },
      interactive,
      staticIslandSrc,
      locale: head.locale,
      dir: head.dir,
    })

    const outFile = outputPathForRoute(route.path)
    await mkdir(path.dirname(outFile), { recursive: true })
    await writeFile(outFile, html, 'utf8')
    generatedPages.push({ path: route.path, html, noindex: String(head.robots || '').includes('noindex') })
    generated++

    // Markdown sibling (SEO.2 FR-15)
    if (markdownAlternate && content?.body != null) {
      const mdPath = markdownSiblingPath(route.path, DIST)
      await mkdir(path.dirname(mdPath), { recursive: true })
      await writeFile(mdPath, content.body.trim() + '\n', 'utf8')
      markdownSiblings++
    }

    if (content?.body && (content.sourceRoot === 'blog' || content.sourceRoot === 'docs')) {
      contentForLlmsFull.push({
        path: route.path,
        title: content.meta?.title || head.title,
        body: content.body,
        robots: head.robots || 'index,follow',
        sourceRoot: content.sourceRoot,
      })
    }

    const schemaTypes = collectSchemaTypes(head.jsonLd)

    const sitemapFlag = route.sitemap !== false && route.descriptor?.sitemap !== false
    seoUrls.push({
      path: route.path,
      title: head.title,
      description: head.description,
      canonical: head.canonical,
      lastmod: lastmod || null,
      robots: head.robots || 'index,follow',
      sitemap: sitemapFlag,
      section: sitemapSectionForPath(route.path),
      priority: route.priority || route.descriptor?.priority || (route.path === '/' ? '1.0' : '0.5'),
      changefreq: route.descriptor?.changefreq,
      markdownAlternate: markdownAlternate || null,
      schemaTypes,
      locale: head.locale || 'en',
      translationOf: route.descriptor?.translationOf || null,
      translationStatus: route.descriptor?.translationStatus || null,
      sourceUpdatedAt: route.descriptor?.sourceUpdatedAt || null,
      alternates: head.alternates || [],
      bytes: Buffer.byteLength(html, 'utf8'),
    })
    if (sitemapFlag && !(head.robots || '').includes('noindex')) {
      imageSitemapEntries.push({ path: route.path, image: head.image, title: head.title, caption: head.imageAlt })
    }
  }

  // Render non-course routes first, then courses
  const staticRoutes = concrete.filter(r => !r.path.startsWith('/courses/') || r.path === '/courses')
  const courseRoutes = concrete.filter(r => r.path.startsWith('/courses/') && r.path !== '/courses')

  await mapPool(staticRoutes, CONCURRENCY, async route => {
    const ssrData = CONTENT_SOURCE === 'api' ? { articleIndex: contentSnapshot } : {}
    if (CONTENT_SOURCE === 'api' && (/^\/blog\/[^/]+$/.test(route.path) || /^\/docs\/[^/]+\/[^/]+$/.test(route.path))) {
      ssrData.article = contentSnapshot.articles.find(article => article.path === route.path) || null
    }
    if (route.path === '/courses' && courses.length > 0) {
      ssrData.coursesIndex = {
        courses: courses.slice(0, 12),
        total: courses.length,
      }
    } else if (catalogHubData.has(route.path)) {
      ssrData.coursesIndex = catalogHubData.get(route.path)
    }
    await writeRoute(route, ssrData)
  })

  if (previousContentPaths.length) {
    await mapPool(previousContentPaths, CONCURRENCY, async routePath => {
      try {
        const response = await fetch(`${LIVE_ORIGIN}${routePath}`, { headers: { 'User-Agent': PRERENDER_UA } })
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        const normalized = normalizeFallbackContentHtml(await response.text())
        const html = normalized.html
        const outFile = outputPathForRoute(routePath)
        await mkdir(path.dirname(outFile), { recursive: true }); await writeFile(outFile, html, 'utf8')
        const read = pattern => html.match(pattern)?.[1]?.replace(/&quot;/g, '"').replace(/&amp;/g, '&') || ''
        const title = normalized.title || read(/<title>([^<]+)<\/title>/i)
        const description = read(/<meta\s+name=["']description["']\s+content=["']([^"']*)/i)
        const canonical = read(/<link\s+rel=["']canonical["']\s+href=["']([^"']*)/i) || `${SITE_ORIGIN}${routePath}`
        const robots = read(/<meta\s+name=["']robots["']\s+content=["']([^"']*)/i) || 'index,follow'
        if (normalized.image) {
          try {
            const imageUrl = new URL(normalized.image, SITE_ORIGIN)
            if ((imageUrl.origin === SITE_ORIGIN || imageUrl.origin === LIVE_ORIGIN) && imageUrl.pathname.startsWith('/og/')) {
              const imageFile = path.join(DIST, imageUrl.pathname.slice(1))
              try {
                await access(imageFile)
              } catch {
                const imageResponse = await fetch(`${LIVE_ORIGIN}${imageUrl.pathname}`, { headers: { 'User-Agent': PRERENDER_UA } })
                if (imageResponse.ok) {
                  await mkdir(path.dirname(imageFile), { recursive: true })
                  await writeFile(imageFile, Buffer.from(await imageResponse.arrayBuffer()))
                } else {
                  console.warn(`[generate-site] WARN: previous OG image unavailable for ${routePath}: HTTP ${imageResponse.status}`)
                }
              }
              if (!robots.includes('noindex')) {
                imageSitemapEntries.push({
                  path: routePath,
                  image: `${SITE_ORIGIN}${imageUrl.pathname}`,
                  title,
                  caption: `${title} — Lextures social preview`,
                })
              }
            }
          } catch (error) {
            console.warn(`[generate-site] WARN: previous OG image unavailable for ${routePath}: ${error.message || error}`)
          }
        }
        generatedPages.push({ path: routePath, html, noindex: robots.includes('noindex'), fallback: true })
        seoUrls.push({ path: routePath, title, description, canonical, lastmod: null, robots, sitemap: !robots.includes('noindex'), section: sitemapSectionForPath(routePath), priority: '0.6', schemaTypes: [], bytes: Buffer.byteLength(html) })
        generated++; contentSnapshot.fallbackUsed = true
        try {
          const md = await fetch(`${LIVE_ORIGIN}${routePath}.md`, { headers: { 'User-Agent': PRERENDER_UA } })
          if (md.ok) { const mdPath = markdownSiblingPath(routePath, DIST); await mkdir(path.dirname(mdPath), { recursive: true }); await writeFile(mdPath, await md.text(), 'utf8'); markdownSiblings++ }
        } catch { /* optional sibling */ }
      } catch (error) { console.warn(`[generate-site] WARN: previous content unavailable for ${routePath}: ${error.message || error}`) }
    })
  }

  if (courseRoutes.length > 0) {
    await mapPool(courseRoutes, CONCURRENCY, async route => {
      if (catalogHubData.has(route.path)) {
        await writeRoute(route, { coursesIndex: catalogHubData.get(route.path) })
        return
      }
      const detail = courseDetails.get(route.path)
      const course = detail?.course
      const slug = course?.slug || course?.courseCode
      const version = course?.updatedAt || course?.createdAt || ''
      const unchanged = Boolean(
        slug && previousCourseCache?.courses?.[slug] === version && !FORCE_COURSE_REBUILD,
      )
      if (unchanged) {
        const html = await fetchPreviousCourseHtml(slug)
        if (html) {
          const outFile = outputPathForRoute(route.path)
          await mkdir(path.dirname(outFile), { recursive: true })
          await writeFile(outFile, html, 'utf8')
          const quality = courseQuality.get(route.path)
          const robots = quality?.indexable === false ? 'noindex,follow' : 'index,follow'
          const title = truncateMeta(courseSeoTitle(course), 60)
          const description = truncateMeta(course.description || course.title)
          const canonical = `${SITE_ORIGIN}${route.path}`
          generatedPages.push({ path: route.path, html, noindex: robots.includes('noindex') })
          seoUrls.push({
            path: route.path, title, description, canonical,
            lastmod: normalizeLastmod(version), robots,
            sitemap: !robots.includes('noindex'), section: 'courses', priority: '0.8',
            schemaTypes: ['Course'], bytes: Buffer.byteLength(html, 'utf8'),
          })
          reusedCourses++
          return
        }
      }
      await writeRoute(route, { courseDetail: detail || null })
    })
  } else if (courseApiFailed) {
    // Reuse previous deploy course HTML from live site (FR-7 / AC-5)
    console.warn('[generate-site] WARN: attempting previous-deploy course HTML reuse')
    // We don't know slugs without API — try reading a prior .course-cache.json if present in dist
    // or fetch live sitemap for /courses/ URLs
    let slugs = []
    try {
      // Prefer section sitemap (SEO.2 index); fall back to root sitemap / legacy flat.
      const candidates = [
        `${LIVE_ORIGIN}/sitemaps/courses.xml`,
        `${LIVE_ORIGIN}/sitemap.xml`,
      ]
      for (const smUrl of candidates) {
        const smRes = await fetch(smUrl, {
          headers: { 'User-Agent': PRERENDER_UA },
        })
        if (!smRes.ok) continue
        const xml = await smRes.text()
        // If this is a sitemap index, follow child locs that mention courses
        if (xml.includes('<sitemapindex')) {
          const childRe = /<loc>([^<]+)<\/loc>/g
          let cm
          while ((cm = childRe.exec(xml)) !== null) {
            if (!/courses/i.test(cm[1])) continue
            try {
              const childRes = await fetch(cm[1], { headers: { 'User-Agent': PRERENDER_UA } })
              if (childRes.ok) {
                const childXml = await childRes.text()
                slugs.push(...extractCourseSlugsFromSitemap(childXml, LIVE_ORIGIN))
              }
            } catch {
              // ignore child
            }
          }
        } else {
          slugs.push(...extractCourseSlugsFromSitemap(xml, LIVE_ORIGIN))
        }
        if (slugs.length) break
      }
      slugs = [...new Set(slugs)]
    } catch {
      // ignore
    }

    for (const slug of slugs) {
      const html = await fetchPreviousCourseHtml(slug)
      if (!html) continue
      const dir = path.join(DIST, 'courses', slug)
      await mkdir(dir, { recursive: true })
      await writeFile(path.join(dir, 'index.html'), html, 'utf8')
      reusedCourses++
      seoUrls.push({
        path: `/courses/${slug}`,
        title: `Course — Lextures`,
        description: 'Marketplace course on Lextures',
        canonical: `${SITE_ORIGIN}/courses/${encodeURIComponent(slug)}`,
        lastmod: null, // unknown for reused HTML — omit (SEO.2 FR-7)
        robots: 'index,follow',
        sitemap: true,
        section: 'courses',
        priority: '0.8',
        schemaTypes: [],
        bytes: Buffer.byteLength(html, 'utf8'),
      })
    }
    if (reusedCourses > 0) {
      console.warn(`[generate-site] WARN: reused ${reusedCourses} course page(s) from ${LIVE_ORIGIN}`)
    } else {
      console.warn('[generate-site] WARN: no previous course HTML available; shipping without course detail pages')
    }
  }

  // Redirect stubs (FR-13)
  for (const rule of REDIRECTS || []) {
    const from = rule.from.replace(/\/+$/, '') || rule.from
    const dir = path.join(DIST, from.replace(/^\//, ''))
    await mkdir(dir, { recursive: true })
    await writeFile(
      path.join(dir, 'index.html'),
      buildRedirectStubHtml(rule.to, SITE_ORIGIN),
      'utf8',
    )
  }

  // Real 404 page (FR-11)
  const notFoundRoute = concrete.find(r => r.path === '/404')
  if (notFoundRoute) {
    const result = await renderPath('/404', {})
    const head = { ...result.head, robots: 'noindex,follow' }
    const html = injectDocument(shell, {
      headTags: buildHeadTags(head),
      bodyHtml: result.bodyHtml,
      ssrData: { path: '/404' },
      interactive: false,
      staticIslandSrc,
    })
    await writeFile(path.join(DIST, '404.html'), html, 'utf8')
    // Also write /404/index.html for direct navigation
    if (!(await fileExists(outputPathForRoute('/404')))) {
      await writeRoute(notFoundRoute, {})
    }
  }

  // _redirects for Cloudflare Pages (FR-14)
  await writeFile(
    path.join(DIST, '_redirects'),
    buildRedirectsFile(REDIRECTS || []),
    'utf8',
  )

  // Optional _headers (Cloudflare / Netlify). GitHub Pages ignores these.
  await writeFile(
    path.join(DIST, '_headers'),
    `/*
  X-Content-Type-Options: nosniff

/assets/*
  Cache-Control: public, max-age=31536000, immutable

/og/*
  Cache-Control: public, max-age=31536000, immutable

/*.md
  Content-Type: text/plain; charset=utf-8
  X-Robots-Tag: noindex

/llms.txt
  Content-Type: text/plain; charset=utf-8

/llms-full.txt
  Content-Type: text/plain; charset=utf-8

/robots.txt
  Content-Type: text/plain; charset=utf-8
`,
    'utf8',
  )

  // Sitemap index + section sitemaps (SEO.2 FR-6…FR-11)
  const sitemapEntries = seoUrls
    .filter(u => {
      if (u.path === '/404') return false
      if ((u.robots || '').includes('noindex')) return false
      if (u.sitemap === false) return false
      return true
    })
    .map(u => ({
      path: u.path,
      lastmod: u.lastmod,
      priority: u.priority || (u.path === '/' ? '1.0' : '0.5'),
      changefreq: u.changefreq,
      section: u.locale && u.locale !== 'en' ? `locale-${u.locale.toLowerCase()}` : (u.section || sitemapSectionForPath(u.path)),
      // hreflang reserved for SEO.17 — emit nothing when unset
      alternates: u.alternates,
    }))

  const { indexXml, files: sitemapFiles } = buildSitemapArtifacts(sitemapEntries, SITE_ORIGIN)
  await mkdir(path.join(DIST, 'sitemaps'), { recursive: true })
  for (const f of sitemapFiles) {
    await writeFile(path.join(DIST, f.relativePath), f.xml, 'utf8')
  }
  await writeFile(path.join(DIST, 'sitemap.xml'), indexXml, 'utf8')
  await writeFile(path.join(DIST, 'sitemaps/images.xml'), buildImageSitemap(imageSitemapEntries, SITE_ORIGIN), 'utf8')

  // robots.txt from typed policy (SEO.2 FR-1…FR-5)
  const robotsTxt = renderRobotsTxt({
    siteOrigin: SITE_ORIGIN,
    disallowAll: ROBOTS_DISALLOW_ALL,
  })
  await writeFile(path.join(DIST, 'robots.txt'), robotsTxt, 'utf8')

  // llms.txt + llms-full.txt (SEO.2 FR-12…FR-14)
  let llmsTxt = renderLlmsTxt(SITE_ORIGIN)
  let previousLlmsFull = null
  if (CONTENT_SOURCE === 'api' && contentSnapshot.fallbackUsed) {
    try {
      const [curated, full] = await Promise.all([fetch(`${LIVE_ORIGIN}/llms.txt`), fetch(`${LIVE_ORIGIN}/llms-full.txt`)])
      if (curated.ok && full.ok) { llmsTxt = await curated.text(); previousLlmsFull = await full.text() }
    } catch { /* retain locally generated artifacts when previous deploy is unavailable */ }
  }
  await writeFile(path.join(DIST, 'llms.txt'), llmsTxt, 'utf8')
  const llmsFull = previousLlmsFull || buildLlmsFullTxt(contentForLlmsFull, SITE_ORIGIN)
  await writeFile(path.join(DIST, 'llms-full.txt'), llmsFull, 'utf8')
  const llmsFullBytes = Buffer.byteLength(llmsFull, 'utf8')
  if (llmsFullBytes > 5 * 1024 * 1024) {
    allErrors.push(`llms-full.txt exceeds 5 MB (${llmsFullBytes} bytes)`)
  }

  const feedPosts = contentSnapshot.articles.filter(article => article.kind === 'blog' && (!article.locale || article.locale === 'en')).map(article => ({
    ...article,
    content: article.bodyMd || article.content || '',
    date: String(article.publishedAt || article.date || '').slice(0, 10),
    author: article.author?.name || article.author?.slug || article.author || '',
    authorName: article.author?.name,
  }))
  let feeds = buildFeeds(feedPosts, SITE_ORIGIN)
  if (CONTENT_SOURCE === 'api' && contentSnapshot.fallbackUsed) {
    try {
      const [rss, json] = await Promise.all([fetch(`${LIVE_ORIGIN}/blog/feed.xml`), fetch(`${LIVE_ORIGIN}/blog/feed.json`)])
      if (rss.ok && json.ok) {
        const rssText = await rss.text(); const jsonText = await json.text()
        feeds = { rss: rssText, json: jsonText, itemCount: JSON.parse(jsonText).items?.length || 0 }
      }
    } catch { /* retain locally generated feeds when previous deploy is unavailable */ }
  }
  await mkdir(path.join(DIST, 'blog'), { recursive: true })
  await writeFile(path.join(DIST, 'blog/feed.xml'), feeds.rss, 'utf8')
  await writeFile(path.join(DIST, 'blog/feed.json'), feeds.json, 'utf8')

  // IndexNow key file (SEO.2 FR-17) — public by design
  await writeFile(path.join(DIST, INDEXNOW_KEY_FILENAME), `${INDEXNOW_KEY}\n`, 'utf8')

  // Course cache snapshot for next FR-7 degradation
  await writeFile(
    path.join(DIST, '.course-cache.json'),
    JSON.stringify(
      {
        ...buildCourseCache(courses),
        apiBase: API_BASE,
        slugs: coursePaths.map(c => c.path.replace(/^\/courses\//, '')),
      },
      null,
      2,
    ),
    'utf8',
  )

  await writeFile(
    path.join(DIST, '.catalog-quality.json'),
    JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        thresholdsVersion: 'SEO.11-v1',
        courses: Object.fromEntries(
          [...courseQuality].map(([routePath, quality]) => [routePath.replace(/^\/courses\//, ''), quality]),
        ),
        summary: {
          evaluated: courseQuality.size,
          indexable: [...courseQuality.values()].filter(result => result.indexable).length,
          noindex: [...courseQuality.values()].filter(result => !result.indexable).length,
        },
      },
      null,
      2,
    ),
    'utf8',
  )

  // SEO manifest (SEO.2 / SEO.16)
  const seoManifest = {
    generatedAt: new Date().toISOString(),
    contentSource: CONTENT_SOURCE,
    contentGeneratedAt: contentSnapshot.generatedAt || null,
    fallbackUsed: Boolean(contentSnapshot.fallbackUsed),
    articleCount: contentSnapshot.articles.length,
    feedItemCount: feeds.itemCount,
    origin: SITE_ORIGIN,
    indexNowKey: INDEXNOW_KEY,
    sitemaps: sitemapFiles.map(f => ({
      path: `/${f.relativePath}`,
      section: f.section,
      urlCount: f.count,
    })),
    urls: seoUrls.sort((a, b) => a.path.localeCompare(b.path)),
  }
  await writeFile(
    path.join(DIST, '.seo-manifest.json'),
    JSON.stringify(seoManifest, null, 2),
    'utf8',
  )

  // Internal/noindex references are intentionally not linked from the public IA.
  const graphPages = generatedPages.filter(page => page.path !== '/404' && !page.noindex && !page.fallback)
  const linkGraph = buildLinkGraph(graphPages)
  allErrors.push(...validateLinkGraph(linkGraph))
  await writeFile(path.join(DIST, '.link-graph.json'), JSON.stringify(linkGraph, null, 2), 'utf8')

  // Bidirectional sitemap ↔ manifest parity (FR-10 / AC-8)
  const parityErrors = assertSitemapManifestParity(
    sitemapEntries.map(e => e.path),
    seoUrls,
  )
  allErrors.push(...parityErrors)

  // Validate llms.txt links resolve to generated pages
  for (const u of seoUrls) {
    if (u.markdownAlternate && shouldEmitMarkdownSibling(u.path)) {
      const mdPath = markdownSiblingPath(u.path, DIST)
      if (!(await fileExists(mdPath))) {
        allErrors.push(`${u.path}: missing markdown sibling ${mdPath}`)
      }
    }
  }

  await vite.close()

  if (allErrors.length > 0) {
    console.error('[generate-site] Validation failed:')
    for (const e of allErrors) console.error(`  - ${e}`)
    process.exit(1)
  }

  const elapsed = ((Date.now() - started) / 1000).toFixed(1)
  const longestTitle = seoUrls.reduce(
    (a, u) => (u.title.length > a.length ? u.title : a),
    '',
  )
  console.log('[generate-site] Summary')
  console.log(`  routes generated : ${generated}`)
  console.log(`  courses reused   : ${reusedCourses}`)
  console.log(`  seo urls         : ${seoUrls.length}`)
  console.log(`  sitemap sections : ${sitemapFiles.map(f => `${f.relativePath}(${f.count})`).join(', ')}`)
  console.log(`  markdown siblings: ${markdownSiblings}`)
  console.log(`  llms-full.txt    : ${(llmsFullBytes / 1024).toFixed(1)} KiB`)
  console.log(`  indexnow key     : /${INDEXNOW_KEY_FILENAME}`)
  console.log(`  longest title    : ${longestTitle.length} chars`)
  console.log(`  slowest render   : ${slowest.path} (${slowest.ms}ms)`)
  console.log(`  elapsed          : ${elapsed}s`)
  console.log(`  origin           : ${SITE_ORIGIN}`)
  if (ROBOTS_DISALLOW_ALL) {
    console.warn('[generate-site] WARN: robots.txt disallows all (staging / non-production origin)')
  }
}

async function localizeContentMedia(articles) {
  const seen = new Map()
  await mapPool(articles, CONCURRENCY, async article => {
    for (const media of article.media || []) {
      const checksum = String(media.checksum || media.id).replace(/[^a-zA-Z0-9_-]/g, '')
      if (!seen.has(checksum)) seen.set(checksum, localizeOneMedia(media, checksum))
      const replacements = await seen.get(checksum)
      for (const [remote, local] of replacements) article.bodyMd = String(article.bodyMd || '').split(remote).join(local)
    }
  })
}

function localizedMediaPath(media, rendition) {
  const checksum = String(media.checksum || media.id).replace(/[^a-zA-Z0-9_-]/g, '')
  return `/assets/content/${checksum}/${rendition.name}.${rendition.ext}`
}

async function localizeOneMedia(media, checksum) {
  const dir = path.join(DIST, 'assets', 'content', checksum)
  await mkdir(dir, { recursive: true })
  const replacements = []
  let sourceBuffer = null
  for (const rendition of media.renditions || []) {
    const remote = resolveApiAssetUrl(rendition.url, CONTENT_API_BASE)
    if (!remote) continue
    const response = await fetchWithRetry(remote, { userAgent: PRERENDER_UA })
    const buffer = Buffer.from(await response.arrayBuffer())
    const filename = `${rendition.name}.${rendition.ext}`
    await writeFile(path.join(dir, filename), buffer)
    if (!sourceBuffer || rendition.name === 'original') sourceBuffer = buffer
    replacements.push([rendition.url, `/assets/content/${checksum}/${filename}`])
    replacements.push([remote, `/assets/content/${checksum}/${filename}`])
  }
  if (sourceBuffer) {
    const names = new Set((media.renditions || []).map(r => `${r.name}.${r.ext}`))
    if (![...names].some(name => name.endsWith('.webp'))) await sharp(sourceBuffer).webp().toFile(path.join(dir, 'generated.webp'))
    if (![...names].some(name => name.endsWith('.avif'))) await sharp(sourceBuffer).avif().toFile(path.join(dir, 'generated.avif'))
  }
  return replacements
}

// Back-compat exports matching prerender-courses.test.mjs names
export const injectHead = (shell, headTags) =>
  injectDocument(shell, { headTags, bodyHtml: '', ssrData: null })
export const STATIC_ROUTES = [
  { loc: '/', priority: '1.0' },
  { loc: '/pricing', priority: '0.8' },
  { loc: '/pricing/calculator', priority: '0.7' },
  { loc: '/docs', priority: '0.7' },
  { loc: '/blog', priority: '0.7' },
  { loc: '/courses', priority: '0.9' },
  { loc: '/get-started', priority: '0.8' },
  { loc: '/request-information', priority: '0.7' },
  { loc: '/higher-ed', priority: '0.6' },
  { loc: '/k12', priority: '0.6' },
  { loc: '/parents', priority: '0.6' },
  { loc: '/homeschool', priority: '0.6' },
  { loc: '/privacy', priority: '0.3' },
  { loc: '/privacy/history', priority: '0.2' },
  { loc: '/privacy-rights/california', priority: '0.3' },
  { loc: '/terms', priority: '0.3' },
  { loc: '/terms/history', priority: '0.2' },
  { loc: '/security', priority: '0.3' },
  { loc: '/accessibility', priority: '0.3' },
  { loc: '/accessibility/vpat', priority: '0.3' },
]

export function parseMarkdownDate(raw) {
  const match = String(raw || '').match(/^---\r?\n([\s\S]*?)\r?\n---/)
  if (!match) return null
  const dateLine = match[1].split('\n').find(l => /^\s*date\s*:/i.test(l))
  if (!dateLine) return null
  const value = dateLine
    .slice(dateLine.indexOf(':') + 1)
    .trim()
    .replace(/^["']|["']$/g, '')
  return value || null
}

const isMain =
  process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isMain) {
  main().catch(err => {
    console.error(err)
    process.exit(1)
  })
}
