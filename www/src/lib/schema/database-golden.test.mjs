import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { createServer } from 'vite'

test('database article structured data matches the golden fixture', async () => {
  globalThis.__LEXTURES_BUILD_CONTENT__ = { source: 'api', categories: [], redirects: [], articles: [{ path: '/blog/golden', kind: 'blog', slug: 'golden', locale: 'en', title: 'Golden article', description: 'Stable fixture', publishedAt: '2026-01-01T00:00:00Z', contentUpdatedAt: '2026-02-02T00:00:00Z', author: { slug: 'ada' }, citations: ['https://example.com/source'], bodyMd: 'Fixture body' }], authors: [{ slug: 'ada', name: 'Ada Example', jobTitle: 'Editor', bio: 'Writes.', knowsAbout: ['learning'], status: 'active', links: { sameAs: ['https://example.com/ada'] } }] }
  const vite = await createServer({ root: process.cwd(), server: { middlewareMode: true, watch: null, hmr: false }, appType: 'custom' })
  try {
    const { blogPostGraph } = await vite.ssrLoadModule('/src/lib/schema/page-graphs.ts')
    const graph = blogPostGraph({ path: '/blog/golden', origin: 'https://lextures.com', params: { slug: 'golden' } })
    const person = graph.find(node => node['@id'] === 'https://lextures.com/authors/ada#person')
    const article = graph.find(node => node['@id'] === 'https://lextures.com/blog/golden#article')
    const actual = [{ type: person['@type'], id: person['@id'], name: person.name, sameAs: person.sameAs }, { type: article['@type'], id: article['@id'], headline: article.headline, datePublished: article.datePublished, dateModified: article.dateModified, authorId: article.author['@id'], citations: article.citation.map(item => item.url || item.name) }]
    assert.deepEqual(actual, JSON.parse(await readFile(new URL('./testdata/database-article.golden.json', import.meta.url))))
  } finally { await vite.close(); delete globalThis.__LEXTURES_BUILD_CONTENT__ }
})
