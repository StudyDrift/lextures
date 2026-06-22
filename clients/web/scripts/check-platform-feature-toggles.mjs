#!/usr/bin/env node
/**
 * Ensures every ff* boolean in settings_platform.go has a Global platform toggle
 * (platform-feature-definitions.ts) or an explicit exemption.
 */
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const webRoot = join(here, '..')
const repoRoot = join(webRoot, '..', '..')

const settingsGo = readFileSync(
  join(repoRoot, 'server/internal/httpserver/settings_platform.go'),
  'utf8',
)
const definitionsTs = readFileSync(
  join(webRoot, 'src/components/settings/platform-feature-definitions.ts'),
  'utf8',
)
const exemptionsTs = readFileSync(
  join(webRoot, 'src/components/settings/platform-feature-exemptions.ts'),
  'utf8',
)

function parseServerFfKeys(source) {
  const structStart = source.indexOf('type platformSettingsJSON struct {')
  const structEnd = source.indexOf('\ntype platformSourcesJSON struct {', structStart)
  const block = source.slice(structStart, structEnd)
  const keys = new Set()
  const re = /\bFF\w+\s+bool\s+`json:"(ff[^"]+)"`/g
  for (const match of block.matchAll(re)) {
    keys.add(match[1])
  }
  return [...keys].sort()
}

function parseDefinitionFfKeys(source) {
  const keys = new Set()
  const re = /key:\s*'(ff[^']+)'/g
  for (const match of source.matchAll(re)) {
    keys.add(match[1])
  }
  return keys
}

function parseExemptFfKeys(source) {
  const keys = new Set()
  const re = /'(ff[^']+)'/g
  for (const match of source.matchAll(re)) {
    keys.add(match[1])
  }
  return keys
}

const serverKeys = parseServerFfKeys(settingsGo)
const toggleKeys = parseDefinitionFfKeys(definitionsTs)
const exemptKeys = parseExemptFfKeys(exemptionsTs)

const missing = serverKeys.filter((key) => !toggleKeys.has(key) && !exemptKeys.has(key))
const unknownToggles = [...toggleKeys].filter((key) => !serverKeys.includes(key))

let failed = false
if (missing.length > 0) {
  failed = true
  console.error('Missing Global platform toggles or exemptions for:', missing.join(', '))
}
if (unknownToggles.length > 0) {
  failed = true
  console.error('Toggle definitions reference unknown server ff* keys:', unknownToggles.join(', '))
}

if (failed) {
  process.exit(1)
}

console.log(`platform feature toggles OK (${serverKeys.length} server ff* flags)`)