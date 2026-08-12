#!/usr/bin/env node
// MC.13 FR-1/FR-2: builds dist/docs-search-index.json from the database-backed
// content source (MC.7's API loader) instead of walking
// the filesystem, so the index reflects published/noindex-filtered help articles
// without an engineer editing files.
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { loadApiContent } from './content-source.mjs'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const outputPath = path.join(root, 'dist/docs-search-index.json')

const API_BASE = process.env.API_BASE || 'http://localhost:8080'
const CONTENT_API_BASE = (process.env.CONTENT_API_BASE || API_BASE).replace(/\/$/, '')
const CONTENT_CACHE_DIR = path.resolve(root, process.env.CONTENT_CACHE_DIR || '.content-cache')
const CONCURRENCY = Math.max(1, Number(process.env.GENERATE_CONCURRENCY || 8))
const PRERENDER_UA = 'lextures-www-docs-search/1.0'

function headingsFrom(body) {
  return [...body.matchAll(/^#{2,4}\s+(.+)$/gm)].map(match => match[1])
}

async function itemsFromApi() {
  const snapshot = await loadApiContent({ apiBase: CONTENT_API_BASE, cacheDir: CONTENT_CACHE_DIR, concurrency: CONCURRENCY, userAgent: PRERENDER_UA })
  const items = (snapshot.articles || [])
    .filter(article => article.kind === 'doc' && !article.noindex)
    .map(article => ({
      title: article.title,
      description: article.description,
      category: article.categorySlug || '',
      path: article.path,
      headings: headingsFrom(article.bodyMd || ''),
    }))
  if (items.length === 0 && snapshot.fallbackUsed) {
    // Content API was unreachable and there was no usable cache: keep whatever
    // index the previous deployment produced rather than publishing an empty one.
    try {
      const previous = JSON.parse(await readFile(outputPath, 'utf8'))
      console.warn(`[generate-docs-search] WARN: no content available from API or cache; keeping previous index (${previous.length} articles).`)
      return previous
    } catch {
      console.warn('[generate-docs-search] WARN: no content available and no previous index to fall back to; writing an empty index.')
    }
  }
  return items
}

const items = await itemsFromApi()
const output = `${JSON.stringify(items)}\n`
if (Buffer.byteLength(output) > 150 * 1024) throw new Error(`Docs search index exceeds 150 KiB: ${Buffer.byteLength(output)} bytes`)
await mkdir(path.join(root, 'dist'), { recursive: true })
await writeFile(outputPath, output)
console.log(`Generated docs search index (source=api) with ${items.length} articles (${Buffer.byteLength(output)} bytes).`)
