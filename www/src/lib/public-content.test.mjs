import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { articleDate, mergePublishedArticles, parseContentPath, previewTokenFromSearch, publicArticleUrl, publicArticlesUrl } from './public-content-core.ts'

describe('parseContentPath', () => {
  it('parses blog and docs paths', () => {
    assert.deepEqual(parseContentPath('/blog/the-more-you-know-expanding-horizons'), {
      kind: 'blog',
      slug: 'the-more-you-know-expanding-horizons',
      locale: 'en',
      path: '/blog/the-more-you-know-expanding-horizons',
    })
    assert.deepEqual(parseContentPath('/docs/courses/building-modules-and-pages/'), {
      kind: 'doc',
      slug: 'building-modules-and-pages',
      category: 'courses',
      locale: 'en',
      path: '/docs/courses/building-modules-and-pages',
    })
  })

  it('strips a locale prefix', () => {
    assert.deepEqual(parseContentPath('/es/blog/hello-world'), {
      kind: 'blog',
      slug: 'hello-world',
      locale: 'es',
      path: '/es/blog/hello-world',
    })
  })

  it('ignores hubs and unknown shapes', () => {
    assert.equal(parseContentPath('/blog'), null)
    assert.equal(parseContentPath('/docs/courses'), null)
    assert.equal(parseContentPath('/pricing'), null)
    assert.equal(parseContentPath('/blog/Not-A-Slug'), null)
  })
})

describe('public content URLs', () => {
  it('builds article and listing URLs', () => {
    assert.match(
      publicArticleUrl({ kind: 'blog', slug: 'hello', locale: 'en', path: '/blog/hello' }),
      /\/api\/v1\/public\/content\/articles\/blog\/hello$/,
    )
    assert.match(
      publicArticleUrl(
        { kind: 'doc', slug: 'install', category: 'self-hosting', locale: 'en', path: '/docs/self-hosting/install' },
        'tok',
      ),
      /\/articles\/docs\/self-hosting\/install\?preview_token=tok$/,
    )
    assert.match(publicArticlesUrl('blog'), /kind=blog/)
  })

  it('reads preview tokens from the query string', () => {
    assert.equal(previewTokenFromSearch('?preview_token=abc'), 'abc')
    assert.equal(previewTokenFromSearch('q=1'), undefined)
  })
})

describe('mergePublishedArticles', () => {
  it('adds live articles and keeps newest first', () => {
    const merged = mergePublishedArticles(
      [{ slug: 'older', date: '2026-01-01', title: 'Older' }],
      [
        { slug: 'the-more-you-know-expanding-horizons', date: '2026-08-18', title: 'The More You Know' },
        { slug: 'older', date: '2026-01-01', title: 'Older live' },
      ],
    )
    assert.equal(merged[0].slug, 'the-more-you-know-expanding-horizons')
    assert.equal(merged[1].title, 'Older live')
    assert.equal(articleDate('2026-08-18T02:34:27.626963Z'), '2026-08-18')
  })
})
