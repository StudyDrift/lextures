#!/usr/bin/env node
/**
 * UX.1 — lextures/no-raw-palette (FR-8).
 *
 * Fails on Tailwind palette literals and arbitrary hex in src TSX files
 * unless the file is allowlisted. Allowlist counts can only ratchet down.
 *
 * Metrics: token_purity_violations
 *
 * Usage:
 *   npm run tokens:purity
 *   node scripts/check-raw-palette.mjs --write-allowlist   # bootstrap / refresh
 *   node scripts/check-raw-palette.mjs --json
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const allowlistPath = join(webRoot, 'raw-palette-allowlist.json')

const PALETTE =
  /\b(?:bg|text|border|ring|outline|fill|stroke|from|via|to|divide|decoration|caret|accent|shadow)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-(?:50|100|200|300|400|500|600|700|800|900|950)\b/g

const ARBITRARY_HEX =
  /\b(?:bg|text|border|ring|outline|fill|stroke)-\[#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\]/g

const BARE_HEX_IN_CLASS =
  /(?:className|class)\s*=\s*(?:\{`|"|'|`)[^"'`]*#[0-9a-fA-F]{3,8}/g

/** Files exempt from purity (token gallery, tests, design specimens). */
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

function countViolations(text) {
  let n = 0
  const m1 = text.match(PALETTE)
  if (m1) n += m1.length
  const m2 = text.match(ARBITRARY_HEX)
  if (m2) n += m2.length
  return n
}

function isExempt(rel) {
  return HARD_EXEMPT.some((re) => re.test(rel.replace(/\\/g, '/')))
}

const writeAllowlist = process.argv.includes('--write-allowlist')
const asJson = process.argv.includes('--json')
const strict = process.argv.includes('--strict') // fail even on allowlisted files that exceed

const files = walk(srcRoot, (n) => n.endsWith('.tsx'))
/** @type {Record<string, number>} */
const found = {}

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  const text = readFileSync(file, 'utf8')
  const n = countViolations(text)
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
          'UX.1 raw-palette allowlist. Counts may only decrease. Empty when migration complete (AC-1).',
        generatedAt: new Date().toISOString(),
        files: sorted,
      },
      null,
      2,
    ) + '\n',
  )
  console.log(`Wrote allowlist with ${Object.keys(sorted).length} files.`)
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

for (const [file, count] of Object.entries(found)) {
  const max = allowed[file]
  if (max == null) {
    violations.push({ file, count, allowed: 0, reason: 'not allowlisted' })
  } else if (count > max) {
    ratchetUps.push(file)
    violations.push({ file, count, allowed: max, reason: 'exceeds allowlist (ratchet up blocked)' })
  }
}

// Detect allowlist entries that increased vs previous (when --strict-allowlist-shape)
let totalViolations = 0
for (const v of Object.values(found)) totalViolations += v

if (asJson) {
  console.log(
    JSON.stringify({
      token_purity_violations: violations.reduce((s, v) => s + v.count, 0),
      files_with_violations: violations.length,
      total_raw_literals: totalViolations,
      allowlisted_files: Object.keys(allowed).length,
      details: violations,
    }),
  )
} else {
  console.log('UX.1 raw palette purity (lextures/no-raw-palette)\n')
  if (violations.length === 0) {
    console.log(`✓ No purity violations (${totalViolations} raw literals within allowlist).`)
  } else {
    for (const v of violations.slice(0, 50)) {
      console.error(`  ✗ ${v.file}: ${v.count} (allowed ${v.allowed}) — ${v.reason}`)
    }
    if (violations.length > 50) {
      console.error(`  … and ${violations.length - 50} more files`)
    }
  }
  console.log(`\ntoken_purity_violations=${violations.reduce((s, v) => s + v.count, 0)}`)
  console.log(`raw_literals_remaining=${totalViolations}`)
  console.log(`allowlisted_files=${Object.keys(allowed).length}`)
}

// Exit 1 only when new files or ratchet-ups (migration mode).
// When allowlist is empty and any remain → fail (AC-1 end state).
const allowlistEmpty = Object.keys(allowed).length === 0
if (allowlistEmpty && totalViolations > 0) {
  console.error('\nAllowlist is empty but raw palette literals remain (AC-1).')
  process.exit(1)
}
if (violations.length > 0) {
  console.error(
    '\nUse semantic tokens (bg-surface-raised, text-fg-muted, …). See docs/design-tokens.md.',
  )
  process.exit(1)
}
if (strict && totalViolations > 0) {
  process.exit(1)
}
