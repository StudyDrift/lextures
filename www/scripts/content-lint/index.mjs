#!/usr/bin/env node
import { readFile, readdir, writeFile, mkdir } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { analyzeContent } from './core.mjs'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
async function filesIn(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  return (await Promise.all(entries.map(entry => entry.isDirectory() ? filesIn(path.join(dir, entry.name)) : /\.mdx?$/.test(entry.name) ? [path.join(dir, entry.name)] : []))).flat()
}
const requested = process.argv.slice(2).filter(a => !a.startsWith('--'))
const files = requested.length ? requested.map(f => path.resolve(process.cwd(), f)) : [...await filesIn(path.join(ROOT, 'src/blog')), ...await filesIn(path.join(ROOT, 'src/docs'))]
const results = []
for (const file of files) results.push(analyzeContent(await readFile(file, 'utf8'), file, { enforce: /\.mdx$/.test(file) }))
for (const result of results) {
  console.log(`${path.relative(ROOT, result.file)}  ${result.score.toFixed(1)}/10  ${result.wordCount} words  grade ${result.readingLevel}`)
  for (const i of result.issues) console.log(`  ${i.severity === 'warning' ? 'WARN' : 'ERROR'} ${i.line}:${i.rule} ${i.message} Fix: ${i.fix}`)
}
if (process.argv.includes('--write-report')) {
  await mkdir(path.join(ROOT, 'dist'), { recursive: true })
  await writeFile(path.join(ROOT, 'dist/.content-quality.json'), JSON.stringify({ generatedAt: new Date().toISOString(), pages: results }, null, 2))
  await writeFile(path.join(ROOT, 'dist/.definitions.json'), JSON.stringify(results.flatMap(r => r.definitions.map(d => ({ ...d, source: `/${r.file.includes('/blog/') ? 'blog' : 'docs'}/${path.basename(r.file).replace(/\.mdx?$/, '')}` }))), null, 2))
}
if (results.some(r => r.issues.some(i => i.severity === 'error'))) process.exitCode = 1
