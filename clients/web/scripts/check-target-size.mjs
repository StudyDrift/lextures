#!/usr/bin/env node
/**
 * UX.5 FR-3 — static target-size ratchet (WCAG 2.2 SC 2.5.8).
 *
 * Full rendered measurement of the top 40 routes is an e2e concern. This script
 * catches the common regression: fixed h-N / w-N utilities smaller than 24px on
 * interactive elements without a min-h-6 / min-w-6 (or larger) companion.
 *
 * Metrics: target_size_violations
 *
 * Usage:
 *   node scripts/check-target-size.mjs
 *   node scripts/check-target-size.mjs --json
 *   node scripts/check-target-size.mjs --write-baseline
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const baselinePath = join(webRoot, 'target-size-baseline.json')
const exceptionsPath = join(webRoot, 'target-size-exceptions.json')

/** Tailwind sizes below 24px (h-5=20, h-4=16, h-3=12, h-2=8). */
const SMALL_FIXED =
  /\b(?:h|w|size)-(?:[0-4]|5|\[(?:1[0-9]|2[0-3])px\])\b/g
/** Acceptable mins that satisfy 24px. */
const MIN_OK =
  /\bmin-(?:h|w)-(?:6|7|8|9|10|11|12|14|16|20|24|\[(?:2[4-9]|[3-9]\d)px\])\b/
/** Interactive host patterns in className-ish context. */
const INTERACTIVE_HINT =
  /<(?:button|Button|IconButton|a\b|Link\b|input\b|select\b|summary\b|label\b)/

const HARD_EXEMPT = [
  /src\/components\/ui\//, // library enforces size tokens
  /src\/pages\/design\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
  /src\/styles\//,
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

function isExempt(rel) {
  return HARD_EXEMPT.some((re) => re.test(rel.replace(/\\/g, '/')))
}

/**
 * Heuristic: lines that open an interactive element and set a fixed small size
 * without min-h-6 / min-w-6 on the same or following attribute chunk.
 */
function findSuspectLines(text, rel) {
  const lines = text.split('\n')
  /** @type {{ file: string; line: number; snippet: string }[]} */
  const hits = []
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!INTERACTIVE_HINT.test(line) && !/className=/.test(line)) continue
    // Look at a short window for class strings on multi-line JSX.
    const window = lines.slice(i, Math.min(lines.length, i + 6)).join(' ')
    if (!SMALL_FIXED.test(window)) continue
    // Reset lastIndex after global regex test
    SMALL_FIXED.lastIndex = 0
    if (MIN_OK.test(window)) continue
    // Icons inside buttons (class on SVG) are not targets — skip pure icon size lines.
    if (/<(?:svg|path|Grip|Chevron|Icon|Lucide)/.test(line) && !/<(?:button|Button|a\b)/.test(line)) {
      continue
    }
    // Only flag when the small size is on the interactive host, not a nested icon.
    if (
      /className=\{?[`'"][^`'"]*\b(?:h|w)-(?:[0-4]|5)\b/.test(window) ||
      /className=\{?`[^`]*\b(?:h|w)-(?:[0-4]|5)\b/.test(window)
    ) {
      // Prefer hosts that also look interactive.
      if (
        /<(?:button|Button|IconButton)\b/.test(window) ||
        /role=["']button["']/.test(window)
      ) {
        hits.push({
          file: rel,
          line: i + 1,
          snippet: line.trim().slice(0, 160),
        })
      }
    }
  }
  return hits
}

const asJson = process.argv.includes('--json')
const writeBaseline = process.argv.includes('--write-baseline')

const exceptions = existsSync(exceptionsPath)
  ? JSON.parse(readFileSync(exceptionsPath, 'utf8'))
  : { exceptions: [] }
const exceptionFiles = new Set(
  (exceptions.exceptions || []).map((e) => e.file?.replace(/\\/g, '/')).filter(Boolean),
)

const files = walk(srcRoot, (n) => n.endsWith('.tsx'))
/** @type {{ file: string; line: number; snippet: string }[]} */
let violations = []

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  if (exceptionFiles.has(rel)) continue
  const text = readFileSync(file, 'utf8')
  violations.push(...findSuspectLines(text, rel))
}

const baseline = existsSync(baselinePath)
  ? JSON.parse(readFileSync(baselinePath, 'utf8'))
  : { target_size_violations: Number.MAX_SAFE_INTEGER }

if (writeBaseline) {
  writeFileSync(
    baselinePath,
    JSON.stringify(
      {
        target_size_violations: violations.length,
        updatedAt: new Date().toISOString(),
      },
      null,
      2,
    ) + '\n',
  )
  console.log(`Wrote baseline target_size_violations=${violations.length}`)
  process.exit(0)
}

const maxAllowed = typeof baseline.target_size_violations === 'number'
  ? baseline.target_size_violations
  : Number.MAX_SAFE_INTEGER

const metric = {
  target_size_violations: violations.length,
  baseline: maxAllowed,
}

if (asJson) {
  console.log(JSON.stringify({ ...metric, violations: violations.slice(0, 50) }, null, 2))
} else {
  console.log(
    `target_size_violations=${metric.target_size_violations} (baseline max ${maxAllowed})`,
  )
  if (violations.length > maxAllowed) {
    console.log('New suspect small targets (first 20):')
    for (const v of violations.slice(0, 20)) {
      console.log(`  ${v.file}:${v.line}  ${v.snippet}`)
    }
  }
}

if (violations.length > maxAllowed) {
  console.error(
    `FAIL: target_size_violations rose from ${maxAllowed} to ${violations.length}. Enlarge controls to ≥24×24 or add a justified exception.`,
  )
  process.exit(1)
}
