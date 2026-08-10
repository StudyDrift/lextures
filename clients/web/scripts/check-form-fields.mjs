#!/usr/bin/env node
/**
 * UX.6 — form control ratchet.
 *
 * Metrics (observability):
 *   - inputs_outside_field_component  (raw <input>/<select>/<textarea> outside ui/)
 *   - fields_without_label            (placeholder-as-label heuristic)
 *
 * Starts as a **baseline ratchet** (counts may only decrease) so migration can
 * proceed directory-by-directory. Use --write-baseline after intentional
 * migration batches. Flip to zero-tolerance when allowlists are empty (AC-10).
 *
 * Usage:
 *   node scripts/check-form-fields.mjs
 *   node scripts/check-form-fields.mjs --json
 *   node scripts/check-form-fields.mjs --write-baseline
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const baselinePath = join(webRoot, 'form-fields-baseline.json')

const RAW_CONTROL_RE = /<(input|select|textarea)\b/g
const PLACEHOLDER_RE = /\bplaceholder\s*=\s*(?:\{|"|')/
/** Library / design / tests exempt from bare-control rule. */
const HARD_EXEMPT = [
  /src\/components\/ui\//,
  /src\/pages\/design\//,
  /src\/__tests__\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
]

const writeBaseline = process.argv.includes('--write-baseline')
const asJson = process.argv.includes('--json')

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

function countMatches(text, re) {
  const m = text.match(re)
  return m ? m.length : 0
}

/**
 * Heuristic: control has placeholder= and neither Field / htmlFor label / aria-label
 * appears in a short window around the tag.
 */
function countPlaceholderAsLabel(text) {
  const lines = text.split('\n')
  let hits = 0
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!RAW_CONTROL_RE.test(line) && !PLACEHOLDER_RE.test(line)) continue
    RAW_CONTROL_RE.lastIndex = 0
    const window = lines.slice(Math.max(0, i - 8), Math.min(lines.length, i + 6)).join('\n')
    if (!PLACEHOLDER_RE.test(window)) continue
    if (!/<(input|select|textarea)\b/.test(window)) continue
    // Accept if labelled
    if (
      /\bhtmlFor\b/.test(window) ||
      /<Field\b/.test(window) ||
      /aria-label\s*=/.test(window) ||
      /aria-labelledby\s*=/.test(window) ||
      /<label\b/.test(window)
    ) {
      continue
    }
    // hidden / type=file often use visual labels via wrapping label
    if (/type\s*=\s*["'](?:hidden|file)["']/.test(window)) continue
    hits++
  }
  return hits
}

const files = walk(srcRoot, (n) => n.endsWith('.tsx') || n.endsWith('.ts'))
let inputsOutside = 0
let placeholderAsLabel = 0
/** @type {Record<string, number>} */
const byFile = {}

for (const abs of files) {
  const rel = relative(webRoot, abs).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  if (!rel.endsWith('.tsx')) continue
  const text = readFileSync(abs, 'utf8')
  const raw = countMatches(text, RAW_CONTROL_RE)
  const ph = countPlaceholderAsLabel(text)
  if (raw > 0 || ph > 0) {
    byFile[rel] = raw
    inputsOutside += raw
    placeholderAsLabel += ph
  }
}

const metrics = {
  inputs_outside_field_component: inputsOutside,
  fields_without_label: placeholderAsLabel,
  files_with_raw_controls: Object.keys(byFile).length,
}

if (writeBaseline) {
  writeFileSync(
    baselinePath,
    JSON.stringify(
      {
        version: 1,
        updated: new Date().toISOString().slice(0, 10),
        metrics,
        note: 'UX.6 ratchet — counts must not increase. Migrate forms to Field + Input.',
      },
      null,
      2,
    ) + '\n',
  )
  console.log('Wrote', relative(webRoot, baselinePath))
  console.log(JSON.stringify(metrics, null, 2))
  process.exit(0)
}

if (!existsSync(baselinePath)) {
  console.error('Missing form-fields-baseline.json — run with --write-baseline first.')
  process.exit(1)
}

const baseline = JSON.parse(readFileSync(baselinePath, 'utf8'))
const base = baseline.metrics || baseline
const failures = []

for (const key of ['inputs_outside_field_component', 'fields_without_label']) {
  const cur = metrics[key]
  const lim = base[key]
  if (typeof lim !== 'number') {
    failures.push(`baseline missing ${key}`)
    continue
  }
  if (cur > lim) {
    failures.push(`${key}: ${cur} > baseline ${lim} (regression)`)
  }
}

if (asJson) {
  console.log(JSON.stringify({ metrics, baseline: base, failures }, null, 2))
} else {
  console.log('UX.6 form-fields check')
  console.log(
    `  inputs_outside_field_component: ${metrics.inputs_outside_field_component} (baseline ${base.inputs_outside_field_component})`,
  )
  console.log(
    `  fields_without_label: ${metrics.fields_without_label} (baseline ${base.fields_without_label})`,
  )
  console.log(`  files_with_raw_controls: ${metrics.files_with_raw_controls}`)
}

if (failures.length) {
  console.error('\nFAIL:')
  for (const f of failures) console.error('  -', f)
  console.error('See docs/runbooks/form-lint-failed.md')
  process.exit(1)
}

console.log('OK — form field metrics within baseline.')
process.exit(0)
