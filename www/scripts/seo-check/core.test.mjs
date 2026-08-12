import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { runChecks } from './core.mjs'

test('reports duplicate titles and removed URLs without redirects', async () => {
  const dist = await mkdtemp(path.join(tmpdir(), 'seo-check-'))
  await mkdir(path.join(dist, 'a'), { recursive: true }); await mkdir(path.join(dist, 'b')); await mkdir(path.join(dist, 'sitemaps'))
  const page = p => `<!doctype html><body><h1>${p}</h1><link rel="canonical" href="https://lextures.com${p}"><meta property="og:image" content="https://lextures.com/og.png"></body>`
  await Promise.all([writeFile(path.join(dist, 'a/index.html'), page('/a')), writeFile(path.join(dist, 'b/index.html'), page('/b')), writeFile(path.join(dist, 'og.png'), 'x'), writeFile(path.join(dist, '_redirects'), ''), writeFile(path.join(dist, '.link-graph.json'), JSON.stringify({nodes:[{path:'/a',depth:1,inbound:1},{path:'/b',depth:1,inbound:1}],edges:[]})), writeFile(path.join(dist, 'sitemaps/pages.xml'), '<urlset><url><loc>https://lextures.com/a</loc></url><url><loc>https://lextures.com/b</loc></url></urlset>')])
  const urls = ['/a','/b'].map(p => ({path:p,title:'Duplicate',description:`${p} ${'useful description '.repeat(9)}`,canonical:`https://lextures.com${p}`,robots:'index,follow',sitemap:true}))
  await writeFile(path.join(dist, '.seo-manifest.json'), JSON.stringify({origin:'https://lextures.com',urls}))
  const results = await runChecks(dist, new Set(['titles','published-urls']), {urls:[{path:'/gone'}]})
  assert.equal(results.find(r => r.check === 'titles').findings.length, 1)
  assert.equal(results.find(r => r.check === 'published-urls').findings[0].page, '/gone')
})
