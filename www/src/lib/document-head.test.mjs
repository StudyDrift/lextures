/**
 * Unit tests for document-head pure helpers (plan MKT10).
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
      `<script type="application/ld+json" id="course-json-ld">${JSON.stringify(opts.jsonLd)}</script>`,
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
  })

  it('truncates descriptions', () => {
    assert.equal(truncateMetaDescription('short'), 'short')
    const long = 'alpha '.repeat(40)
    const out = truncateMetaDescription(long, 50)
    assert.ok(out.endsWith('…'))
    assert.ok(out.length <= 51)
  })
})
