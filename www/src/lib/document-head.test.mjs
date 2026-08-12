/**
 * Unit tests for document-head pure helpers (SEO.1).
 * Mirrors the TypeScript module logic for Node's test runner.
 */
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

function escapeHtml(value) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function serializeJsonLd(nodes) {
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

function truncateMetaDescription(text, maxLen = 160) {
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}

function buildPrerenderHeadTags(opts) {
  const title = escapeHtml(opts.title)
  const description = escapeHtml(opts.description)
  const canonical = escapeHtml(opts.canonical)
  const image = escapeHtml(opts.image || 'https://lextures.com/assets/lextures-mark.svg')
  const robots = escapeHtml(opts.robots || 'index,follow')
  const lines = [
    `<title>${title}</title>`,
    `<meta name="description" content="${description}" />`,
    `<meta name="robots" content="${robots}" />`,
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
      `<script type="application/ld+json" id="site-json-ld">${serializeJsonLd(opts.jsonLd)}</script>`,
    )
  }
  return lines.join('\n    ')
}

describe('document-head helpers', () => {
  it('escapes HTML in meta builders', () => {
    const html = buildPrerenderHeadTags({
      title: '<script>x</script>',
      description: 'a & b',
      canonical: 'https://lextures.com/courses/s',
      jsonLd: { '@type': 'Course', name: 'Safe' },
    })
    assert.match(html, /&lt;script&gt;/)
    assert.doesNotMatch(html, /<script>x<\/script>/)
    assert.match(html, /a &amp; b/)
    assert.match(html, /"@type":"Course"/)
    assert.match(html, /name="robots"/)
  })

  it('truncates descriptions', () => {
    assert.equal(truncateMetaDescription('short'), 'short')
    const long = 'alpha '.repeat(40)
    const out = truncateMetaDescription(long, 50)
    assert.ok(out.endsWith('…'))
    assert.ok(out.length <= 51)
  })

  it('serializes multi-node JSON-LD as @graph and escapes script close', () => {
    const out = serializeJsonLd([
      { '@type': 'WebPage', name: 'Pricing', '@id': 'https://lextures.com/pricing#webpage' },
      { '@type': 'BreadcrumbList', name: '</script><img onerror=1>', '@id': 'https://lextures.com/pricing#breadcrumb' },
    ])
    assert.match(out, /"@context":"https:\/\/schema.org"/)
    assert.match(out, /"@graph"/)
    assert.match(out, /WebPage/)
    assert.match(out, /BreadcrumbList/)
    assert.match(out, /\\u003c/)
    assert.doesNotMatch(out, /<\/script>/i)
  })

  it('escapes hostile course title that tries to break out of script (AC-7)', () => {
    const hostile = '</script><img src=x onerror=alert(1)>'
    const out = serializeJsonLd([
      {
        '@type': 'Course',
        '@id': 'https://lextures.com/courses/evil#course',
        name: hostile,
      },
    ])
    assert.doesNotMatch(out, /<\/script>/i)
    assert.match(out, /\\u003c\/script/)
    const html = buildPrerenderHeadTags({
      title: 'Course',
      description: 'd',
      canonical: 'https://lextures.com/courses/evil',
      jsonLd: [{ '@type': 'Course', name: hostile, '@id': 'https://lextures.com/courses/evil#course' }],
    })
    // Outer script tag still closed properly once; payload cannot inject tags
    assert.equal((html.match(/<script type="application\/ld\+json"/g) || []).length, 1)
    // Angle brackets are unicode-escaped so the browser never sees a real </script> breakout
    assert.doesNotMatch(html, /<\/script><img/i)
    assert.match(html, /\\u003cimg/)
  })
})
