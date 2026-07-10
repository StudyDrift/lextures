/** Pure document-head helpers shared by runtime hook and prerender (plan MKT10). */

export type DocumentHeadOptions = {
  title: string
  description: string
  canonical: string
  image?: string
  jsonLd?: Record<string, unknown> | null
}

export const DEFAULT_OG_IMAGE = 'https://lextures.com/assets/lextures-mark.svg'
export const JSON_LD_SCRIPT_ID = 'course-json-ld'

export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function truncateMetaDescription(text: string, maxLen = 160): string {
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
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

export function applyDocumentHead(opts: DocumentHeadOptions): void {
  const title = opts.title
  const description = opts.description
  const image = opts.image || DEFAULT_OG_IMAGE

  document.title = title
  upsertMeta('name', 'description', description)
  upsertLink('canonical', opts.canonical)

  upsertMeta('property', 'og:title', title)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:image', image)
  upsertMeta('property', 'og:type', 'website')
  upsertMeta('property', 'og:url', opts.canonical)

  upsertMeta('name', 'twitter:card', 'summary_large_image')
  upsertMeta('name', 'twitter:title', title)
  upsertMeta('name', 'twitter:description', description)
  upsertMeta('name', 'twitter:image', image)

  if (opts.jsonLd) {
    let el = document.getElementById(JSON_LD_SCRIPT_ID) as HTMLScriptElement | null
    if (!el) {
      el = document.createElement('script')
      el.id = JSON_LD_SCRIPT_ID
      el.type = 'application/ld+json'
      document.head.appendChild(el)
    }
    el.textContent = JSON.stringify(opts.jsonLd)
  } else {
    clearJsonLd()
  }
}

export function clearJsonLd(): void {
  document.getElementById(JSON_LD_SCRIPT_ID)?.remove()
}

/** Build the HTML fragment injected into prerendered course pages. */
export function buildPrerenderHeadTags(opts: DocumentHeadOptions): string {
  const title = escapeHtml(opts.title)
  const description = escapeHtml(opts.description)
  const canonical = escapeHtml(opts.canonical)
  const image = escapeHtml(opts.image || DEFAULT_OG_IMAGE)
  const lines = [
    `<title>${title}</title>`,
    `<meta name="description" content="${description}" />`,
    `<link rel="canonical" href="${canonical}" />`,
    `<meta property="og:title" content="${title}" />`,
    `<meta property="og:description" content="${description}" />`,
    `<meta property="og:image" content="${image}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:url" content="${canonical}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${title}" />`,
    `<meta name="twitter:description" content="${description}" />`,
    `<meta name="twitter:image" content="${image}" />`,
  ]
  if (opts.jsonLd) {
    lines.push(
      `<script type="application/ld+json" id="${JSON_LD_SCRIPT_ID}">${JSON.stringify(opts.jsonLd)}</script>`,
    )
  }
  return lines.join('\n    ')
}
