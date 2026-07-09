import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  buildHeadTags,
  buildRobots,
  buildSitemap,
  escapeHtml,
  injectHead,
  truncateMeta,
} from './prerender-courses.mjs'

describe('escapeHtml', () => {
  it('escapes script payloads', () => {
    assert.equal(
      escapeHtml(`</title><script>alert(1)</script>`),
      `&lt;/title&gt;&lt;script&gt;alert(1)&lt;/script&gt;`,
    )
  })
  it('escapes quotes and ampersands', () => {
    assert.equal(escapeHtml(`A & "B" <C>`), `A &amp; &quot;B&quot; &lt;C&gt;`)
  })
})

describe('truncateMeta', () => {
  it('leaves short text alone', () => {
    assert.equal(truncateMeta('Hello world'), 'Hello world')
  })
  it('truncates long text on a word boundary', () => {
    const long = 'word '.repeat(50)
    const out = truncateMeta(long, 40)
    assert.ok(out.length <= 41)
    assert.ok(out.endsWith('…'))
  })
})

describe('buildHeadTags', () => {
  it('includes title, description, canonical, OG, Twitter, and JSON-LD', () => {
    const html = buildHeadTags({
      title: 'Intro <Python> — Lextures',
      description: 'Learn "Python" & more',
      canonical: 'https://lextures.com/courses/intro-python',
      image: 'https://cdn.example/hero.jpg',
      jsonLd: { '@type': 'Course', name: 'Intro' },
    })
    assert.match(html, /<title>Intro &lt;Python&gt; — Lextures<\/title>/)
    assert.match(html, /content="Learn &quot;Python&quot; &amp; more"/)
    assert.match(html, /rel="canonical" href="https:\/\/lextures.com\/courses\/intro-python"/)
    assert.match(html, /og:image" content="https:\/\/cdn.example\/hero.jpg"/)
    assert.match(html, /twitter:card" content="summary_large_image"/)
    assert.match(html, /application\/ld\+json/)
    assert.match(html, /"@type":"Course"/)
    assert.doesNotMatch(html, /<script>alert/)
  })
})

describe('injectHead', () => {
  it('replaces title and injects meta into shell', () => {
    const shell = `<!doctype html><html><head>
    <meta name="description" content="old" />
    <meta property="og:title" content="Old" />
    <title>Old Title</title>
  </head><body></body></html>`
    const tags = buildHeadTags({
      title: 'New Title',
      description: 'New desc',
      canonical: 'https://lextures.com/courses/x',
    })
    const out = injectHead(shell, tags)
    assert.match(out, /<title>New Title<\/title>/)
    assert.match(out, /content="New desc"/)
    assert.match(out, /rel="canonical"/)
    assert.doesNotMatch(out, /Old Title/)
  })
})

describe('buildSitemap', () => {
  it('lists static routes and course URLs', () => {
    const xml = buildSitemap([
      { slug: 'intro-python', lastmod: '2026-01-15T00:00:00Z' },
      { slug: 'evil<script>', lastmod: '2026-01-16' },
    ])
    assert.match(xml, /https:\/\/lextures.com\/courses/)
    assert.match(xml, /https:\/\/lextures.com\/courses\/intro-python/)
    assert.match(xml, /<lastmod>2026-01-15<\/lastmod>/)
    assert.match(xml, /https:\/\/lextures.com\/pricing/)
    // slug is path-encoded
    assert.match(xml, /evil%3Cscript%3E/)
  })
})

describe('buildRobots', () => {
  it('allows courses and references sitemap', () => {
    const txt = buildRobots()
    assert.match(txt, /Allow: \/courses/)
    assert.match(txt, /Sitemap: https:\/\/lextures.com\/sitemap.xml/)
  })
})
