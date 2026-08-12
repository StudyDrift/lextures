#!/usr/bin/env node
import { readdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'src/docs')
const files = (await Promise.all((await readdir(root, { withFileTypes: true })).filter(e => e.isDirectory()).map(async directory => (await readdir(path.join(root, directory.name))).filter(name => name.endsWith('.md')).map(name => path.join(root, directory.name, name))))).flat()
const cutoff = Date.now() - 180 * 86400000
const stale = []
for (const file of files) {
  const raw = await readFile(file, 'utf8')
  const verified = raw.match(/^updated:\s*['"]?([^'"\n]+)['"]?$/m)?.[1]?.trim()
  if (!verified || Date.parse(`${verified}T00:00:00Z`) < cutoff) stale.push(path.relative(root, file))
}
console.log(`${stale.length}/${files.length} help articles are stale.`)
if (stale.length) console.log(stale.join('\n'))
if (files.length && stale.length / files.length > 0.1) process.exitCode = 1
