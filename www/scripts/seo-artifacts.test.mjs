import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  assertSitemapManifestParity,
  batchIndexNowUrls,
  buildIndexNowBody,
  buildLlmsFullTxt,
  buildSitemapArtifacts,
  buildSitemapIndex,
  buildUrlset,
  diffManifestUrls,
  normalizeLastmod,
  parseFrontmatter,
  resolveLastmod,
  shardEntries,
  shouldEmitMarkdownSibling,
  sitemapSectionForPath,
  SITEMAP_MAX_URLS,
} from './seo-artifacts.mjs'

describe('normalizeLastmod / resolveLastmod', () => {
  it('normalizes ISO datetime to YYYY-MM-DD', () => {
    assert.equal(normalizeLastmod('2026-05-06T12:00:00Z'), '2026-05-06')
    assert.equal(normalizeLastmod('2026-01-15'), '2026-01-15')
  })

  it('returns null for empty / invalid (omit lastmod)', () => {
    assert.equal(normalizeLastmod(null), null)
    assert.equal(normalizeLastmod(''), null)
    assert.equal(normalizeLastmod('not-a-date'), null)
  })

  it('prefers frontmatter updated over date over git over course', () => {
    assert.equal(
      resolveLastmod({
        frontmatterUpdated: '2026-06-01',
        frontmatterDate: '2026-01-01',
        gitDate: '2025-01-01',
        courseCreatedAt: '2024-01-01',
      }),
      '2026-06-01',
    )
    assert.equal(
      resolveLastmod({
        frontmatterDate: '2026-01-01',
        gitDate: '2025-06-01',
      }),
      '2026-01-01',
    )
    assert.equal(
      resolveLastmod({
        gitDate: '2025-06-01',
        courseUpdatedAt: '2026-03-01',
      }),
      '2025-06-01',
    )
    assert.equal(
      resolveLastmod({
        courseUpdatedAt: '2026-03-01T00:00:00Z',
        courseCreatedAt: '2020-01-01',
      }),
      '2026-03-01',
    )
    assert.equal(resolveLastmod({}), null)
  })
})

describe('sitemap sections and sharding', () => {
  it('assigns sections by path prefix', () => {
    assert.equal(sitemapSectionForPath('/'), 'pages')
    assert.equal(sitemapSectionForPath('/pricing'), 'pages')
    assert.equal(sitemapSectionForPath('/blog'), 'blog')
    assert.equal(sitemapSectionForPath('/blog/x'), 'blog')
    assert.equal(sitemapSectionForPath('/docs/self-hosting'), 'docs')
    assert.equal(sitemapSectionForPath('/courses'), 'courses')
    assert.equal(sitemapSectionForPath('/courses/slug'), 'courses')
    assert.equal(sitemapSectionForPath('/compare/foo'), 'compare')
  })

  it('shards at 50k', () => {
    const items = Array.from({ length: SITEMAP_MAX_URLS + 3 }, (_, i) => i)
    const shards = shardEntries(items)
    assert.equal(shards.length, 2)
    assert.equal(shards[0].length, SITEMAP_MAX_URLS)
    assert.equal(shards[1].length, 3)
  })

  it('builds sitemap index with only non-empty sections', () => {
    const { indexXml, files } = buildSitemapArtifacts(
      [
        { path: '/', lastmod: '2026-01-01', priority: '1.0' },
        { path: '/blog/a', lastmod: '2026-05-06', priority: '0.6' },
        { path: '/courses/x', lastmod: null, priority: '0.8' },
      ],
      'https://lextures.com',
    )
    assert.match(indexXml, /<sitemapindex/)
    assert.match(indexXml, /sitemaps\/pages\.xml/)
    assert.match(indexXml, /sitemaps\/blog\.xml/)
    assert.match(indexXml, /sitemaps\/courses\.xml/)
    assert.doesNotMatch(indexXml, /glossary/)
    assert.equal(files.length, 3)

    const blog = files.find(f => f.section === 'blog')
    assert.match(blog.xml, /<lastmod>2026-05-06<\/lastmod>/)
    assert.match(blog.xml, /https:\/\/lextures.com\/blog\/a/)

    const courses = files.find(f => f.section === 'courses')
    // null lastmod omitted
    assert.doesNotMatch(courses.xml, /<lastmod>/)
  })

  it('omits lastmod when unknown in urlset', () => {
    const xml = buildUrlset([{ path: '/pricing' }], 'https://lextures.com')
    assert.match(xml, /<loc>https:\/\/lextures.com\/pricing<\/loc>/)
    assert.doesNotMatch(xml, /<lastmod>/)
  })

  it('accepts hreflang alternates when provided (SEO.17 plumbing)', () => {
    const xml = buildUrlset(
      [
        {
          path: '/pricing',
          alternates: [{ hreflang: 'en', href: 'https://lextures.com/pricing' }],
        },
      ],
      'https://lextures.com',
    )
    assert.match(xml, /xmlns:xhtml/)
    assert.match(xml, /hreflang="en"/)
  })

  it('shards courses into courses-1.xml when over limit', () => {
    const entries = Array.from({ length: 3 }, (_, i) => ({
      path: `/courses/c-${i}`,
      section: 'courses',
    }))
    const { files, indexXml } = buildSitemapArtifacts(entries, 'https://lextures.com')
    // under limit → single courses.xml
    assert.equal(files.length, 1)
    assert.match(indexXml, /courses\.xml/)

    const many = Array.from({ length: SITEMAP_MAX_URLS + 1 }, (_, i) => ({
      path: `/courses/c-${i}`,
      section: 'courses',
    }))
    const sharded = buildSitemapArtifacts(many, 'https://lextures.com')
    assert.ok(sharded.files.some(f => f.relativePath === 'sitemaps/courses-1.xml'))
    assert.ok(sharded.files.some(f => f.relativePath === 'sitemaps/courses-2.xml'))
  })
})

describe('assertSitemapManifestParity', () => {
  it('passes when sets match', () => {
    const errors = assertSitemapManifestParity(
      ['/', '/pricing'],
      [
        { path: '/', robots: 'index,follow', sitemap: true },
        { path: '/pricing', robots: 'index,follow', sitemap: true },
        { path: '/privacy/history', robots: 'noindex,follow', sitemap: true },
      ],
    )
    assert.deepEqual(errors, [])
  })

  it('fails when sitemap outruns manifest (AC-8)', () => {
    const errors = assertSitemapManifestParity(
      ['/', '/ghost'],
      [{ path: '/', robots: 'index,follow', sitemap: true }],
    )
    assert.ok(errors.some(e => e.includes('/ghost')))
  })

  it('fails when indexable manifest URL missing from sitemap', () => {
    const errors = assertSitemapManifestParity(
      ['/'],
      [
        { path: '/', robots: 'index,follow', sitemap: true },
        { path: '/pricing', robots: 'index,follow', sitemap: true },
      ],
    )
    assert.ok(errors.some(e => e.includes('/pricing')))
  })

  it('fails when noindex URL is in sitemap', () => {
    const errors = assertSitemapManifestParity(
      ['/', '/404'],
      [
        { path: '/', robots: 'index,follow', sitemap: true },
        { path: '/404', robots: 'noindex,follow', sitemap: false },
      ],
    )
    assert.ok(errors.some(e => e.includes('noindex')))
  })
})

describe('llms-full', () => {
  it('concatenates bodies and excludes noindex / legal history', () => {
    const out = buildLlmsFullTxt(
      [
        {
          path: '/docs/self-hosting',
          title: 'Self-hosting',
          body: 'Install with Docker.',
          sourceRoot: 'docs',
        },
        {
          path: '/privacy/history',
          title: 'History',
          body: 'Secret legal',
          sourceRoot: 'legal',
        },
        {
          path: '/hidden',
          title: 'Hidden',
          body: 'nope',
          robots: 'noindex,follow',
          sourceRoot: 'docs',
        },
      ],
      'https://lextures.com',
    )
    assert.match(out, /Self-hosting/)
    assert.match(out, /Install with Docker/)
    assert.doesNotMatch(out, /Secret legal/)
    assert.doesNotMatch(out, /nope/)
    assert.match(out, /llms\.txt/)
  })

  it('truncates at maxBytes with notice', () => {
    const out = buildLlmsFullTxt(
      [
        { path: '/a', title: 'A', body: 'x'.repeat(200), sourceRoot: 'docs' },
        { path: '/b', title: 'B', body: 'y'.repeat(200), sourceRoot: 'docs' },
      ],
      'https://lextures.com',
      { maxBytes: 350 },
    )
    assert.match(out, /TRUNCATED/)
  })
})

describe('IndexNow batching and diff', () => {
  it('batches at 10k', () => {
    const urls = Array.from({ length: 10_001 }, (_, i) => `https://lextures.com/p/${i}`)
    const batches = batchIndexNowUrls(urls)
    assert.equal(batches.length, 2)
    assert.equal(batches[0].length, 10_000)
    assert.equal(batches[1].length, 1)
  })

  it('diffs manifests for new and changed paths', () => {
    const prev = {
      urls: [
        { path: '/', title: 'Home', canonical: 'https://lextures.com/' },
        { path: '/pricing', title: 'Old', canonical: 'https://lextures.com/pricing' },
      ],
    }
    const next = {
      origin: 'https://lextures.com',
      urls: [
        { path: '/', title: 'Home', canonical: 'https://lextures.com/' },
        { path: '/pricing', title: 'New', canonical: 'https://lextures.com/pricing' },
        { path: '/blog/x', title: 'Post', canonical: 'https://lextures.com/blog/x' },
      ],
    }
    const changed = diffManifestUrls(prev, next)
    assert.ok(changed.includes('https://lextures.com/pricing'))
    assert.ok(changed.includes('https://lextures.com/blog/x'))
    assert.ok(!changed.includes('https://lextures.com/'))
  })

  it('builds IndexNow body', () => {
    const body = buildIndexNowBody({
      host: 'lextures.com',
      key: 'abc',
      keyLocation: 'https://lextures.com/abc.txt',
      urlList: ['https://lextures.com/'],
    })
    assert.equal(body.host, 'lextures.com')
    assert.deepEqual(body.urlList, ['https://lextures.com/'])
  })
})

describe('frontmatter + markdown siblings', () => {
  it('parses frontmatter updated and date', () => {
    const { meta, body } = parseFrontmatter(
      '---\ntitle: Hi\ndate: "2026-05-06"\nupdated: 2026-06-01\n---\n\nBody here\n',
    )
    assert.equal(meta.date, '2026-05-06')
    assert.equal(meta.updated, '2026-06-01')
    assert.match(body, /Body here/)
  })

  it('shouldEmitMarkdownSibling only for content detail paths', () => {
    assert.equal(shouldEmitMarkdownSibling('/docs/self-hosting'), false)
    assert.equal(shouldEmitMarkdownSibling('/docs/self-hosting/install'), true)
    assert.equal(shouldEmitMarkdownSibling('/es/docs/cursos/encontrar-tu-curso'), true)
    assert.equal(shouldEmitMarkdownSibling('/es/blog/hola'), true)
    assert.equal(shouldEmitMarkdownSibling('/blog/x'), true)
    assert.equal(shouldEmitMarkdownSibling('/docs'), false)
    assert.equal(shouldEmitMarkdownSibling('/pricing'), false)
  })
})

describe('buildSitemapIndex', () => {
  it('emits sitemapindex entries', () => {
    const xml = buildSitemapIndex([
      { loc: 'https://lextures.com/sitemaps/pages.xml', lastmod: '2026-01-01' },
    ])
    assert.match(xml, /<sitemapindex/)
    assert.match(xml, /sitemaps\/pages\.xml/)
    assert.match(xml, /<lastmod>2026-01-01<\/lastmod>/)
  })
})
