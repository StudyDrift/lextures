#!/usr/bin/env node
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { runChecks } from './core.mjs'

const args = process.argv.slice(2), onlyArg = args.find(a => a.startsWith('--only='))
const selected = onlyArg ? new Set(onlyArg.slice(7).split(',').filter(Boolean)) : null
const warnOnly = args.includes('--warn-only')
let previous = null
if (process.env.SEO_PREVIOUS_MANIFEST) {
  try { previous = JSON.parse(await readFile(process.env.SEO_PREVIOUS_MANIFEST, 'utf8')) } catch (error) { console.warn(`Previous manifest unavailable: ${error.message}`) }
}
try {
  const dist = path.resolve(process.env.SEO_DIST || 'dist')
  const results = await runChecks(dist, selected, previous)
  await mkdir(dist, { recursive: true })
  await writeFile(path.join(dist, '.seo-check.json'), JSON.stringify({ generatedAt: new Date().toISOString(), results }, null, 2))
  for (const result of results) {
    console.log(`${result.status.toUpperCase()} ${result.check}: ${result.findings.length} error(s), ${result.warnings.length} warning(s)`)
    for (const item of [...result.findings, ...result.warnings]) console.log(`  ${item.page}: ${item.message}`)
  }
  if (!warnOnly && results.some(r => r.status === 'fail')) process.exitCode = 1
} catch (error) {
  console.error(`SEO checks skipped: ${error.message}`)
  process.exitCode = warnOnly ? 0 : 1
}
