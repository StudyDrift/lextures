/** Pure document-head helpers shared by runtime hook and static generator (SEO.1). */

export type JsonLdNode = Record<string, unknown>

export type DocumentHeadOptions = {
  title: string
  description: string
  canonical: string
  image?: string
  imageAlt?: string
  /** Single node (legacy) or multi-node graph. */
  jsonLd?: JsonLdNode | JsonLdNode[] | null
  robots?: 'index,follow' | 'noindex,follow' | string
  /**
   * Absolute URL of the plain-text markdown sibling (SEO.2 FR-15).
   * Emitted as `<link rel="alternate" type="text/markdown" href="…">`.
   */
  markdownAlternate?: string | null
  alternates?: Array<{ hreflang: string; href: string }>
  locale?: string
  dir?: 'ltr' | 'rtl'
}

export const DEFAULT_OG_IMAGE = 'https://lextures.com/assets/og-default.png'
export const JSON_LD_SCRIPT_ID = 'site-json-ld'
/** @deprecated Prefer JSON_LD_SCRIPT_ID; kept for existing course prerender ids. */
export const COURSE_JSON_LD_SCRIPT_ID = 'course-json-ld'

function cardSection(pathname: string): string {
  if (pathname.startsWith('/blog/')) return 'Guide'
  if (pathname.startsWith('/docs/')) return 'Help'
  if (pathname.startsWith('/research/')) return 'Research'
  if (pathname.startsWith('/compare') || pathname.startsWith('/vs/')) return 'Comparison'
  if (pathname.startsWith('/courses/')) return 'Course'
  return 'Lextures'
}

/** Browser-safe twin of the build renderer's stable content-address function. */
export function socialCardHash(title: string, pathname: string): string {
  const input = `seo14-v1\0${cardSection(pathname)}\0${title}`
  let a = 0x811c9dc5
  let b = 0x9e3779b9
  for (let i = 0; i < input.length; i++) {
    const code = input.charCodeAt(i)
    a = Math.imul(a ^ code, 0x01000193) >>> 0
    b = Math.imul(b ^ code, 0x85ebca6b) >>> 0
  }
  return a.toString(16).padStart(8, '0') + b.toString(16).padStart(8, '0')
}

export function socialCardUrl(title: string, canonical: string): string {
  const url = new URL(canonical)
  return `${url.origin}/og/${socialCardHash(title, url.pathname)}.png`
}

export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/**
 * Serialize JSON-LD for embedding in a `<script type="application/ld+json">`.
 * Always emits a single `@graph` envelope (SEO.3 FR-1). Escapes `<`, `>`, `&`
 * and the `</script` sequence so user content cannot break out (FR-3).
 */
export function serializeJsonLd(nodes: JsonLdNode | JsonLdNode[]): string {
  let graphNodes: JsonLdNode[]
  if (Array.isArray(nodes)) {
    // If caller already passed a single envelope with @graph, unwrap it
    if (nodes.length === 1 && Array.isArray(nodes[0]['@graph'])) {
      graphNodes = nodes[0]['@graph'] as JsonLdNode[]
    } else {
      graphNodes = nodes
    }
  } else if (nodes && Array.isArray(nodes['@graph'])) {
    graphNodes = nodes['@graph'] as JsonLdNode[]
  } else {
    graphNodes = nodes ? [nodes] : []
  }

  // Strip nested @context — graph builder owns context
  const cleaned = graphNodes.map(n => {
    if (!n || typeof n !== 'object') return n
    const { ['@context']: _c, ...rest } = n as JsonLdNode & { '@context'?: unknown }
    return rest as JsonLdNode
  })

  const payload = {
    '@context': 'https://schema.org',
    '@graph': cleaned,
  }
  return JSON.stringify(payload)
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026')
    .replace(/\u2028/g, '\\u2028')
    .replace(/\u2029/g, '\\u2029')
}

export function truncateMetaDescription(text: string, maxLen = 160): string {
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

/** Soft cap for SEO titles (AC-3). Keeps uniqueness of the prefix when possible. */
export function truncateTitle(text: string, maxLen = 60): string {
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 20 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

function upsertMeta(attr: 'name' | 'property', key: string, content: string): void {
  let el = document.head.querySelector(`meta[${attr}="${key}"]`) as HTMLMetaElement | null
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, key)
    document.head.appendChild(el)
  }
  el.content = content
}

function upsertLink(rel: string, href: string): void {
  let el = document.head.querySelector(`link[rel="${rel}"]`) as HTMLLinkElement | null
  if (!el) {
    el = document.createElement('link')
    el.rel = rel
    document.head.appendChild(el)
  }
  el.href = href
}

function normalizeJsonLd(
  jsonLd: DocumentHeadOptions['jsonLd'],
): JsonLdNode | JsonLdNode[] | null {
  if (!jsonLd) return null
  if (Array.isArray(jsonLd)) return jsonLd.length ? jsonLd : null
  return jsonLd
}

export function applyDocumentHead(opts: DocumentHeadOptions): void {
  const title = opts.title
  const description = opts.description
  const image = opts.image || socialCardUrl(opts.title, opts.canonical)
  const imageAlt = opts.imageAlt || `${opts.title} — Lextures`
  const robots = opts.robots || 'index,follow'

  document.title = title
  upsertMeta('name', 'description', description)
  upsertMeta('name', 'robots', robots)
  upsertLink('canonical', opts.canonical)
  document.documentElement.lang = opts.locale || 'en'
  document.documentElement.dir = opts.dir || 'ltr'

  document.head.querySelectorAll('link[rel="alternate"][hreflang]').forEach(node => node.remove())
  for (const alternate of opts.alternates ?? []) {
    const link = document.createElement('link')
    link.rel = 'alternate'
    link.hreflang = alternate.hreflang
    link.href = alternate.href
    document.head.appendChild(link)
  }

  // Markdown alternate (SEO.2) — remove stale link when navigating away
  const existingMd = document.head.querySelector(
    'link[rel="alternate"][type="text/markdown"]',
  ) as HTMLLinkElement | null
  if (opts.markdownAlternate) {
    let el = existingMd
    if (!el) {
      el = document.createElement('link')
      el.rel = 'alternate'
      el.type = 'text/markdown'
      document.head.appendChild(el)
    }
    el.href = opts.markdownAlternate
  } else if (existingMd) {
    existingMd.remove()
  }
  upsertMeta('property', 'og:title', title)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:image', image)
  upsertMeta('property', 'og:image:width', '1200')
  upsertMeta('property', 'og:image:height', '630')
  upsertMeta('property', 'og:image:alt', imageAlt)
  upsertMeta('property', 'og:type', 'website')
  upsertMeta('property', 'og:url', opts.canonical)

  upsertMeta('name', 'twitter:card', 'summary_large_image')
  upsertMeta('name', 'twitter:title', title)
  upsertMeta('name', 'twitter:description', description)
  upsertMeta('name', 'twitter:image', image)
  upsertMeta('name', 'twitter:image:alt', imageAlt)

  const ld = normalizeJsonLd(opts.jsonLd)
  // Remove legacy course-only id if present
  document.getElementById(COURSE_JSON_LD_SCRIPT_ID)?.remove()
  if (ld) {
    let el = document.getElementById(JSON_LD_SCRIPT_ID) as HTMLScriptElement | null
    if (!el) {
      el = document.createElement('script')
      el.id = JSON_LD_SCRIPT_ID
      el.type = 'application/ld+json'
      document.head.appendChild(el)
    }
    el.textContent = serializeJsonLd(ld)
  } else {
    clearJsonLd()
  }
}

export function clearJsonLd(): void {
  document.getElementById(JSON_LD_SCRIPT_ID)?.remove()
  document.getElementById(COURSE_JSON_LD_SCRIPT_ID)?.remove()
}

/** Build the HTML fragment injected into prerendered pages. */
export function buildPrerenderHeadTags(opts: DocumentHeadOptions): string {
  const title = escapeHtml(opts.title)
  const description = escapeHtml(opts.description)
  const canonical = escapeHtml(opts.canonical)
  const image = escapeHtml(opts.image || DEFAULT_OG_IMAGE)
  const imageAlt = escapeHtml(opts.imageAlt || `${opts.title} — Lextures`)
  const robots = escapeHtml(opts.robots || 'index,follow')
  const lines = [
    `<title>${title}</title>`,
    `<meta name="description" content="${description}" />`,
    `<meta name="robots" content="${robots}" />`,
    `<link rel="canonical" href="${canonical}" />`,
    `<meta property="og:title" content="${title}" />`,
    `<meta property="og:description" content="${description}" />`,
    `<meta property="og:image" content="${image}" />`,
    `<meta property="og:image:width" content="1200" />`,
    `<meta property="og:image:height" content="630" />`,
    `<meta property="og:image:alt" content="${imageAlt}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:url" content="${canonical}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${title}" />`,
    `<meta name="twitter:description" content="${description}" />`,
    `<meta name="twitter:image" content="${image}" />`,
    `<meta name="twitter:image:alt" content="${imageAlt}" />`,
  ]
  if (opts.markdownAlternate) {
    lines.push(
      `<link rel="alternate" type="text/markdown" href="${escapeHtml(opts.markdownAlternate)}" />`,
    )
  }
  for (const alternate of opts.alternates ?? []) {
    lines.push(`<link rel="alternate" hreflang="${escapeHtml(alternate.hreflang)}" href="${escapeHtml(alternate.href)}" />`)
  }
  const ld = normalizeJsonLd(opts.jsonLd)
  if (ld) {
    lines.push(
      `<script type="application/ld+json" id="${JSON_LD_SCRIPT_ID}">${serializeJsonLd(ld)}</script>`,
    )
  }
  return lines.join('\n    ')
}
