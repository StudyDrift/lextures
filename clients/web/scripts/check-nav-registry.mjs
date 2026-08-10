#!/usr/bin/env node
/**
 * UX.7 — Navigation registry integrity + collision check (FR-5, FR-20).
 *
 * Fails on:
 *   - duplicate destination ids within a scope
 *   - duplicate icons within a scope (non-utility)
 *   - near-duplicate labels within a scope (Levenshtein ≤2)
 *   - missing icon name in icons.tsx map (best-effort parse)
 *   - destination without section + priority
 *
 * Usage:
 *   node scripts/check-nav-registry.mjs
 *   node scripts/check-nav-registry.mjs --json
 */
import { readFileSync, readdirSync, existsSync } from 'node:fs'
import { join, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const navDir = join(webRoot, 'src/lib/nav')
const asJson = process.argv.includes('--json')

function levenshtein(a, b) {
  const s = a.toLowerCase()
  const t = b.toLowerCase()
  if (s === t) return 0
  if (!s.length) return t.length
  if (!t.length) return s.length
  const d = Array.from({ length: s.length + 1 }, () => Array(t.length + 1).fill(0))
  for (let i = 0; i <= s.length; i++) d[i][0] = i
  for (let j = 0; j <= t.length; j++) d[0][j] = j
  for (let i = 1; i <= s.length; i++) {
    for (let j = 1; j <= t.length; j++) {
      const cost = s[i - 1] === t[j - 1] ? 0 : 1
      d[i][j] = Math.min(d[i - 1][j] + 1, d[i][j - 1] + 1, d[i - 1][j - 1] + cost)
    }
  }
  return d[s.length][t.length]
}

function normaliseLabel(label) {
  return String(label)
    .toLowerCase()
    .replace(/['’]/g, '')
    .replace(/[^a-z0-9\s]/g, ' ')
    .replace(/\b(my|the|a|an)\b/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function labelsNearDuplicate(a, b) {
  const na = normaliseLabel(a)
  const nb = normaliseLabel(b)
  if (!na || !nb) return false
  if (na === nb) return true
  if (Math.min(na.length, nb.length) < 5) return false
  return levenshtein(na, nb) <= 2
}

/**
 * Parse registry TS files for destination object literals.
 * Heuristic — not a full TS parser; expects id/label/icon/section/priority fields.
 */
function parseDestinations(filePath) {
  const text = readFileSync(filePath, 'utf8')
  const dests = []
  // Match blocks starting with id: '...'
  const re =
    /\{\s*id:\s*'([^']+)'[\s\S]*?label:\s*'((?:\\'|[^'])*)'[\s\S]*?icon:\s*'([^']+)'[\s\S]*?section:\s*'([^']+)'[\s\S]*?priority:\s*(\d+)/g
  let m
  while ((m = re.exec(text)) !== null) {
    const block = m[0]
    const utility = /utility:\s*true/.test(block)
    dests.push({
      id: m[1],
      label: m[2].replace(/\\'/g, "'"),
      icon: m[3],
      section: m[4],
      priority: Number(m[5]),
      utility,
      file: filePath,
    })
  }
  return dests
}

function knownIcons() {
  const iconsPath = join(navDir, 'icons.tsx')
  if (!existsSync(iconsPath)) return new Set()
  const text = readFileSync(iconsPath, 'utf8')
  const set = new Set()
  const re = /^\s{2}([A-Z][A-Za-z0-9]+),?$/gm
  let m
  while ((m = re.exec(text)) !== null) {
    // Only collect from ICONS map roughly — also from import list
    set.add(m[1])
  }
  // Also from import { A, B }
  const imp = text.match(/import\s*\{([\s\S]*?)\}\s*from\s*'lucide-react'/)
  if (imp) {
    for (const part of imp[1].split(',')) {
      const name = part.trim().split(/\s+as\s+/)[0].trim()
      if (name) set.add(name)
    }
  }
  return set
}

function main() {
  const registryFiles = readdirSync(navDir)
    .filter((f) => f.startsWith('registry-') && f.endsWith('.ts'))
    .map((f) => join(navDir, f))

  if (!registryFiles.length) {
    console.error('No registry-*.ts files found in', navDir)
    process.exit(1)
  }

  const icons = knownIcons()
  const findings = []
  let total = 0

  for (const file of registryFiles) {
    const scope = file.includes('course') ? 'course' : file.includes('global') ? 'global' : 'other'
    const dests = parseDestinations(file)
    total += dests.length
    const byId = new Map()
    const byIcon = new Map()
    const list = dests.filter((d) => !d.utility)

    for (const d of dests) {
      if (!d.section || !Number.isFinite(d.priority)) {
        findings.push({ scope, kind: 'missing-meta', detail: `${d.id} missing section/priority` })
      }
      if (byId.has(d.id)) {
        findings.push({
          scope,
          kind: 'duplicate-id',
          detail: `duplicate id ${d.id}`,
        })
      } else {
        byId.set(d.id, d)
      }
      if (!d.utility) {
        if (byIcon.has(d.icon)) {
          findings.push({
            scope,
            kind: 'duplicate-icon',
            detail: `icon ${d.icon} used by ${byIcon.get(d.icon).id} and ${d.id}`,
          })
        } else {
          byIcon.set(d.icon, d)
        }
      }
      if (icons.size && !icons.has(d.icon)) {
        findings.push({
          scope,
          kind: 'unknown-icon',
          detail: `${d.id} icon ${d.icon} not in icons.tsx`,
        })
      }
    }

    for (let i = 0; i < list.length; i++) {
      for (let j = i + 1; j < list.length; j++) {
        if (labelsNearDuplicate(list[i].label, list[j].label)) {
          findings.push({
            scope,
            kind: 'near-duplicate-label',
            detail: `"${list[i].label}" (${list[i].id}) ≈ "${list[j].label}" (${list[j].id})`,
          })
        }
      }
    }
  }

  if (asJson) {
    console.log(JSON.stringify({ total, findings }, null, 2))
  } else {
    console.log(`nav-registry: ${total} destinations across ${registryFiles.length} scopes`)
    if (findings.length) {
      console.error(`nav-registry: ${findings.length} finding(s)`)
      for (const f of findings) {
        console.error(`  [${f.scope}] ${f.kind}: ${f.detail}`)
      }
    } else {
      console.log('nav-registry: 0 collisions')
    }
  }

  process.exit(findings.length ? 1 : 0)
}

main()
