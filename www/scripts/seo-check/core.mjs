import { readFile, readdir, stat } from 'node:fs/promises'
import path from 'node:path'
import { validateHreflang } from './checks/hreflang.mjs'

const text = (html, re) => html.match(re)?.[1]?.trim() || ''
const normalizePath = value => {
  try { return new URL(value, 'https://lextures.com').pathname.replace(/\/$/, '') || '/' } catch { return value }
}

export async function walk(dir) {
  const out = []
  for (const name of await readdir(dir)) {
    const file = path.join(dir, name)
    if ((await stat(file)).isDirectory()) out.push(...await walk(file))
    else out.push(file)
  }
  return out
}

export async function loadSite(dist) {
  const manifest = JSON.parse(await readFile(path.join(dist, '.seo-manifest.json'), 'utf8'))
  const graph = JSON.parse(await readFile(path.join(dist, '.link-graph.json'), 'utf8'))
  const redirects = new Map()
  for (const line of (await readFile(path.join(dist, '_redirects'), 'utf8')).split('\n')) {
    const [from, to, status] = line.trim().split(/\s+/)
    if (from?.startsWith('/') && to?.startsWith('/') && /^(301|308)$/.test(status || '')) redirects.set(from, to)
  }
  return { manifest, graph, redirects }
}

const finding = (page, message) => ({ page, message, line: null })

export async function runChecks(dist, selected = null, previousManifest = null) {
  const { manifest, graph, redirects } = await loadSite(dist)
  const all = new Map(manifest.urls.map(u => [u.path, u]))
  const results = []
  const add = (check, findings, warnings = []) => {
    if (!selected || selected.has(check)) results.push({ check, status: findings.length ? 'fail' : 'pass', severity: findings.length ? 'error' : 'info', findings, warnings })
  }

  const htmlByPath = new Map()
  const outputFindings = []
  for (const page of manifest.urls) {
    const file = page.path === '/' ? path.join(dist, 'index.html') : path.join(dist, page.path.slice(1), 'index.html')
    let html = ''
    try { html = await readFile(file, 'utf8') } catch { outputFindings.push(finding(page.path, `Missing generated output: ${path.relative(dist, file)}`)); continue }
    htmlByPath.set(page.path, html)
    const body = text(html, /<body\b[^>]*>([\s\S]*?)<\/body>/i).replace(/<[^>]+>/g, '').trim()
    if (!body) outputFindings.push(finding(page.path, 'Generated page has an empty <body>.'))
  }
  add('manifest-parity', outputFindings)

  const titleFindings = [], titleWarnings = [], titleOwner = new Map(), descOwner = new Map()
  for (const page of manifest.urls) {
    if (!page.title) titleFindings.push(finding(page.path, 'Title is empty.'))
    if (!page.description) titleFindings.push(finding(page.path, 'Description is empty.'))
    if (page.title?.length > 60) titleFindings.push(finding(page.path, `Title is ${page.title.length} characters; maximum is 60.`))
    if (page.description && (page.description.length < 120 || page.description.length > 160)) titleWarnings.push(finding(page.path, `Description is ${page.description.length} characters; target is 120–160 (legacy debt).`))
    for (const [value, owners, label] of [[page.title, titleOwner, 'title'], [page.description, descOwner, 'description']]) {
      if (!value) continue
      if (owners.has(value)) titleFindings.push(finding(page.path, `Duplicate ${label} with ${owners.get(value)}.`))
      else owners.set(value, page.path)
    }
  }
  add('titles', titleFindings, titleWarnings)

  const canonicalFindings = []
  for (const page of manifest.urls) {
    const actual = text(htmlByPath.get(page.path) || '', /<link\b[^>]*rel=["']canonical["'][^>]*href=["']([^"']+)/i)
    const expected = redirects.get(page.path) || page.path
    if (!actual || !/^https:\/\//.test(actual) || normalizePath(actual) !== normalizePath(expected)) canonicalFindings.push(finding(page.path, `Canonical must be absolute and ${redirects.has(page.path) ? 'point to the redirect target' : 'self-referential'}; found ${actual || 'none'}.`))
  }
  add('canonicals', canonicalFindings)

  const sitemapXmls = (await walk(path.join(dist, 'sitemaps'))).filter(f => f.endsWith('.xml'))
  const sitemapDocuments = await Promise.all(sitemapXmls.map(file => readFile(file, 'utf8')))
  const sitemapPaths = new Set()
  for (const file of sitemapXmls) for (const m of (await readFile(file, 'utf8')).matchAll(/<loc>([^<]+)<\/loc>/g)) sitemapPaths.add(normalizePath(m[1]))
  const sitemapFindings = []
  for (const page of manifest.urls) {
    const should = page.sitemap && !String(page.robots).includes('noindex')
    if (should !== sitemapPaths.has(normalizePath(page.path))) sitemapFindings.push(finding(page.path, should ? 'Indexable manifest route is missing from sitemaps.' : 'Non-indexable route appears in a sitemap.'))
  }
  for (const route of sitemapPaths) if (!all.has(route)) sitemapFindings.push(finding(route, 'Sitemap URL is absent from the manifest.'))
  add('sitemap-parity', sitemapFindings)
  add('hreflang', validateHreflang({ manifest, htmlByPath, sitemapXmls: sitemapDocuments }))

  const schemaFindings = []
  for (const [page, html] of htmlByPath) for (const match of html.matchAll(/<script\b[^>]*type=["']application\/ld\+json["'][^>]*>([\s\S]*?)<\/script>/gi)) {
    let data
    try { data = JSON.parse(match[1]) } catch (error) { schemaFindings.push(finding(page, `JSON-LD does not parse: ${error.message}`)); continue }
    const nodes = data['@graph'] || [data], ids = new Set(nodes.map(n => n?.['@id']).filter(Boolean))
    const visit = value => { if (!value || typeof value !== 'object') return; if (Array.isArray(value)) return value.forEach(visit); if (typeof value['@id'] === 'string' && !/^https?:\/\//.test(value['@id'])) schemaFindings.push(finding(page, `JSON-LD @id is not absolute: ${value['@id']}`)); if (Object.keys(value).length === 1 && value['@id'] && !ids.has(value['@id'])) schemaFindings.push(finding(page, `Dangling JSON-LD @id: ${value['@id']}`)); Object.entries(value).filter(([k]) => k !== '@id').forEach(([,v]) => visit(v)) }
    visit(nodes)
  }
  add('schema-validate', schemaFindings)

  const graphFindings = [], graphWarnings = []
  for (const node of graph.nodes || []) {
    if (node.path !== '/' && node.inbound === 0) graphWarnings.push(finding(node.path, 'Orphan page has no internal inbound links (legacy IA debt).'))
    if (node.depth == null || node.depth > 3) graphWarnings.push(finding(node.path, `Page depth is ${node.depth ?? 'unreachable'}; maximum is 3 (legacy IA debt).`))
  }
  add('link-graph', graphFindings, graphWarnings)

  const broken = []
  for (const edge of graph.edges || []) if (!all.has(edge.to) && !redirects.has(edge.to) && !edge.to.startsWith('/assets/') && !edge.to.includes('#')) broken.push(finding(edge.from, `Broken internal link: ${edge.to}`))
  add('broken-links', broken)

  const redirectFindings = []
  for (const [from, to] of redirects) { if (redirects.has(to)) redirectFindings.push(finding(from, `Redirect chain targets ${to}.`)); if (!all.has(to)) redirectFindings.push(finding(from, `Redirect target is not a manifest route: ${to}.`)) }
  add('redirects', redirectFindings)

  const ogFindings = []
  for (const [page, html] of htmlByPath) {
    if (String(all.get(page)?.robots).includes('noindex') || redirects.has(page)) continue
    const image = text(html, /<meta\b[^>]*property=["']og:image["'][^>]*content=["']([^"']+)/i)
    if (!image || !/\.(png|jpe?g|webp|avif)(?:\?|$)/i.test(image)) ogFindings.push(finding(page, `OG image must use a raster format; found ${image || 'none'}.`))
    else if (new URL(image).origin === manifest.origin) { const file = path.join(dist, new URL(image).pathname.slice(1)); try { await stat(file) } catch { ogFindings.push(finding(page, `OG image does not resolve in build output: ${image}.`)) } }
  }
  add('og-images', ogFindings)

  const removed = []
  if (previousManifest) for (const page of previousManifest.urls || []) if (!all.has(page.path) && !redirects.has(page.path)) removed.push(finding(page.path, 'Published URL was removed without a redirect.'))
  add('published-urls', removed)
  return results
}
