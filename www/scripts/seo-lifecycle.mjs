#!/usr/bin/env node
import { readFile, readdir, writeFile, mkdir } from 'node:fs/promises'
import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import path from 'node:path'

const root = path.resolve('src'), today = new Date(process.env.LIFECYCLE_DATE || Date.now())
async function files(dir) { return (await Promise.all((await readdir(dir, { withFileTypes:true })).map(e => e.isDirectory() ? files(path.join(dir,e.name)) : path.join(dir,e.name)))).flat() }
const content = (await Promise.all(['blog','docs'].map(async d => { try { return await files(path.join(root,d)) } catch { return [] } }))).flat().filter(f => f.endsWith('.md'))
const rows = [], missing = [], staleHelp = [], staleComparison = []
for (const file of content) {
  const source = await readFile(file, 'utf8'), front = source.match(/^---\n([\s\S]*?)\n---/)?.[1] || ''
  const field = name => front.match(new RegExp(`^${name}:\\s*["']?([^\\n"']+)`, 'm'))?.[1]?.trim()
  const owner = field('owner') || field('author'), reviewDue = field('reviewDue'), updated = field('updated') || field('date'), noindex = field('noindex')
  const rel = path.relative(process.cwd(), file), kind = rel.includes('/docs/') ? 'help' : rel.includes('/compare/') ? 'comparison' : 'content'
  const due = reviewDue ? new Date(`${reviewDue}T00:00:00Z`) : null
  if (!owner || !reviewDue) missing.push({ rel, owner, reviewDue })
  if (due && due < today) { rows.push({rel, reason:`past reviewDue (${reviewDue})`}); if (kind === 'help') staleHelp.push(rel); if (kind === 'comparison') staleComparison.push(rel) }
  if (updated) { const age = (today - new Date(`${updated.slice(0,10)}T00:00:00Z`))/864e5, limit = kind === 'comparison' ? 120 : kind === 'help' ? 180 : Infinity; if (age > limit) rows.push({rel, reason:`unverified ${Math.floor(age)} days`}) }
  if (noindex === 'true') rows.push({rel, reason:'noindex (front matter)'})
}
const ratio = (n,d) => d ? n/d : 0
const report = ['# SEO content lifecycle report','',`Generated: ${today.toISOString()}`,'',`Content files: ${content.length}`,'',`Missing owner/reviewDue: ${missing.length}`,'',...missing.map(x => `- ${x.rel}: ${!x.owner?'missing owner':''}${!x.owner&&!x.reviewDue?', ':''}${!x.reviewDue?'missing reviewDue':''}`),'','## Stale, unverified, and noindex pages','',...(rows.length ? rows.map(x => `- ${x.rel}: ${x.reason}`) : ['- None.']),'','## Analytics-dependent review queues','','Search Console categories are recorded as unavailable when credentials/data are not supplied: zero impressions after 180 days; declining traffic over 90 days. Import those queues during the weekly review.','','## Link health','','Orphans fail `seo:check`; near-orphans (<3 inbound links) are warnings in its machine-readable link graph output. External links are checked by the weekly live job.',''].join('\n')
let translationRows = []
try {
  const manifest = JSON.parse(await readFile('dist/.seo-manifest.json', 'utf8'))
  translationRows = (manifest.urls || []).filter(page => page.translationOf && page.translationStatus === 'stale').map(page => {
    const age = page.sourceUpdatedAt ? Math.floor((today - new Date(page.sourceUpdatedAt)) / 864e5) : null
    return `- ${page.path}: source ${page.translationOf} changed${age == null ? '' : ` ${age} days ago`}${age != null && age >= 90 ? ' — visible notice required' : ''}`
  })
} catch { /* build manifest is optional for source-only lifecycle runs */ }
const reportWithTranslations = `${report}\n## Translation staleness\n\n${translationRows.length ? translationRows.join('\n') : '- None.'}\n`
await mkdir('dist', {recursive:true}); await writeFile('dist/seo-lifecycle.md', reportWithTranslations)
console.log(reportWithTranslations)
if (process.env.SEO_BASE_SHA) {
  const { stdout } = await promisify(execFile)('git', ['diff', '--diff-filter=A', '--name-only', process.env.SEO_BASE_SHA, '--', 'www/src/blog', 'www/src/docs'], { cwd: path.resolve('..') })
  const added = new Set(stdout.trim().split('\n').filter(Boolean).map(f => f.replace(/^www\//, '')))
  const invalidNew = missing.filter(item => added.has(item.rel))
  if (invalidNew.length) { invalidNew.forEach(item => console.error(`New content requires owner/author and reviewDue: ${item.rel}`)); process.exitCode=1 }
}
if (ratio(staleHelp, content.filter(f=>f.includes('/docs/')).length) > .10) { console.error(`More than 10% of help articles are stale (${staleHelp.length}).`); process.exitCode=1 }
if (staleComparison.length && ratio(staleComparison, content.filter(f=>f.includes('/compare/')).length) > .10) { console.error(`More than 10% of comparison pages are stale (${staleComparison.length}).`); process.exitCode=1 }
