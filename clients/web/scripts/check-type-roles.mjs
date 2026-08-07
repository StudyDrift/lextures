#!/usr/bin/env node
/**
 * UX.3 — forbid raw Tailwind type sizes in feature TSX (FR-5, AC-1).
 *
 * Metrics: type_role_violations, sub_13px_text_instances
 *
 * Usage:
 *   npm run type:purity
 *   node scripts/check-type-roles.mjs --write-allowlist
 *   node scripts/check-type-roles.mjs --json
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const allowlistPath = join(webRoot, 'type-role-allowlist.json')

/** Raw size utilities that must migrate to semantic roles. */
const RAW_SIZE =
  /\btext-(?:xs|sm|base|lg|xl|2xl|3xl|4xl|5xl|6xl|7xl|8xl|9xl)\b|\btext-\[\d+(?:\.\d+)?(?:px|rem)\]/g

/** Sub-13px arbitrary sizes (decorative floor violations). */
const SUB_13 =
  /\btext-\[(?:1[0-2]|[0-9])(?:\.\d+)?px\]|\btext-\[0\.(?:5|6|7|75)rem\]/g

const HARD_EXEMPT = [
  /src\/pages\/typeface-page\.tsx$/,
  /src\/pages\/design\//,
  /src\/lib\/tokens\//,
  /src\/__tests__\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
]

function walk(dir, pred) {
  const files = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      if (name === 'node_modules' || name === 'dist') continue
      files.push(...walk(path, pred))
    } else if (pred(name)) {
      files.push(path)
    }
  }
  return files
}

function countMatches(text, re) {
  const m = text.match(re)
  return m ? m.length : 0
}

function isExempt(rel) {
  return HARD_EXEMPT.some((re) => re.test(rel.replace(/\\/g, '/')))
}

const writeAllowlist = process.argv.includes('--write-allowlist')
const asJson = process.argv.includes('--json')

const files = walk(srcRoot, (n) => n.endsWith('.tsx'))
/** @type {Record<string, number>} */
const found = {}
let sub13Total = 0

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  const text = readFileSync(file, 'utf8')
  const n = countMatches(text, RAW_SIZE)
  sub13Total += countMatches(text, SUB_13)
  if (n > 0) found[rel] = n
}

if (writeAllowlist) {
  const sorted = Object.fromEntries(
    Object.entries(found).sort((a, b) => a[0].localeCompare(b[0])),
  )
  writeFileSync(
    allowlistPath,
    JSON.stringify(
      {
        description:
          'UX.3 type-role allowlist. Counts may only decrease. Empty when migration complete (AC-1).',
        generatedAt: new Date().toISOString(),
        files: sorted,
      },
      null,
      2,
    ) + '\n',
  )
  console.log(`Wrote allowlist with ${Object.keys(sorted).length} files (${Object.values(sorted).reduce((a, b) => a + b, 0)} violations).`)
  process.exit(0)
}

/** @type {{ files?: Record<string, number> }} */
let allowlist = { files: {} }
if (existsSync(allowlistPath)) {
  allowlist = JSON.parse(readFileSync(allowlistPath, 'utf8'))
}
const allowed = allowlist.files ?? {}

/** @type {{ file: string; count: number; allowed: number; reason: string }[]} */
const violations = []
/** @type {string[]} */
const ratchetUps = []
let totalFound = 0
let totalAllowed = 0

for (const [file, count] of Object.entries(found)) {
  totalFound += count
  const cap = allowed[file]
  if (cap == null) {
    violations.push({ file, count, allowed: 0, reason: 'not in allowlist' })
  } else if (count > cap) {
    violations.push({ file, count, allowed: cap, reason: 'exceeds allowlist (ratchet up)' })
    ratchetUps.push(file)
  }
}

for (const [file, cap] of Object.entries(allowed)) {
  totalAllowed += cap
  if (!(file in found) && cap > 0) {
    // File cleaned — fine; allowlist may still list it until --write-allowlist
  }
}

if (asJson) {
  console.log(
    JSON.stringify(
      {
        type_role_violations: totalFound,
        sub_13px_text_instances: sub13Total,
        allowlist_budget: totalAllowed,
        new_or_increased: violations,
      },
      null,
      2,
    ),
  )
  process.exit(violations.length ? 1 : 0)
}

console.log('UX.3 type-role purity check\n')
console.log(`  type_role_violations: ${totalFound}`)
console.log(`  sub_13px_text_instances: ${sub13Total}`)
console.log(`  allowlist budget: ${totalAllowed}`)
console.log(`  files with raw sizes: ${Object.keys(found).length}`)

if (violations.length) {
  console.error('\nFailures (new files or increased counts):')
  for (const v of violations.slice(0, 40)) {
    console.error(`  ${v.file}: ${v.count} (allowed ${v.allowed}) — ${v.reason}`)
  }
  if (violations.length > 40) console.error(`  … and ${violations.length - 40} more`)
  console.error('\nMigrate to text-body / text-caption / … or run --write-allowlist only when ratcheting down.')
  process.exit(1)
}

console.log('\nOK — no new/increased raw type-size violations.')
process.exit(0)
