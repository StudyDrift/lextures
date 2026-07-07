#!/usr/bin/env node
/**
 * Lists translation keys and source namespaces for translation-memory handoff (plan W01 §8).
 */
import { mkdir, readdir, readFile, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const webRoot = join(fileURLToPath(new URL('.', import.meta.url)), '..')
const enRoot = join(webRoot, 'public', 'locales', 'en')
const outPath = join(webRoot, 'dist', 'i18n-extract-manifest.json')

function flattenKeys(obj, prefix = '') {
  const keys = []
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, path))
    } else {
      keys.push({ key: path, value: String(v) })
    }
  }
  return keys
}

async function listNamespaces() {
  const files = await readdir(enRoot)
  return files.filter((f) => f.endsWith('.json')).map((f) => f.replace(/\.json$/, ''))
}

const manifest = []
const namespaces = await listNamespaces()
for (const ns of namespaces.sort()) {
  const source = `public/locales/en/${ns}.json`
  const raw = await readFile(join(enRoot, `${ns}.json`), 'utf8')
  const data = JSON.parse(raw)
  for (const entry of flattenKeys(data)) {
    manifest.push({
      namespace: ns,
      key: entry.key,
      en: entry.value,
      source,
    })
  }
}

await mkdir(join(webRoot, 'dist'), { recursive: true })
await writeFile(outPath, `${JSON.stringify({ generatedAt: new Date().toISOString(), keys: manifest }, null, 2)}\n`)
console.log(`[i18n:extract] wrote ${manifest.length} keys to ${outPath}`)
