import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  buildHeadTags,
  buildRobots,
  buildLegacyAudienceRedirectHtml,
  buildRedirectStubHtml,
  buildRedirectsFile,
  buildSitemap,
  escapeHtml,
  injectDocument,
  injectHead,
  parseMarkdownDate,
  resolveApiAssetUrl,
  serializeJsonLd,
  STATIC_ROUTES,
  truncateMeta,
  truncateTitle,
  normalizeFallbackContentHtml,
  articleSummaryFromHtml,
  categoriesFromArticleSummaries,
  validateGeneratedPage,
  outputPathForRoute,
  buildLinkGraph,
  validateLinkGraph,
} from './generate-site.mjs'

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

describe('articleSummaryFromHtml', () => {
  it('reconstructs blog and docs listing metadata from deployed HTML', () => {
    const blog = articleSummaryFromHtml(
      '/blog/the-synthetic-renaissance/',
      '<title>The Synthetic Renaissance: How AI is Reshaping the… — Lextures</title><meta name="description" content="Understand how accountable AI can support feedback." />',
    )
    assert.deepEqual(blog, {
      path: '/blog/the-synthetic-renaissance',
      kind: 'blog',
      slug: 'the-synthetic-renaissance',
      locale: 'en',
      categorySlug: undefined,
      title: 'The Synthetic Renaissance: How AI is Reshaping the…',
      description: 'Understand how accountable AI can support feedback.',
      publishedAt: null,
      bodyMd: '',
    })
    const doc = articleSummaryFromHtml(
      '/docs/integrations/using-lextures-with-make',
      '<title>Using Lextures with Make — Lextures</title><meta name="description" content="Build a Make scenario." />',
    )
    assert.equal(doc.kind, 'doc')
    assert.equal(doc.categorySlug, 'integrations')
    assert.deepEqual(
      categoriesFromArticleSummaries([doc]),
      [{ slug: 'integrations', locale: 'en', title: 'Integrations', description: '', sortOrder: 0 }],
    )
  })
})

describe('normalizeFallbackContentHtml', () => {
  it('truncates overlong titles and preserves og:image', () => {
    const longTitle = 'How Adaptive AI Is Changing What Personalized Learning Means — Lextures'
    assert.ok(longTitle.length > 60)
    const { html, title, image } = normalizeFallbackContentHtml(`<!doctype html><html><head>
      <title>${longTitle}</title>
      <meta property="og:title" content="${longTitle}" />
      <meta property="og:image" content="https://lextures.com/og/1d7eaa8ac29fabee.png" />
    </head><body>ok</body></html>`)
    assert.equal(title, truncateTitle(longTitle))
    assert.ok(title.length <= 60)
    assert.match(html, new RegExp(`<title>${title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}<\\/title>`))
    assert.equal(image, 'https://lextures.com/og/1d7eaa8ac29fabee.png')
  })
})

describe('serializeJsonLd', () => {
  it('escapes script breakouts in JSON-LD', () => {
    const out = serializeJsonLd({
      '@type': 'Course',
      name: '</script><img onerror=alert(1)>',
    })
    assert.match(out, /\\u003c/)
    assert.doesNotMatch(out, /<\/script>/i)
  })
})

describe('buildHeadTags', () => {
  it('includes title, description, canonical, robots, OG, Twitter, and JSON-LD', () => {
    const html = buildHeadTags({
      title: 'Intro <Python> — Lextures',
      description: 'Learn "Python" & more',
      canonical: 'https://lextures.com/courses/intro-python',
      image: 'https://cdn.example/hero.jpg',
      robots: 'index,follow',
      jsonLd: { '@type': 'Course', name: 'Intro' },
    })
    assert.match(html, /<title>Intro &lt;Python&gt; — Lextures<\/title>/)
    assert.match(html, /content="Learn &quot;Python&quot; &amp; more"/)
    assert.match(html, /rel="canonical" href="https:\/\/lextures.com\/courses\/intro-python"/)
    assert.match(html, /name="robots" content="index,follow"/)
    assert.match(html, /og:image" content="https:\/\/cdn.example\/hero.jpg"/)
    assert.match(html, /twitter:card" content="summary_large_image"/)
    assert.match(html, /application\/ld\+json/)
    assert.match(html, /"@type":"Course"/)
    assert.doesNotMatch(html, /<script>alert/)
  })
})

describe('injectDocument', () => {
  it('replaces title, injects meta, and fills #root', () => {
    const shell = `<!doctype html><html><head>
    <meta name="description" content="old" />
    <meta property="og:title" content="Old" />
    <title>Old Title</title>
  </head><body><div id="root"></div>
  <script type="module" src="/assets/main-abc.js"></script>
  </body></html>`
    const tags = buildHeadTags({
      title: 'New Title',
      description: 'New desc',
      canonical: 'https://lextures.com/pricing',
    })
    const out = injectDocument(shell, {
      headTags: tags,
      bodyHtml: '<h1>Pricing</h1>',
      ssrData: { path: '/pricing' },
      interactive: true,
    })
    assert.match(out, /<title>New Title<\/title>/)
    assert.match(out, /content="New desc"/)
    assert.match(out, /rel="canonical"/)
    assert.match(out, /<div id="root"><h1>Pricing<\/h1><\/div>/)
    assert.match(out, /window\.__LEXTURES_SSR__/)
    assert.match(out, /data-interactive="true"/)
    assert.doesNotMatch(out, /Old Title/)
  })

  it('strips React entry and uses static-island for interactive:false', () => {
    const shell = `<!doctype html><html lang="en"><head>
    <title>Old</title>
  </head><body><div id="root"></div>
  <script type="module" src="/assets/main-abc.js"></script>
  </body></html>`
    const tags = buildHeadTags({
      title: 'Privacy Policy — Lextures',
      description: 'Privacy',
      canonical: 'https://lextures.com/privacy',
    })
    const out = injectDocument(shell, {
      headTags: tags,
      bodyHtml: '<h1>Privacy</h1>',
      ssrData: { path: '/privacy' },
      interactive: false,
      staticIslandSrc: '/assets/static-island-xyz.js',
    })
    assert.match(out, /data-interactive="false"/)
    assert.match(out, /static-island-xyz\.js/)
    assert.doesNotMatch(out, /main-abc\.js/)
    assert.doesNotMatch(out, /window\.__LEXTURES_SSR__/)
    assert.doesNotMatch(out, /fonts\.googleapis/)
  })

  it('inserts meta name=description when the shell has none', () => {
    const shell = `<!doctype html><html><head>
    <title>Old Title</title>
  </head><body><div id="root"></div></body></html>`
    const tags = buildHeadTags({
      title: 'Home — Lextures',
      description: 'Shell has no description meta yet.',
      canonical: 'https://lextures.com/',
    })
    const out = injectDocument(shell, {
      headTags: tags,
      bodyHtml: '<h1>Home</h1>',
      ssrData: { path: '/' },
      interactive: true,
    })
    assert.match(out, /<meta name="description" content="Shell has no description meta yet\." \/>/)
    assert.match(out, /<title>Home — Lextures<\/title>/)
  })
})

describe('injectHead (compat)', () => {
  it('replaces title and injects meta into shell', () => {
    const shell = `<!doctype html><html><head>
    <meta name="description" content="old" />
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
  })
})

describe('buildSitemap', () => {
  it('lists concrete paths and encodes path segments', () => {
    const xml = buildSitemap([
      { path: '/', priority: '1.0' },
      { path: '/pricing', priority: '0.8' },
      { path: '/courses/intro-python', lastmod: '2026-01-15T00:00:00Z', priority: '0.8' },
      { path: '/courses/evil<script>', lastmod: '2026-01-16', priority: '0.8' },
    ])
    assert.match(xml, /https:\/\/lextures.com\//)
    assert.match(xml, /https:\/\/lextures.com\/pricing/)
    assert.match(xml, /https:\/\/lextures.com\/courses\/intro-python/)
    assert.match(xml, /<lastmod>2026-01-15<\/lastmod>/)
    assert.match(xml, /evil%3Cscript%3E/)
  })

  it('omits lastmod when unknown (never fabricates build date)', () => {
    const xml = buildSitemap([{ path: '/pricing', priority: '0.8' }])
    assert.match(xml, /<loc>https:\/\/lextures.com\/pricing<\/loc>/)
    assert.doesNotMatch(xml, /<lastmod>/)
  })

  it('includes /homeschool and excludes self-learner', () => {
    const xml = buildSitemap([
      { path: '/homeschool', priority: '0.6' },
      { path: '/pricing', priority: '0.8' },
    ])
    assert.match(xml, /<loc>https:\/\/lextures.com\/homeschool<\/loc>/)
    assert.doesNotMatch(xml, /self-learner/)
  })

  it('STATIC_ROUTES never lists the legacy self-learner path', () => {
    assert.ok(!STATIC_ROUTES.some(r => r.loc.includes('self-learner')))
  })
})

describe('parseMarkdownDate', () => {
  it('reads quoted and bare frontmatter dates', () => {
    assert.equal(parseMarkdownDate('---\ndate: "2026-05-06"\n---\nbody'), '2026-05-06')
    assert.equal(parseMarkdownDate('---\ndate: 2024-05-20\n---\nbody'), '2024-05-20')
  })
  it('returns null without frontmatter date', () => {
    assert.equal(parseMarkdownDate('# No frontmatter'), null)
  })
})

describe('redirect stubs', () => {
  it('emits meta refresh, canonical, and fallback link', () => {
    const html = buildLegacyAudienceRedirectHtml('https://lextures.com')
    assert.match(html, /http-equiv="refresh" content="0; url=\/homeschool"/)
    assert.match(html, /rel="canonical" href="https:\/\/lextures.com\/homeschool"/)
    assert.match(html, /<a href="\/homeschool">/)
  })

  it('escapes the site origin in the canonical href', () => {
    const html = buildRedirectStubHtml('/homeschool', 'https://evil.com/"onclick="alert(1)')
    assert.match(html, /&quot;/)
    assert.doesNotMatch(html, /onclick="alert/)
  })

  it('builds _redirects file', () => {
    const body = buildRedirectsFile([
      { from: '/self-learner', to: '/homeschool', status: 301 },
    ])
    assert.match(body, /\/self-learner \/homeschool 301/)
    assert.match(body, /404\.html\s+404/)
  })
})

describe('buildRobots', () => {
  it('references sitemap index and disallows non-indexable paths', () => {
    const txt = buildRobots()
    assert.match(txt, /Allow: \//)
    assert.match(txt, /Disallow: \/404/)
    assert.match(txt, /Sitemap: https:\/\/lextures.com\/sitemap.xml/)
    // redundant Allow: /courses removed (SEO.2 FR-5)
    assert.doesNotMatch(txt, /Allow: \/courses/)
  })

  it('supports staging disallow-all', () => {
    const txt = buildRobots('https://staging.example', { disallowAll: true })
    assert.match(txt, /Disallow: \//)
  })
})

describe('buildHeadTags markdown alternate', () => {
  it('emits text/markdown alternate when provided', () => {
    const html = buildHeadTags({
      title: 'Doc',
      description: 'Guide',
      canonical: 'https://lextures.com/docs/self-hosting',
      markdownAlternate: 'https://lextures.com/docs/self-hosting.md',
    })
    assert.match(html, /rel="alternate" type="text\/markdown"/)
    assert.match(html, /self-hosting\.md/)
  })
})

describe('resolveApiAssetUrl', () => {
  it('prefixes API-relative hero image paths', () => {
    assert.equal(
      resolveApiAssetUrl(
        '/api/v1/courses/C-AIESS1/course-files/ff993800-114b-4316-ba70-27406837f8a5/content',
        'https://self.lextures.com',
      ),
      'https://self.lextures.com/api/v1/courses/C-AIESS1/course-files/ff993800-114b-4316-ba70-27406837f8a5/content',
    )
  })
})

describe('validateGeneratedPage', () => {
  it('flags empty body and missing title', () => {
    const errors = validateGeneratedPage({
      path: '/x',
      head: { title: '', description: 'ok', canonical: 'https://lextures.com/x' },
      bodyHtml: '',
    })
    assert.ok(errors.some(e => e.includes('empty body')))
    assert.ok(errors.some(e => e.includes('missing title')))
  })

  it('flags description over 160 chars', () => {
    const errors = validateGeneratedPage({
      path: '/x',
      head: {
        title: 'T',
        description: 'x'.repeat(161),
        canonical: 'https://lextures.com/x',
      },
      bodyHtml: '<h1>X</h1>',
    })
    assert.ok(errors.some(e => e.includes('description')))
  })
})

describe('outputPathForRoute', () => {
  it('maps / to dist/index.html and nested paths to folders', () => {
    assert.ok(outputPathForRoute('/').endsWith('index.html'))
    assert.ok(outputPathForRoute('/pricing').includes('pricing'))
    assert.ok(outputPathForRoute('/docs/self-hosting').includes('self-hosting'))
  })
})

describe('internal link graph', () => {
  it('computes crawl depth and inbound links', () => {
    const graph = buildLinkGraph([
      { path: '/', html: '<a href="/hub">Resource hub</a>' },
      { path: '/hub', html: '<a href="/hub/article">Assessment guide</a>' },
      { path: '/hub/article', html: '<a href="/">Lextures home</a>' },
    ])
    assert.equal(graph.nodes.find(node => node.path === '/hub/article').depth, 2)
    assert.deepEqual(validateLinkGraph(graph), [])
  })

  it('rejects orphans, excessive depth, and weak anchor text', () => {
    const graph = buildLinkGraph([
      { path: '/', html: '<a href="/a">click here</a>' },
      { path: '/a', html: '<a href="/b">B</a>' },
      { path: '/b', html: '<a href="/c">C</a>' },
      { path: '/c', html: '<a href="/deep">Deep</a>' },
      { path: '/deep', html: '' }, { path: '/orphan', html: '' },
    ])
    const errors = validateLinkGraph(graph).join('\n')
    assert.match(errors, /non-descriptive anchor/)
    assert.match(errors, /depth 4 exceeds 3/)
    assert.match(errors, /orphan page/)
  })
})
