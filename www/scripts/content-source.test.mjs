import assert from 'node:assert/strict'
import { mkdtemp, writeFile, readFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { loadApiContent, mergeRedirects } from './content-source.mjs'

test('empty live index is retried and not written to cache', async t => {
  const originalFetch = globalThis.fetch
  const cacheDir = await mkdtemp(path.join(tmpdir(), 'lextures-content-empty-'))
  let indexCalls = 0
  globalThis.fetch = async url => {
    if (String(url).endsWith('/index')) {
      indexCalls++
      return new Response(JSON.stringify({ generatedAt: '1970-01-01T00:00:00Z', articles: [] }), { status: 200 })
    }
    return new Response('missing', { status: 404 })
  }
  t.after(() => { globalThis.fetch = originalFetch })
  const snapshot = await loadApiContent({ apiBase: 'https://example.test', cacheDir, userAgent: 'test', concurrency: 2 })
  assert.equal(snapshot.articles.length, 0)
  assert.ok(indexCalls >= 2)
  await assert.rejects(readFile(path.join(cacheDir, 'index.json')))
})

test('empty cache is ignored when the live index is unreachable', async t => {
  const originalFetch = globalThis.fetch
  const cacheDir = await mkdtemp(path.join(tmpdir(), 'lextures-content-stale-'))
  await writeFile(path.join(cacheDir, 'index.json'), JSON.stringify({ generatedAt: '1970-01-01T00:00:00Z', articles: [] }))
  globalThis.fetch = async () => { throw new Error('offline') }
  t.after(() => { globalThis.fetch = originalFetch })
  const snapshot = await loadApiContent({ apiBase: 'https://example.test', cacheDir, userAgent: 'test', concurrency: 2 })
  assert.equal(snapshot.articles.length, 0)
  assert.equal(snapshot.fallbackUsed, true)
})

test('API source fetches uncached bodies then reuses the hash cache', async t => {
  const originalFetch = globalThis.fetch
  const cacheDir = await mkdtemp(path.join(tmpdir(), 'lextures-content-'))
  let details = 0
  const index = { generatedAt: '2026-01-01T00:00:00Z', articles: [{ path: '/blog/test', kind: 'blog', slug: 'test', contentHash: 'abc123' }], categories: [], authors: [], redirects: [] }
  globalThis.fetch = async url => new Response(JSON.stringify(String(url).endsWith('/index') ? index : (details++, { ...index.articles[0], bodyMd: '# Test' })), { status: 200 })
  t.after(() => { globalThis.fetch = originalFetch })
  const cold = await loadApiContent({ apiBase: 'https://example.test', cacheDir, userAgent: 'test', concurrency: 2 })
  const warm = await loadApiContent({ apiBase: 'https://example.test', cacheDir, userAgent: 'test', concurrency: 2 })
  assert.equal(details, 1); assert.equal(cold.fetched, 1); assert.equal(warm.cacheHits, 1)
})

test('static redirects win conflicts', () => {
  assert.deepEqual(mergeRedirects([{ from: '/old', to: '/static', status: 301 }], [{ from: '/old', to: '/api', statusCode: 302 }, { from: '/other', to: '/new', statusCode: 301 }]), [{ from: '/old', to: '/static', status: 301 }, { from: '/other', to: '/new', status: 301 }])
})

test('content hreflang is omitted for a single locale and reciprocal otherwise', async () => {
  const { contentHreflangAlternates } = await import('./content-source.mjs')
  assert.deepEqual(contentHreflangAlternates({ availableLocales: [{ locale: 'en', path: '/blog/hello' }] }, 'https://lextures.com'), [])
  const alts = contentHreflangAlternates({
    availableLocales: [
      { locale: 'en', path: '/blog/hello' },
      { locale: 'es', path: '/es/blog/hola' },
    ],
  }, 'https://lextures.com')
  assert.deepEqual(alts, [
    { hreflang: 'en', href: 'https://lextures.com/blog/hello' },
    { hreflang: 'es', href: 'https://lextures.com/es/blog/hola' },
    { hreflang: 'x-default', href: 'https://lextures.com/blog/hello' },
  ])
})
