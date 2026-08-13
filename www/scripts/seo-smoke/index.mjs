#!/usr/bin/env node
/**
 * Post-deploy smoke checks (SEO.16 FR-17…FR-19).
 *
 * GitHub Pages serves directory indexes with a trailing-slash redirect while
 * our URL policy / canonicals omit the slash — compare normalized pathnames.
 *
 * AI crawler probes spoof User-Agent from CI IPs. Cloudflare may 403 those as
 * unverified bot impersonation even when real crawlers (verified IPs) and
 * robots.txt allow access. In that case we still require robots allow + a
 * rendered h1 on an anonymous fetch (the JS-shell failure mode).
 */
import {
  AI_AGENTS,
  agentAllowedByRobots,
  extractCanonical,
  extractH1,
  isEdgeBotBlock,
  normalizePath,
} from './lib.mjs'

const arg = process.argv.find(a => a.startsWith('--origin='))
const origin = (arg?.slice(9) || process.env.SEO_ORIGIN || '').replace(/\/$/, '')
if (!/^https?:\/\//.test(origin)) throw new Error('Pass --origin=https://example.com')

const timeout = Number(process.env.SEO_SMOKE_TIMEOUT_MS || 15000)
const failures = []
const warnings = []

async function get(url, agent, type) {
  const response = await fetch(url, {
    headers: agent ? { 'user-agent': agent } : {},
    signal: AbortSignal.timeout(timeout),
  })
  const body = await response.text()
  const contentType = response.headers.get('content-type') || ''
  if (!response.ok || !body.trim()) {
    if (!(agent && isEdgeBotBlock(response.status, body, contentType))) {
      failures.push(`${url}: HTTP ${response.status} or empty body`)
    }
  } else if (type && !contentType.includes(type)) {
    failures.push(`${url}: expected content-type containing ${type}, found ${contentType}`)
  }
  return { response, body, contentType }
}

const index = await get(`${origin}/sitemap.xml`)
const childUrls = [...index.body.matchAll(/<loc>([^<]+)<\/loc>/g)].map(m => m[1])
const pageUrls = []
for (const child of childUrls) {
  const result = await get(child)
  for (const m of result.body.matchAll(/<loc>([^<]+)<\/loc>/g)) pageUrls.push(m[1])
}
const targets =
  pageUrls.length > 1000 && !process.argv.includes('--exhaustive')
    ? pageUrls.filter((_, i) => i % Math.ceil(pageUrls.length / 1000) === 0)
    : pageUrls

for (const url of targets) {
  const { response, body } = await get(url)
  const canonical = extractCanonical(body)
  if (
    canonical &&
    normalizePath(canonical) !== normalizePath(response.url)
  ) {
    failures.push(`${url}: canonical mismatch (${canonical})`)
  }
}

const robots = await get(`${origin}/robots.txt`, null, 'text/plain')
await get(`${origin}/llms.txt`, null, 'text/plain')

let key = process.env.INDEXNOW_KEY
try {
  key ||= JSON.parse((await get(`${origin}/.seo-manifest.json`)).body).indexNowKey
} catch {
  failures.push(`${origin}/.seo-manifest.json: cannot discover IndexNow key`)
}
if (key) await get(`${origin}/${key}.txt`, null, 'text/plain')

const sample = pageUrls.filter(u => u !== `${origin}/`).slice(0, 10)
for (const agent of AI_AGENTS) {
  for (const url of sample) {
    const { response, body, contentType } = await get(url, agent)
    if (isEdgeBotBlock(response.status, body, contentType)) {
      if (!agentAllowedByRobots(robots.body, agent)) {
        failures.push(`${url}: ${agent} blocked by edge and disallowed in robots.txt`)
        continue
      }
      const anon = await get(url)
      const h1 = extractH1(anon.body)
      if (!h1 || !anon.body.includes(h1)) {
        failures.push(`${url}: ${agent} edge-blocked and anonymous response lacks rendered h1`)
      } else {
        warnings.push(
          `${url}: ${agent} edge-blocked from CI IP (robots allow + rendered h1 ok; verify Cloudflare AI bot policy)`,
        )
      }
      continue
    }
    const h1 = extractH1(body)
    if (!h1 || !body.includes(h1)) {
      failures.push(`${url}: ${agent} response lacks rendered h1`)
    }
  }
}

console.log(
  `Checked ${targets.length} sitemap pages and ${sample.length * AI_AGENTS.length} AI-crawler responses.`,
)
for (const w of warnings) console.warn(w)
if (failures.length) {
  failures.forEach(x => console.error(x))
  process.exitCode = 1
}
