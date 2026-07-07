#!/usr/bin/env node
/**
 * Verifies translation key parity: every key in en must exist in es, fr, and ar.
 * Fails CI when missing keys exceed I18N_MAX_MISSING_KEYS (default 0).
 */
import { readdir, readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'public', 'locales')
const baseLocale = 'en'
const requiredLocales = ['es', 'fr', 'ar']
const maxMissing = Number.parseInt(process.env.I18N_MAX_MISSING_KEYS ?? '0', 10)

function flattenKeys(obj, prefix = '') {
  const keys = []
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, path))
    } else {
      keys.push(path)
    }
  }
  return keys
}

async function loadNamespace(locale, ns) {
  const path = join(root, locale, `${ns}.json`)
  const raw = await readFile(path, 'utf8')
  return JSON.parse(raw)
}

async function listNamespaces(locale) {
  const dir = join(root, locale)
  const files = await readdir(dir)
  return files.filter((f) => f.endsWith('.json')).map((f) => f.replace(/\.json$/, ''))
}

let failures = 0

const namespaces = await listNamespaces(baseLocale)
for (const ns of namespaces) {
  const en = await loadNamespace(baseLocale, ns)
  const enKeys = new Set(flattenKeys(en))
  for (const locale of requiredLocales) {
    const target = await loadNamespace(locale, ns)
    const targetKeys = new Set(flattenKeys(target))
    for (const key of enKeys) {
      if (!targetKeys.has(key)) {
        console.error(`[i18n] missing key "${key}" in ${locale}/${ns}.json (present in en)`)
        failures++
      }
    }
  }
}

if (failures > maxMissing) {
  console.error(
    `[i18n] ${failures} missing translation key(s) (threshold ${maxMissing}). Add keys or raise I18N_MAX_MISSING_KEYS.`,
  )
  process.exit(1)
}

if (failures > 0) {
  console.warn(`[i18n] ${failures} missing key(s) within threshold ${maxMissing}`)
}

console.log(
  `[i18n] locale parity OK for ${namespaces.length} namespace(s) (${requiredLocales.join(', ')} vs en)`,
)
