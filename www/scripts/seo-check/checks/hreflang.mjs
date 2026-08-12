const normalizePath = value => {
  try { return new URL(value, 'https://lextures.com').pathname.replace(/\/$/, '') || '/' } catch { return value }
}

const parseAttributes = tag => Object.fromEntries(
  [...tag.matchAll(/([\w:-]+)=["']([^"']*)["']/g)].map(match => [match[1].toLowerCase(), match[2]]),
)

export function htmlAlternates(html) {
  return [...String(html).matchAll(/<link\b[^>]*rel=["']alternate["'][^>]*hreflang=["'][^"']+["'][^>]*>/gi)]
    .map(match => parseAttributes(match[0]))
    .filter(attrs => attrs.hreflang && attrs.href)
    .map(attrs => ({ hreflang: attrs.hreflang, href: attrs.href }))
}

export function sitemapAlternates(xmlDocuments) {
  const byPath = new Map()
  for (const xml of xmlDocuments) {
    for (const block of String(xml).matchAll(/<url>([\s\S]*?)<\/url>/gi)) {
      const loc = block[1].match(/<loc>([^<]+)<\/loc>/i)?.[1]
      if (!loc) continue
      const alternates = [...block[1].matchAll(/<xhtml:link\b[^>]*>/gi)]
        .map(match => parseAttributes(match[0]))
        .filter(attrs => attrs.hreflang && attrs.href)
        .map(attrs => ({ hreflang: attrs.hreflang, href: attrs.href }))
      byPath.set(normalizePath(loc), alternates)
    }
  }
  return byPath
}

const signature = alternates => alternates.map(alt => `${alt.hreflang.toLowerCase()}=${normalizePath(alt.href)}`).sort().join('|')

export function validateHreflang({ manifest, htmlByPath, sitemapXmls }) {
  const findings = [], pages = new Map((manifest.urls || []).map(page => [page.path, page]))
  const sitemapByPath = sitemapAlternates(sitemapXmls), clusters = new Map()
  for (const [pagePath, page] of pages) {
    const html = htmlAlternates(htmlByPath.get(pagePath) || ''), sitemap = sitemapByPath.get(pagePath) || []
    const localized = page.locale && page.locale !== 'en'
    if (!html.length && !localized) continue
    if (signature(html) !== signature(sitemap)) findings.push({ page: pagePath, message: 'HTML and sitemap hreflang annotations do not match.', line: null })
    const locale = String(page.locale || 'en')
    if (!html.some(alt => alt.hreflang.toLowerCase() === locale.toLowerCase() && normalizePath(alt.href) === pagePath)) findings.push({ page: pagePath, message: `Missing self-referencing hreflang ${locale}.`, line: null })
    const xDefault = html.find(alt => alt.hreflang.toLowerCase() === 'x-default')
    if (!xDefault) findings.push({ page: pagePath, message: 'Missing x-default hreflang.', line: null })
    else if (normalizePath(xDefault.href) !== (page.translationOf || pagePath)) findings.push({ page: pagePath, message: 'x-default must point to the English root URL.', line: null })
    clusters.set(pagePath, html)
    for (const alternate of html) {
      if (alternate.hreflang.toLowerCase() === 'x-default') continue
      const targetPath = normalizePath(alternate.href), target = pages.get(targetPath)
      if (!target || String(target.robots).includes('noindex') || !target.sitemap) findings.push({ page: pagePath, message: `hreflang ${alternate.hreflang} targets a missing or non-indexable URL: ${targetPath}.`, line: null })
    }
  }
  for (const [pagePath, alternates] of clusters) for (const alternate of alternates) {
    if (alternate.hreflang.toLowerCase() === 'x-default') continue
    const targetPath = normalizePath(alternate.href)
    if (targetPath !== pagePath && !(clusters.get(targetPath) || []).some(back => back.hreflang.toLowerCase() !== 'x-default' && normalizePath(back.href) === pagePath)) findings.push({ page: pagePath, message: `Non-reciprocal hreflang pair: ${pagePath} → ${targetPath}.`, line: null })
  }
  return findings
}
