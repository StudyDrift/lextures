#!/usr/bin/env node
/**
 * SEO.4 FR-1 / FR-2 — assert gzipped JS/CSS budgets from a production build.
 *
 * Usage (after `npm run build`):
 *   node scripts/check-perf-budget.mjs
 *
 * Reads www/perf-budget.json and dist/.seo-manifest.json (optional) + dist assets.
 * Reports route class, metric, value, budget, and delta on failure.
 */

import { readFile, readdir, stat } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { gzipSync } from 'node:zlib'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const DIST = path.join(ROOT, 'dist')
const BUDGET_PATH = path.join(ROOT, 'perf-budget.json')

function kb(bytes) {
  return Math.round((bytes / 1024) * 10) / 10
}

async function gzipSize(filePath) {
  const buf = await readFile(filePath)
  return gzipSync(buf).length
}

async function collectAssets(dir, base = '') {
  /** @type {Array<{ rel: string, abs: string, bytes: number, gzip: number }>} */
  const out = []
  let entries
  try {
    entries = await readdir(dir, { withFileTypes: true })
  } catch {
    return out
  }
  for (const ent of entries) {
    const abs = path.join(dir, ent.name)
    const rel = path.posix.join(base, ent.name)
    if (ent.isDirectory()) {
      if (ent.name === 'fonts' || ent.name === 'assets' || ent.name === 'screenshots' || !base) {
        out.push(...(await collectAssets(abs, rel)))
      }
      continue
    }
    const st = await stat(abs)
    if (!st.isFile()) continue
    if (!/\.(js|css|woff2|svg|png|jpg|jpeg|webp|avif|html)$/i.test(ent.name)) continue
    const gzip = await gzipSize(abs)
    out.push({ rel, abs, bytes: st.size, gzip })
  }
  return out
}

function matchClass(routePath, budget) {
  const map = budget.routeClass || {}
  if (map[routePath]) return map[routePath]
  for (const [pattern, cls] of Object.entries(map)) {
    if (pattern === 'default') continue
    if (pattern.endsWith('/*')) {
      const prefix = pattern.slice(0, -1)
      if (routePath.startsWith(prefix)) return cls
    }
  }
  return map.default || 'content'
}

/**
 * Estimate transferred JS/CSS for a route class from the Vite build graph.
 * Content (interactive:false) pages strip the main module → only CSS + fonts.
 * Interactive pages pay for entry + shared chunks (sum of all JS as upper bound).
 */
function estimateForClass(cls, assets, budget) {
  const jsAssets = assets.filter(a => a.rel.endsWith('.js'))
  const cssAssets = assets.filter(a => a.rel.endsWith('.css'))
  const fontAssets = assets.filter(a => a.rel.endsWith('.woff2'))

  // Entry + chunk JS. For content class we assume islands skip full hydration
  // and only ship CSS (+ optional tiny island ≤ 8 KB gzip).
  let jsGzip
  if (cls === 'content') {
    const island = jsAssets.filter(a => /static-island|web-vitals|nav-island/i.test(a.rel))
    jsGzip = island.reduce((s, a) => s + a.gzip, 0)
    // If no dedicated island yet, content pages still reference entry only when interactive:true;
    // treat as 0 JS when generate-site strips scripts (measured via HTML later).
    if (jsGzip === 0) jsGzip = 0
  } else {
    jsGzip = jsAssets.reduce((s, a) => s + a.gzip, 0)
  }

  const cssGzip = cssAssets.reduce((s, a) => s + a.gzip, 0)
  // Preload fonts are part of critical path transfer for both classes
  const fontGzip = fontAssets
    .filter(a => /lextures-400|lextures-600|spectral-400|spectral-600/.test(a.rel))
    .reduce((s, a) => s + a.gzip, 0)
  const totalGzip = jsGzip + cssGzip + fontGzip

  const limits = budget.classes[cls]
  return {
    cls,
    jsGzip,
    cssGzip,
    totalGzip,
    limits,
    requests: 1 + jsAssets.length + cssAssets.length + 4, // rough HTML+assets upper bound
  }
}

async function measureHtmlScripts(htmlPath) {
  try {
    const html = await readFile(htmlPath, 'utf8')
    const scripts = [...html.matchAll(/<script[^>]+src=["']([^"']+)["']/gi)].map(m => m[1])
    const styles = [...html.matchAll(/<link[^>]+href=["']([^"']+\.css[^"']*)["']/gi)].map(m => m[1])
    // Lazy media is below the fold and is not part of the initial page transfer/request budget.
    const criticalHtml = html
      .replace(/<picture\b[^>]*>[\s\S]*?<img\b[^>]*loading=["']lazy["'][^>]*>[\s\S]*?<\/picture>/gi, '')
      .replace(/<img\b[^>]*loading=["']lazy["'][^>]*>/gi, '')
    const assets = [
      ...criticalHtml.matchAll(/<(?:img|source)[^>]+(?:src|srcset)=["']([^"']+)["']/gi),
      ...criticalHtml.matchAll(/<link[^>]+rel=["'](?:preload|icon)["'][^>]+href=["']([^"']+)["']/gi),
    ].flatMap(m => m[1].split(',').map(value => value.trim().split(/\s+/)[0]))
    return { scripts, styles, referencedAssets: [...new Set(assets)], html }
  } catch {
    return { scripts: [], styles: [], referencedAssets: [], html: null }
  }
}

async function resolveAssetGzip(src, assets) {
  const clean = src.replace(/^\//, '').split('?')[0]
  const hit = assets.find(a => a.rel === clean || a.rel.endsWith(clean) || `/${a.rel}` === src)
  if (hit) return hit.gzip
  // Try dist path
  const abs = path.join(DIST, clean)
  try {
    return await gzipSize(abs)
  } catch {
    return 0
  }
}

/**
 * Walk static ESM imports from an entry (and vite preload helpers) so budgets
 * include shared chunks, not just the entry file size.
 */
async function collectModuleGraphGzip(entrySrc, assets, seen = new Set()) {
  const clean = entrySrc.replace(/^\//, '').split('?')[0]
  if (seen.has(clean)) return 0
  seen.add(clean)

  let abs = path.join(DIST, clean)
  const hit = assets.find(a => a.rel === clean || a.rel.endsWith(clean))
  if (hit) abs = hit.abs

  let source
  try {
    source = await readFile(abs, 'utf8')
  } catch {
    return 0
  }

  let total = await gzipSize(abs)

  // import "..." / from "..." / __vite__mapDeps string lists
  const re = /(?:from|import)\s*["'](\.?\.?\/[^"']+\.js)["']/g
  let m
  while ((m = re.exec(source))) {
    const dep = m[1]
    // Resolve relative to the current file
    const baseDir = path.posix.dirname(clean)
    const resolved = path.posix.normalize(
      dep.startsWith('.') ? path.posix.join(baseDir, dep) : dep.replace(/^\//, ''),
    )
    total += await collectModuleGraphGzip('/' + resolved, assets, seen)
  }

  // Vite preload map: "assets/foo.js"
  const mapRe = /["'](assets\/[^"']+\.js)["']/g
  while ((m = mapRe.exec(source))) {
    // Only count chunks that are static deps of the entry graph for interactive:
    // mapDeps includes all possible routes — do NOT sum them for initial load.
    // Skip mapDeps arrays; only static import walk above counts.
  }

  return total
}

async function main() {
  const budget = JSON.parse(await readFile(BUDGET_PATH, 'utf8'))
  const assets = await collectAssets(DIST)

  if (!assets.length) {
    console.error('[perf-budget] dist/ is empty — run npm run build first')
    process.exit(1)
  }

  const failures = []
  const rows = []

  // Audit every indexable generated route, not only the Lighthouse sample.
  let routes = budget.benchmarkUrls || []
  try {
    const manifest = JSON.parse(await readFile(path.join(DIST, '.seo-manifest.json'), 'utf8'))
    routes = manifest.urls
      .filter(entry => !String(entry.robots || '').startsWith('noindex'))
      .map(entry => entry.path)
  } catch {
    failures.push({
      route: '*', metric: 'manifest', value: 0, budget: 1, delta: -1,
      message: 'missing dist/.seo-manifest.json; cannot audit every route',
    })
  }

  for (const route of [...new Set(routes)]) {
    const cls = matchClass(route, budget)
    const limits = budget.classes[cls]
    const htmlRel =
      route === '/' ? 'index.html' : path.posix.join(route.replace(/^\//, ''), 'index.html')
    const htmlPath = path.join(DIST, htmlRel)
    const { scripts, styles, referencedAssets, html } = await measureHtmlScripts(htmlPath)
    if (html == null) {
      failures.push({
        route, metric: 'html', value: 0, budget: 1, delta: -1,
        message: `missing ${htmlRel}`,
      })
      continue
    }

    let jsGzip = 0
    const seen = new Set()
    for (const s of scripts) {
      // Skip third-party
      if (/^https?:\/\//i.test(s)) {
        if (/googletagmanager|google-analytics|gtag|fonts\./i.test(s)) continue
        continue
      }
      jsGzip += await collectModuleGraphGzip(s, assets, seen)
    }
    let cssGzip = 0
    for (const s of styles) {
      if (/^https?:\/\//i.test(s) && /fonts\.googleapis/i.test(s)) {
        failures.push({
          route,
          metric: 'thirdPartyFonts',
          value: s,
          budget: 0,
          delta: s,
          message: 'Google Fonts stylesheet still referenced',
        })
        continue
      }
      cssGzip += await resolveAssetGzip(s, assets)
    }
    // Inline module scripts count as JS when present
    if (/fonts\.googleapis\.com|fonts\.gstatic\.com/i.test(html)) {
      failures.push({
        route,
        metric: 'thirdPartyFonts',
        value: 1,
        budget: 0,
        delta: 1,
        message: 'fonts.googleapis.com / fonts.gstatic.com reference found in HTML',
      })
    }

    let mediaGzip = 0
    let mediaRequests = 0
    for (const src of referencedAssets) {
      if (/^(?:https?:)?\/\//i.test(src) || src.startsWith('data:')) continue
      const size = await resolveAssetGzip(src, assets)
      if (size > 0) {
        mediaGzip += size
        mediaRequests += 1
      }
    }
    const htmlGzip = gzipSync(Buffer.from(html)).length
    const requests = 1 + seen.size + styles.length + mediaRequests
    const total = jsGzip + cssGzip + mediaGzip + htmlGzip

    rows.push({
      route,
      cls,
      jsGzipKb: kb(jsGzip),
      cssGzipKb: kb(cssGzip),
      totalGzipKb: kb(total),
      requests,
      jsBudget: limits.jsGzipKb,
      cssBudget: limits.cssGzipKb,
      totalBudget: limits.totalGzipKb,
    })

    if (kb(jsGzip) > limits.jsGzipKb) {
      failures.push({
        route,
        metric: 'jsGzipKb',
        value: kb(jsGzip),
        budget: limits.jsGzipKb,
        delta: Math.round((kb(jsGzip) - limits.jsGzipKb) * 10) / 10,
      })
    }
    if (kb(cssGzip) > limits.cssGzipKb) {
      failures.push({
        route,
        metric: 'cssGzipKb',
        value: kb(cssGzip),
        budget: limits.cssGzipKb,
        delta: Math.round((kb(cssGzip) - limits.cssGzipKb) * 10) / 10,
      })
    }
    if (kb(total) > limits.totalGzipKb) {
      failures.push({
        route,
        metric: 'totalGzipKb',
        value: kb(total),
        budget: limits.totalGzipKb,
        delta: Math.round((kb(total) - limits.totalGzipKb) * 10) / 10,
      })
    }
    if (requests > limits.requests) {
      failures.push({
        route,
        metric: 'requests',
        value: requests,
        budget: limits.requests,
        delta: requests - limits.requests,
      })
    }
  }

  // Global: no react-markdown in client graph
  const jsText = (
    await Promise.all(
      assets.filter(a => a.rel.endsWith('.js')).map(async a => (await readFile(a.abs, 'utf8')).slice(0, 500000)),
    )
  ).join('\n')
  if (/react-markdown|remark-gfm|ReactMarkdown/i.test(jsText)) {
    failures.push({
      route: '*',
      metric: 'clientMarkdown',
      value: 1,
      budget: 0,
      delta: 1,
      message: 'react-markdown / remark-gfm still present in client JS (FR-5)',
    })
  }

  // Bundle summary
  const allJs = assets.filter(a => a.rel.endsWith('.js')).reduce((s, a) => s + a.gzip, 0)
  const allCss = assets.filter(a => a.rel.endsWith('.css')).reduce((s, a) => s + a.gzip, 0)
  console.log('[perf-budget] asset totals (gzip): JS', kb(allJs), 'KB · CSS', kb(allCss), 'KB')
  console.log('[perf-budget] per-route:')
  for (const r of rows) {
    console.log(
      `  ${r.route.padEnd(40)} ${r.cls.padEnd(12)} js=${String(r.jsGzipKb).padStart(6)}/${r.jsBudget}  css=${String(r.cssGzipKb).padStart(5)}/${r.cssBudget}  total=${String(r.totalGzipKb).padStart(6)}/${r.totalBudget} KB  requests=${r.requests}`,
    )
  }

  // Also estimate interactive upper bound
  const interactiveEst = estimateForClass('interactive', assets, budget)
  console.log(
    `[perf-budget] interactive class upper bound (all JS chunks): ${kb(interactiveEst.jsGzip)} KB gzip`,
  )

  if (failures.length) {
    console.error('\n[perf-budget] FAILED:')
    for (const f of failures) {
      if (f.message) {
        console.error(`  ${f.route}  ${f.metric}: ${f.message}`)
      } else {
        console.error(
          `  ${f.route}  ${f.metric}=${f.value}  budget=${f.budget}  delta=+${f.delta}`,
        )
      }
    }
    process.exit(1)
  }

  console.log('[perf-budget] OK — all benchmark routes within budget')
}

main().catch(err => {
  console.error(err)
  process.exit(1)
})
