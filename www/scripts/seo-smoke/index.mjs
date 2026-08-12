#!/usr/bin/env node
const arg = process.argv.find(a => a.startsWith('--origin=')), origin = (arg?.slice(9) || process.env.SEO_ORIGIN || '').replace(/\/$/, '')
if (!/^https?:\/\//.test(origin)) throw new Error('Pass --origin=https://example.com')
const timeout = Number(process.env.SEO_SMOKE_TIMEOUT_MS || 15000), failures = []
async function get(url, agent, type) { const response = await fetch(url, {headers: agent ? {'user-agent':agent}:{}, signal:AbortSignal.timeout(timeout)}); const body = await response.text(); if (!response.ok || !body.trim()) failures.push(`${url}: HTTP ${response.status} or empty body`); if (type && !response.headers.get('content-type')?.includes(type)) failures.push(`${url}: expected content-type containing ${type}, found ${response.headers.get('content-type')}`); return {response,body} }
const index = await get(`${origin}/sitemap.xml`), childUrls = [...index.body.matchAll(/<loc>([^<]+)<\/loc>/g)].map(m=>m[1])
const pageUrls = []
for (const child of childUrls) { const result = await get(child); for (const m of result.body.matchAll(/<loc>([^<]+)<\/loc>/g)) pageUrls.push(m[1]) }
const targets = pageUrls.length > 1000 && !process.argv.includes('--exhaustive') ? pageUrls.filter((_,i)=>i%Math.ceil(pageUrls.length/1000)===0) : pageUrls
for (const url of targets) { const {response,body} = await get(url); const canonical = body.match(/<link\b[^>]*rel=["']canonical["'][^>]*href=["']([^"']+)/i)?.[1]; if (canonical && new URL(canonical).pathname !== new URL(response.url).pathname) failures.push(`${url}: canonical mismatch (${canonical})`) }
for (const fixed of ['/robots.txt','/llms.txt']) await get(`${origin}${fixed}`, null, 'text/plain')
let key = process.env.INDEXNOW_KEY
try { key ||= JSON.parse((await get(`${origin}/.seo-manifest.json`)).body).indexNowKey } catch { failures.push(`${origin}/.seo-manifest.json: cannot discover IndexNow key`) }
if (key) await get(`${origin}/${key}.txt`, null, 'text/plain')
const sample = pageUrls.filter(u=>u!==`${origin}/`).slice(0,10)
for (const agent of ['GPTBot','OAI-SearchBot','ClaudeBot','PerplexityBot']) for (const url of sample) { const {body}=await get(url,agent); const h1=body.match(/<h1\b[^>]*>([\s\S]*?)<\/h1>/i)?.[1].replace(/<[^>]+>/g,'').trim(); if (!h1 || !body.includes(h1)) failures.push(`${url}: ${agent} response lacks rendered h1`) }
console.log(`Checked ${targets.length} sitemap pages and ${sample.length * 4} AI-crawler responses.`)
if (failures.length) { failures.forEach(x=>console.error(x)); process.exitCode=1 }
