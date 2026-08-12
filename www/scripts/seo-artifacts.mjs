/**
 * Pure SEO.2 artefact builders (sitemaps, lastmod, llms-full, IndexNow batching).
 * Unit-tested without Vite; generate-site.mjs orchestrates I/O.
 */

export const SITEMAP_MAX_URLS = 50_000
export const LLMS_FULL_MAX_BYTES = 5 * 1024 * 1024 // 5 MB
export const INDEXNOW_BATCH_SIZE = 10_000

/** Section names from SEO.2 FR-6. Empty sections are omitted from the index. */
export const SITEMAP_SECTIONS = [
  'pages',
  'blog',
  'docs',
  'courses',
  'compare',
  'glossary',
  'research',
]

/**
 * Assign a concrete path to a sitemap section.
 * @param {string} routePath
 * @returns {string}
 */
export function sitemapSectionForPath(routePath) {
  if (routePath === '/blog' || routePath.startsWith('/blog/')) return 'blog'
  if (routePath === '/docs' || routePath.startsWith('/docs/')) return 'docs'
  if (routePath === '/courses' || routePath.startsWith('/courses/')) return 'courses'
  if (routePath.startsWith('/compare') || routePath.startsWith('/vs/')) return 'compare'
  if (routePath.startsWith('/glossary')) return 'glossary'
  if (routePath.startsWith('/research')) return 'research'
  return 'pages'
}

/**
 * Normalize to YYYY-MM-DD or null (omit lastmod when unknown).
 * @param {unknown} value
 * @returns {string | null}
 */
export function normalizeLastmod(value) {
  if (value == null || value === '') return null
  const s = String(value).trim()
  // ISO date or datetime
  const day = s.slice(0, 10)
  if (/^\d{4}-\d{2}-\d{2}$/.test(day)) return day
  const d = new Date(s)
  if (!Number.isNaN(d.getTime())) return d.toISOString().slice(0, 10)
  return null
}

/**
 * lastmod resolution order (FR-7):
 * front-matter updated → front-matter date → git → course updatedAt/createdAt → omit.
 * Never falls back to "today" / build date.
 *
 * @param {{
 *   frontmatterUpdated?: string | null
 *   frontmatterDate?: string | null
 *   gitDate?: string | null
 *   courseUpdatedAt?: string | null
 *   courseCreatedAt?: string | null
 *   contentUpdatedAt?: string | null
 *   publishedAt?: string | null
 * }} sources
 * @returns {string | null}
 */
export function resolveLastmod(sources = {}) {
  const candidates = [
    sources.contentUpdatedAt,
    sources.publishedAt,
    sources.frontmatterUpdated,
    sources.frontmatterDate,
    sources.gitDate,
    sources.courseUpdatedAt,
    sources.courseCreatedAt,
  ]
  for (const c of candidates) {
    const n = normalizeLastmod(c)
    if (n) return n
  }
  return null
}

export function encodeLocPath(routePath) {
  if (!routePath || routePath === '/') return '/'
  return (
    '/' +
    routePath
      .replace(/^\//, '')
      .split('/')
      .map(seg => encodeURIComponent(seg))
      .join('/')
  )
}

/**
 * @param {string} routePath
 * @param {string} siteOrigin
 */
export function absoluteUrl(routePath, siteOrigin) {
  const origin = String(siteOrigin).replace(/\/$/, '')
  const pathPart = encodeLocPath(routePath)
  return pathPart === '/' ? `${origin}/` : `${origin}${pathPart}`
}

/**
 * Escape text for XML element content.
 * @param {string} value
 */
export function escapeXml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

/** Build an image sitemap containing each page's primary original/social image. */
export function buildImageSitemap(entries, siteOrigin) {
  const urls = entries.map(entry => `  <url>\n    <loc>${escapeXml(absoluteUrl(entry.path, siteOrigin))}</loc>\n    <image:image>\n      <image:loc>${escapeXml(entry.image)}</image:loc>\n      <image:title>${escapeXml(entry.title)}</image:title>\n      <image:caption>${escapeXml(entry.caption)}</image:caption>\n    </image:image>\n  </url>`)
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">\n${urls.join('\n')}\n</urlset>\n`
}

/**
 * Build a single urlset document.
 * @param {Array<{
 *   path: string
 *   lastmod?: string | null
 *   priority?: string
 *   changefreq?: string
 *   alternates?: Array<{ hreflang: string, href: string }>
 * }>} entries
 * @param {string} siteOrigin
 */
export function buildUrlset(entries, siteOrigin) {
  const hasAlternates = entries.some(e => e.alternates?.length)
  const ns = hasAlternates
    ? 'xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml"'
    : 'xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"'

  const urls = entries.map(e => {
    const loc = absoluteUrl(e.path, siteOrigin)
    const parts = [`    <loc>${escapeXml(loc)}</loc>`]
    const lastmod = normalizeLastmod(e.lastmod)
    if (lastmod) parts.push(`    <lastmod>${lastmod}</lastmod>`)
    if (e.changefreq) parts.push(`    <changefreq>${escapeXml(e.changefreq)}</changefreq>`)
    if (e.priority) parts.push(`    <priority>${escapeXml(e.priority)}</priority>`)
    if (e.alternates?.length) {
      for (const alt of e.alternates) {
        if (!alt?.hreflang || !alt?.href) continue
        parts.push(
          `    <xhtml:link rel="alternate" hreflang="${escapeXml(alt.hreflang)}" href="${escapeXml(alt.href)}" />`,
        )
      }
    }
    return `  <url>\n${parts.join('\n')}\n  </url>`
  })

  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset ${ns}>
${urls.join('\n')}
</urlset>
`
}

/**
 * Shard entries into chunks of SITEMAP_MAX_URLS.
 * @template T
 * @param {T[]} entries
 * @param {number} [maxUrls]
 * @returns {T[][]}
 */
export function shardEntries(entries, maxUrls = SITEMAP_MAX_URLS) {
  if (entries.length === 0) return []
  if (entries.length <= maxUrls) return [entries]
  const shards = []
  for (let i = 0; i < entries.length; i += maxUrls) {
    shards.push(entries.slice(i, i + maxUrls))
  }
  return shards
}

/**
 * Build section sitemap files + index.
 *
 * @param {Array<{
 *   path: string
 *   lastmod?: string | null
 *   priority?: string
 *   changefreq?: string
 *   section?: string
 *   alternates?: Array<{ hreflang: string, href: string }>
 * }>} entries indexable sitemap entries only
 * @param {string} siteOrigin
 * @returns {{
 *   indexXml: string
 *   files: Array<{ relativePath: string, xml: string, section: string, count: number }>
 * }}
 */
export function buildSitemapArtifacts(entries, siteOrigin) {
  const origin = String(siteOrigin).replace(/\/$/, '')
  /** @type {Map<string, typeof entries>} */
  const bySection = new Map()
  for (const e of entries) {
    const section = e.section || sitemapSectionForPath(e.path)
    if (!bySection.has(section)) bySection.set(section, [])
    bySection.get(section).push(e)
  }

  /** @type {Array<{ relativePath: string, xml: string, section: string, count: number }>} */
  const files = []
  /** @type {Array<{ loc: string, lastmod?: string | null }>} */
  const indexEntries = []

  const orderedSections = [
    ...SITEMAP_SECTIONS,
    ...[...bySection.keys()].filter(section => !SITEMAP_SECTIONS.includes(section)).sort(),
  ]
  for (const section of orderedSections) {
    const sectionEntries = bySection.get(section) || []
    if (sectionEntries.length === 0) continue

    const shards = shardEntries(sectionEntries)
    shards.forEach((shard, i) => {
      const name =
        shards.length === 1
          ? `${section}.xml`
          : `${section}-${i + 1}.xml`
      const relativePath = `sitemaps/${name}`
      const xml = buildUrlset(shard, origin)
      files.push({ relativePath, xml, section, count: shard.length })

      // Index lastmod = max child lastmod when any present
      let maxLast = null
      for (const u of shard) {
        const lm = normalizeLastmod(u.lastmod)
        if (lm && (!maxLast || lm > maxLast)) maxLast = lm
      }
      indexEntries.push({
        loc: `${origin}/${relativePath}`,
        lastmod: maxLast,
      })
    })
  }

  const indexXml = buildSitemapIndex(indexEntries)
  return { indexXml, files }
}

/**
 * @param {Array<{ loc: string, lastmod?: string | null }>} sitemaps
 */
export function buildSitemapIndex(sitemaps) {
  const body = sitemaps
    .map(s => {
      const parts = [`    <loc>${escapeXml(s.loc)}</loc>`]
      const lastmod = normalizeLastmod(s.lastmod)
      if (lastmod) parts.push(`    <lastmod>${lastmod}</lastmod>`)
      return `  <sitemap>\n${parts.join('\n')}\n  </sitemap>`
    })
    .join('\n')

  return `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${body}
</sitemapindex>
`
}

/**
 * Bidirectional parity between sitemap URLs and indexable seo-manifest URLs (FR-10).
 *
 * @param {string[]} sitemapPaths
 * @param {Array<{ path: string, robots?: string, sitemap?: boolean }>} manifestUrls
 * @returns {string[]} error messages (empty = ok)
 */
export function assertSitemapManifestParity(sitemapPaths, manifestUrls) {
  const errors = []
  const sitemapSet = new Set(sitemapPaths)
  const indexable = manifestUrls.filter(u => {
    if (u.path === '/404') return false
    if ((u.robots || 'index,follow').includes('noindex')) return false
    if (u.sitemap === false) return false
    return true
  })
  const indexableSet = new Set(indexable.map(u => u.path))

  for (const p of sitemapSet) {
    if (!indexableSet.has(p)) {
      errors.push(
        `sitemap URL absent from indexable .seo-manifest.json: ${p} (fix: add to manifest or remove from sitemap)`,
      )
    }
  }
  for (const p of indexableSet) {
    if (!sitemapSet.has(p)) {
      errors.push(
        `indexable manifest URL absent from sitemaps: ${p} (fix: set sitemap:true or robots noindex intentionally)`,
      )
    }
  }

  // noindex must never appear in sitemaps
  for (const u of manifestUrls) {
    if ((u.robots || '').includes('noindex') && sitemapSet.has(u.path)) {
      errors.push(`noindex URL must not appear in sitemaps: ${u.path}`)
    }
  }

  return errors
}

/**
 * Parse YAML-ish frontmatter from markdown.
 * @param {string} raw
 * @returns {{ meta: Record<string, string>, body: string }}
 */
export function parseFrontmatter(raw) {
  const match = String(raw || '').match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/)
  if (!match) return { meta: {}, body: String(raw || '') }
  /** @type {Record<string, string>} */
  const meta = {}
  for (const line of match[1].split('\n')) {
    const colon = line.indexOf(':')
    if (colon === -1) continue
    const key = line.slice(0, colon).trim()
    const value = line
      .slice(colon + 1)
      .trim()
      .replace(/^["']|["']$/g, '')
    meta[key] = value
  }
  return { meta, body: match[2].replace(/^\r?\n/, '') }
}

/**
 * Rewrite relative markdown links to absolute site URLs (best-effort).
 * @param {string} markdown
 * @param {string} siteOrigin
 * @param {string} pagePath path of the page (e.g. /docs/self-hosting)
 */
export function absolutizeMarkdownLinks(markdown, siteOrigin, pagePath) {
  const origin = String(siteOrigin).replace(/\/$/, '')
  const dir = pagePath === '/' ? '' : pagePath.replace(/\/[^/]+$/, '') || ''

  return String(markdown).replace(
    /\[([^\]]+)\]\(([^)]+)\)/g,
    (full, text, href) => {
      const h = href.trim()
      if (!h || h.startsWith('#') || h.startsWith('mailto:') || h.startsWith('tel:')) return full
      if (/^[a-z][a-z0-9+.-]*:/i.test(h)) return full // already absolute
      if (h.startsWith('//')) return `[${text}](https:${h})`
      if (h.startsWith('/')) return `[${text}](${origin}${h})`
      // relative to page dir
      const base = dir || ''
      const joined = `${base}/${h}`.replace(/\/+/g, '/')
      return `[${text}](${origin}${joined})`
    },
  )
}

/**
 * Build llms-full.txt from content documents.
 * Refuses legal-history and noindex paths (FR privacy).
 *
 * @param {Array<{
 *   path: string
 *   title: string
 *   body: string
 *   robots?: string
 *   sourceRoot?: string
 * }>} docs
 * @param {string} siteOrigin
 * @param {{ maxBytes?: number }} [opts]
 */
export function buildLlmsFullTxt(docs, siteOrigin, opts = {}) {
  const maxBytes = opts.maxBytes ?? LLMS_FULL_MAX_BYTES
  const origin = String(siteOrigin).replace(/\/$/, '')
  const allowed = docs.filter(d => {
    if ((d.robots || 'index,follow').includes('noindex')) return false
    if (d.sourceRoot && /legal/i.test(d.sourceRoot)) return false
    if (d.path.includes('/privacy/history') || d.path.includes('/terms/history')) return false
    if (d.path.startsWith('/privacy') && d.path.includes('history')) return false
    return true
  })

  const header = [
    '# Lextures — full text corpus (llms-full.txt)',
    `# Generated for AI agents. Prefer ${origin}/llms.txt for a curated map.`,
    `# Source: public help center + blog only. Legal history and noindex pages excluded.`,
    '',
  ].join('\n')

  let out = header
  let truncated = false
  let included = 0

  for (const doc of allowed) {
    const absBody = absolutizeMarkdownLinks(doc.body, origin, doc.path)
    const block = [
      '',
      '---',
      '',
      `# ${doc.title}`,
      '',
      `Source: ${origin}${doc.path === '/' ? '/' : doc.path}`,
      '',
      absBody.trim(),
      '',
    ].join('\n')

    const next = out + block
    if (Buffer.byteLength(next, 'utf8') > maxBytes) {
      truncated = true
      break
    }
    out = next
    included++
  }

  if (truncated) {
    out += [
      '',
      '---',
      '',
      `# TRUNCATED`,
      '',
      `Corpus capped at ${maxBytes} bytes. Included ${included} of ${allowed.length} documents.`,
      `See curated map: ${origin}/llms.txt`,
      '',
    ].join('\n')
  }

  return out
}

/**
 * Split URL list into IndexNow batches (FR-18).
 * @param {string[]} urls
 * @param {number} [batchSize]
 * @returns {string[][]}
 */
export function batchIndexNowUrls(urls, batchSize = INDEXNOW_BATCH_SIZE) {
  const unique = [...new Set(urls.filter(Boolean))]
  if (unique.length === 0) return []
  const batches = []
  for (let i = 0; i < unique.length; i += batchSize) {
    batches.push(unique.slice(i, i + batchSize))
  }
  return batches
}

/**
 * Diff two seo-manifest URL path lists → absolute URLs that changed or are new.
 * @param {{ urls?: Array<{ path: string, canonical?: string }> } | null} prev
 * @param {{ urls?: Array<{ path: string, canonical?: string }>, origin?: string }} next
 * @returns {string[]}
 */
export function diffManifestUrls(prev, next) {
  const origin = (next?.origin || 'https://lextures.com').replace(/\/$/, '')
  const prevMap = new Map((prev?.urls || []).map(u => [u.path, u]))
  const changed = []
  for (const u of next?.urls || []) {
    const p = prevMap.get(u.path)
    if (!p) {
      changed.push(u.canonical || absoluteUrl(u.path, origin))
      continue
    }
    // Treat any field change as worth submitting (title/desc/lastmod)
    const prevKey = JSON.stringify(p)
    const nextKey = JSON.stringify(u)
    if (prevKey !== nextKey) {
      changed.push(u.canonical || absoluteUrl(u.path, origin))
    }
  }
  return changed
}

/**
 * Build IndexNow POST body.
 * @param {{ host: string, key: string, keyLocation: string, urlList: string[] }} opts
 */
export function buildIndexNowBody(opts) {
  return {
    host: opts.host,
    key: opts.key,
    keyLocation: opts.keyLocation,
    urlList: opts.urlList,
  }
}

/**
 * Whether a page should emit a markdown sibling + alternate link.
 * @param {string} path
 */
export function shouldEmitMarkdownSibling(path) {
  if (/^\/(?:[a-z]{2}(?:-[a-z0-9]+)?\/)?blog\/[^/]+$/.test(path)) return true
  // Categorized help articles have two path segments after /docs. Category
  // hubs are HTML indexes and do not have a source Markdown document.
  if (/^\/(?:[a-z]{2}(?:-[a-z0-9]+)?\/)?docs\/[^/]+\/[^/]+$/.test(path)) return true
  return false
}

/**
 * Output path for a markdown sibling of a route.
 * @param {string} routePath e.g. /docs/self-hosting
 * @param {string} distRoot
 */
export function markdownSiblingPath(routePath, distRoot) {
  const clean = routePath.replace(/^\//, '').replace(/\/+$/, '')
  return `${distRoot}/${clean}.md`
}
