#!/usr/bin/env node
import { mkdir, readdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
async function markdownFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map(entry => entry.isDirectory() ? markdownFiles(path.join(directory, entry.name)) : entry.name.endsWith('.md') && directory !== path.join(root, 'src/docs') ? [path.join(directory, entry.name)] : []))
  return nested.flat()
}
function parse(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/)
  const meta = {}
  for (const line of (match?.[1] || '').split(/\r?\n/)) {
    const colon = line.indexOf(':')
    if (colon > 0) meta[line.slice(0, colon).trim()] = line.slice(colon + 1).trim().replace(/^['"]|['"]$/g, '')
  }
  return { meta, body: match?.[2] || raw }
}
const items = []
for (const file of await markdownFiles(path.join(root, 'src/docs'))) {
  const { meta, body } = parse(await readFile(file, 'utf8'))
  const relative = path.relative(path.join(root, 'src/docs'), file).replace(/\.md$/, '')
  const [category, slug] = relative.split('/')
  items.push({ title: meta.title, description: meta.description, category, path: `/docs/${category}/${slug}`, headings: [...body.matchAll(/^#{2,4}\s+(.+)$/gm)].map(match => match[1]) })
}
const output = `${JSON.stringify(items)}\n`
if (Buffer.byteLength(output) > 150 * 1024) throw new Error(`Docs search index exceeds 150 KiB: ${Buffer.byteLength(output)} bytes`)
await mkdir(path.join(root, 'dist'), { recursive: true })
await writeFile(path.join(root, 'dist/docs-search-index.json'), output)
console.log(`Generated docs search index with ${items.length} articles (${Buffer.byteLength(output)} bytes).`)
