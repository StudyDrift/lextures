#!/usr/bin/env node
/**
 * UX.2 FR-10 — design-system coverage ratchet.
 *
 * design-system-coverage =
 *   (DS interactive JSX tags) / (DS tags + raw interactive HTML tags)
 *
 * Metrics: design_system_coverage, raw_button_count, raw_dialog_count
 *
 * Usage:
 *   npm run ds:coverage
 *   node scripts/check-design-system-coverage.mjs --write-baseline
 *   node scripts/check-design-system-coverage.mjs --json
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const baselinePath = join(webRoot, 'design-system-coverage-baseline.json')
const rawAllowlistPath = join(webRoot, 'raw-interactive-allowlist.json')

/** Core library interactive components (JSX tags). */
const DS_TAGS = [
  'Button',
  'IconButton',
  'LinkButton',
  'ButtonGroup',
  'SplitButton',
  'Input',
  'Textarea',
  'Select',
  'Combobox',
  'Checkbox',
  'Radio',
  'RadioGroup',
  'Switch',
  'SegmentedControl',
  'DatePicker',
  'FileInput',
  'Dialog',
  'AlertDialog',
  'Sheet',
  'Drawer',
  'Popover',
  'Tooltip',
  'Menu',
  'ContextMenu',
  'Tabs',
  'Tab',
  'TabList',
  'Disclosure',
  'Pagination',
  'UiNavLink',
  'ConfirmDialog',
  'OverlaySurface',
]

const DS_TAG_RE = new RegExp(`<(${DS_TAGS.join('|')})\\b`, 'g')
const RAW_BUTTON_RE = /<button\b/g
const RAW_INPUT_RE = /<input\b/g
const RAW_SELECT_RE = /<select\b/g
const RAW_TEXTAREA_RE = /<textarea\b/g
const ROLE_DIALOG_RE = /role=["']dialog["']/g
const ROLE_MENU_RE = /role=["']menu["']/g
const ROLE_TABLIST_RE = /role=["']tablist["']/g
const TITLE_TOOLTIP_RE = /\btitle=["'][^"']+["']/g

const HARD_EXEMPT = [
  /src\/components\/ui\//,
  /src\/pages\/design\//,
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

const writeBaseline = process.argv.includes('--write-baseline')
const writeAllowlist = process.argv.includes('--write-allowlist')
const asJson = process.argv.includes('--json')

const files = walk(srcRoot, (n) => n.endsWith('.tsx'))

let dsCount = 0
let rawButton = 0
let rawInput = 0
let rawSelect = 0
let rawTextarea = 0
let roleDialog = 0
let roleMenu = 0
let roleTablist = 0
let titleTooltip = 0

/** @type {Record<string, { rawButton: number, rawInput: number, rawSelect: number, rawTextarea: number, roleDialog: number, roleMenu: number, roleTablist: number, titleTooltip: number }>} */
const perFile = {}

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  const text = readFileSync(file, 'utf8')
  const ds = countMatches(text, DS_TAG_RE)
  const rb = countMatches(text, RAW_BUTTON_RE)
  const ri = countMatches(text, RAW_INPUT_RE)
  const rs = countMatches(text, RAW_SELECT_RE)
  const rt = countMatches(text, RAW_TEXTAREA_RE)
  const rd = countMatches(text, ROLE_DIALOG_RE)
  const rm = countMatches(text, ROLE_MENU_RE)
  const rtl = countMatches(text, ROLE_TABLIST_RE)
  const tt = countMatches(text, TITLE_TOOLTIP_RE)

  dsCount += ds
  rawButton += rb
  rawInput += ri
  rawSelect += rs
  rawTextarea += rt
  roleDialog += rd
  roleMenu += rm
  roleTablist += rtl
  titleTooltip += tt

  if (rb + ri + rs + rt + rd + rm + rtl + tt > 0) {
    perFile[rel] = {
      rawButton: rb,
      rawInput: ri,
      rawSelect: rs,
      rawTextarea: rt,
      roleDialog: rd,
      roleMenu: rm,
      roleTablist: rtl,
      titleTooltip: tt,
    }
  }
}

const rawInteractive = rawButton + rawInput + rawSelect + rawTextarea
// Dialogs/menus/tabs hand-rolls count toward denominator as interactive patterns
const rawPatterns = roleDialog + roleMenu + roleTablist + titleTooltip
const total = dsCount + rawInteractive + rawPatterns
const coverage = total === 0 ? 1 : dsCount / total

const metrics = {
  design_system_coverage: Number(coverage.toFixed(6)),
  design_system_tags: dsCount,
  raw_button_count: rawButton,
  raw_input_count: rawInput,
  raw_select_count: rawSelect,
  raw_textarea_count: rawTextarea,
  raw_dialog_count: roleDialog,
  raw_menu_count: roleMenu,
  raw_tablist_count: roleTablist,
  title_tooltip_count: titleTooltip,
  raw_interactive_total: rawInteractive + rawPatterns,
  total_interactive: total,
}

if (writeAllowlist) {
  const sorted = Object.fromEntries(
    Object.entries(perFile).sort((a, b) => a[0].localeCompare(b[0])),
  )
  writeFileSync(
    rawAllowlistPath,
    JSON.stringify(
      {
        description:
          'UX.2 raw interactive allowlist (FR-11). Per-file counts may only decrease. Empty when migration complete (AC-2).',
        generatedAt: new Date().toISOString(),
        files: sorted,
      },
      null,
      2,
    ) + '\n',
  )
  console.log(`Wrote raw interactive allowlist with ${Object.keys(sorted).length} files.`)
}

if (writeBaseline) {
  writeFileSync(
    baselinePath,
    JSON.stringify(
      {
        description:
          'UX.2 design-system coverage baseline (FR-10). Coverage may only increase; raw counts may only decrease.',
        generatedAt: new Date().toISOString(),
        metrics,
      },
      null,
      2,
    ) + '\n',
  )
  console.log(
    `Wrote baseline coverage=${(metrics.design_system_coverage * 100).toFixed(2)}% ds=${dsCount} raw=${metrics.raw_interactive_total}`,
  )
  process.exit(0)
}

if (writeAllowlist && !process.argv.includes('--check')) {
  process.exit(0)
}

/** @type {{ metrics?: typeof metrics }} */
let baseline = { metrics: undefined }
if (existsSync(baselinePath)) {
  baseline = JSON.parse(readFileSync(baselinePath, 'utf8'))
}

/** @type {{ files?: Record<string, Record<string, number>> }} */
let allowlist = { files: {} }
if (existsSync(rawAllowlistPath)) {
  allowlist = JSON.parse(readFileSync(rawAllowlistPath, 'utf8'))
}

/** @type {string[]} */
const failures = []

if (baseline.metrics) {
  const b = baseline.metrics
  if (metrics.design_system_coverage + 1e-9 < b.design_system_coverage) {
    failures.push(
      `design_system_coverage decreased: ${(metrics.design_system_coverage * 100).toFixed(3)}% < baseline ${(b.design_system_coverage * 100).toFixed(3)}%`,
    )
  }
  for (const key of [
    'raw_button_count',
    'raw_input_count',
    'raw_select_count',
    'raw_textarea_count',
    'raw_dialog_count',
    'raw_menu_count',
    'raw_tablist_count',
    'title_tooltip_count',
  ]) {
    if (metrics[key] > b[key]) {
      failures.push(`${key} increased: ${metrics[key]} > baseline ${b[key]}`)
    }
  }
} else {
  failures.push('Missing design-system-coverage-baseline.json — run with --write-baseline')
}

const allowed = allowlist.files ?? {}
for (const [file, counts] of Object.entries(perFile)) {
  const a = allowed[file]
  if (!a) {
    failures.push(`New raw interactive usage in ${file} (not on allowlist)`)
    continue
  }
  for (const [k, n] of Object.entries(counts)) {
    const max = a[k] ?? 0
    if (n > max) {
      failures.push(`${file}: ${k} ${n} > allowlisted ${max}`)
    }
  }
}

// Ratchet: allowlisted files that are fully clean should be dropped eventually;
// files on allowlist with zero current violations are fine (stale entries ok until refresh).

if (asJson) {
  console.log(JSON.stringify({ metrics, failures }, null, 2))
} else {
  console.log('UX.2 design-system coverage')
  console.log(
    `  coverage: ${(metrics.design_system_coverage * 100).toFixed(2)}% (${dsCount} ds / ${total} total)`,
  )
  console.log(`  raw_button_count: ${rawButton}`)
  console.log(`  raw_dialog_count: ${roleDialog}`)
  console.log(`  raw_interactive_total: ${metrics.raw_interactive_total}`)
  if (baseline.metrics) {
    console.log(
      `  baseline coverage: ${(baseline.metrics.design_system_coverage * 100).toFixed(2)}%`,
    )
  }
}

if (failures.length) {
  console.error('\nFAIL:')
  for (const f of failures) console.error(`  - ${f}`)
  process.exit(1)
}

console.log('\nOK — coverage ratchet held.')
process.exit(0)
