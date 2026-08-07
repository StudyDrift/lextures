#!/usr/bin/env node
/**
 * UX.4 FR-13 — ARIA widget contract coverage ratchet.
 *
 * For every declared ARIA widget role (`tablist`, `menu`) and every
 * `aria-modal` overlay outside `components/ui/`, decide whether the file
 * (or a DS import it delegates to) implements the required keyboard / focus
 * contract.
 *
 * Metrics (CI):
 *   aria_contract_coverage        — satisfied / total widgets (0–1)
 *   aria_modal_without_trap       — modals missing focus trap / DS dialog
 *   role_menu_without_keyboard    — menus missing arrow-key contract
 *   role_tablist_without_keyboard — tablists missing arrow-key contract
 *   title_attribute_tooltips      — `title=` pseudo-tooltips in feature TSX
 *
 * Coverage may only increase; the four defect counts may only decrease.
 *
 * Usage:
 *   npm run a11y:contracts
 *   node scripts/check-aria-contracts.mjs --write-baseline
 *   node scripts/check-aria-contracts.mjs --json
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const baselinePath = join(webRoot, 'aria-contract-baseline.json')

const HARD_EXEMPT = [
  /src\/components\/ui\//,
  /src\/pages\/design\//,
  /src\/__tests__\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
]

/** Imports that fully implement the matching contract. */
const DS_DIALOG_IMPORT =
  /from\s+['"][^'"]*(?:components\/ui(?:\/(?:dialog|alert-dialog|sheet|overlay-surface|index))?|\/ui(?:\/(?:dialog|alert-dialog|sheet))?|lib\/a11y(?:\/(?:focus-trap|index))?)['"]/
const DS_MENU_IMPORT =
  /from\s+['"][^'"]*(?:components\/ui(?:\/(?:menu|context-menu|index))?|\/ui(?:\/menu)?)['"]/
const DS_TABS_IMPORT =
  /from\s+['"][^'"]*(?:components\/ui(?:\/(?:tabs|index))?|\/ui(?:\/tabs)?)['"]/
const FOCUS_TRAP_USE =
  /\b(?:createFocusTrap|useInertBackground|pushModalOverlay|handleMenuKeyDown|handleTablistKeyDown|focusFirstMenuitem)\b/
const DIALOG_COMPONENT =
  /<(?:Dialog|AlertDialog|Sheet|Drawer|ConfirmDialog|OverlaySurface)\b/
const MENU_COMPONENT = /<(?:Menu|ContextMenu)\b/
const TABS_COMPONENT = /<(?:Tabs|TabList)\b/

const TABLIST_KEYS =
  /Arrow(?:Left|Right|Up|Down)|Home|End|handleTablistKeyDown/
const MENU_KEYS =
  /Arrow(?:Up|Down)|Home|End|typeahead|textValue|handleMenuKeyDown|focusFirstMenuitem/

const ROLE_TABLIST = /role\s*=\s*\{?\s*["']tablist["']\s*\}?/g
const ROLE_MENU = /role\s*=\s*\{?\s*["']menu["']\s*\}?/g
const ARIA_MODAL = /aria-modal\s*=\s*\{?\s*(?:["']true["']|!?\w+)/g
/** title= used as tooltip (not React prop `title` on page shells with complex JSX). */
const TITLE_TOOLTIP = /\btitle\s*=\s*["'][^"']+["']/g
/** Exclude known non-tooltip title props: page chrome / LmsPage / Section / OnboardingShell. */
const TITLE_FALSE_POSITIVE =
  /<(?:LmsPage|Section|OnboardingShell|PageHeader|Helmet|title)\b[^>]*\btitle\s*=/

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
  const flags = re.flags.includes('g') ? re.flags : re.flags + 'g'
  const r = new RegExp(re.source, flags)
  const m = text.match(r)
  return m ? m.length : 0
}

function isExempt(rel) {
  return HARD_EXEMPT.some((re) => re.test(rel.replace(/\\/g, '/')))
}

/**
 * Count title= tooltips, ignoring LmsPage/Section-style title props on the same line
 * when the tag is a known non-tooltip host. Heuristic: strip matches that sit on a
 * line containing those hosts.
 */
function countTitleTooltips(text) {
  const lines = text.split('\n')
  let n = 0
  for (const line of lines) {
    if (TITLE_FALSE_POSITIVE.test(line)) continue
    // Page-level props often: title={t(...)} — only quoted string titles count as tooltips.
    const m = line.match(TITLE_TOOLTIP)
    if (m) n += m.length
  }
  return n
}

function analyzeFile(text) {
  const tablists = countMatches(text, ROLE_TABLIST)
  const menus = countMatches(text, ROLE_MENU)
  const modals = countMatches(text, ARIA_MODAL)
  const titles = countTitleTooltips(text)

  const usesDsDialog =
    DS_DIALOG_IMPORT.test(text) || FOCUS_TRAP_USE.test(text) || DIALOG_COMPONENT.test(text)
  const usesDsMenu = DS_MENU_IMPORT.test(text) || MENU_COMPONENT.test(text)
  const usesDsTabs = DS_TABS_IMPORT.test(text) || TABS_COMPONENT.test(text)
  const hasTablistKeys = TABLIST_KEYS.test(text)
  const hasMenuKeys = MENU_KEYS.test(text)

  const tablistOk = tablists === 0 || usesDsTabs || hasTablistKeys
  const menuOk = menus === 0 || usesDsMenu || hasMenuKeys
  const modalOk = modals === 0 || usesDsDialog

  const widgets = tablists + menus + modals
  let satisfied = 0
  if (tablists > 0 && tablistOk) satisfied += tablists
  if (menus > 0 && menuOk) satisfied += menus
  if (modals > 0 && modalOk) satisfied += modals

  return {
    tablists,
    menus,
    modals,
    titles,
    tablistOk,
    menuOk,
    modalOk,
    widgets,
    satisfied,
    tablistWithoutKeyboard: tablists > 0 && !tablistOk ? tablists : 0,
    menuWithoutKeyboard: menus > 0 && !menuOk ? menus : 0,
    modalWithoutTrap: modals > 0 && !modalOk ? modals : 0,
  }
}

const writeBaseline = process.argv.includes('--write-baseline')
const asJson = process.argv.includes('--json')

const files = walk(srcRoot, (n) => n.endsWith('.tsx') || n.endsWith('.ts'))

/** @type {ReturnType<typeof analyzeFile>[]} */
const results = []
/** @type {{ file: string; kind: string; count: number }[]} */
const defects = []

let totalWidgets = 0
let totalSatisfied = 0
let ariaModalWithoutTrap = 0
let roleMenuWithoutKeyboard = 0
let roleTablistWithoutKeyboard = 0
let titleAttributeTooltips = 0

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  // Only scan TSX for roles (TS libs rarely declare ARIA roles).
  if (!rel.endsWith('.tsx')) continue
  const text = readFileSync(file, 'utf8')
  const a = analyzeFile(text)
  results.push(a)
  totalWidgets += a.widgets
  totalSatisfied += a.satisfied
  ariaModalWithoutTrap += a.modalWithoutTrap
  roleMenuWithoutKeyboard += a.menuWithoutKeyboard
  roleTablistWithoutKeyboard += a.tablistWithoutKeyboard
  titleAttributeTooltips += a.titles

  if (a.modalWithoutTrap) {
    defects.push({ file: rel, kind: 'aria_modal_without_trap', count: a.modalWithoutTrap })
  }
  if (a.menuWithoutKeyboard) {
    defects.push({ file: rel, kind: 'role_menu_without_keyboard', count: a.menuWithoutKeyboard })
  }
  if (a.tablistWithoutKeyboard) {
    defects.push({
      file: rel,
      kind: 'role_tablist_without_keyboard',
      count: a.tablistWithoutKeyboard,
    })
  }
}

const coverage = totalWidgets === 0 ? 1 : totalSatisfied / totalWidgets

const metrics = {
  aria_contract_coverage: Number(coverage.toFixed(6)),
  aria_contract_widgets: totalWidgets,
  aria_contract_satisfied: totalSatisfied,
  aria_modal_without_trap: ariaModalWithoutTrap,
  role_menu_without_keyboard: roleMenuWithoutKeyboard,
  role_tablist_without_keyboard: roleTablistWithoutKeyboard,
  title_attribute_tooltips: titleAttributeTooltips,
}

if (writeBaseline) {
  writeFileSync(
    baselinePath,
    JSON.stringify(
      {
        description:
          'UX.4 ARIA contract baseline. Coverage may only increase; defect counts may only decrease.',
        generatedAt: new Date().toISOString(),
        metrics,
      },
      null,
      2,
    ) + '\n',
  )
  console.log('Wrote aria-contract-baseline.json')
  console.log(JSON.stringify(metrics, null, 2))
  process.exit(0)
}

/** @type {{ metrics?: typeof metrics }} */
let baseline = { metrics: undefined }
if (existsSync(baselinePath)) {
  baseline = JSON.parse(readFileSync(baselinePath, 'utf8'))
}

const base = baseline.metrics
/** @type {string[]} */
const failures = []

if (base) {
  if (metrics.aria_contract_coverage + 1e-9 < base.aria_contract_coverage) {
    failures.push(
      `aria_contract_coverage decreased: ${metrics.aria_contract_coverage} < ${base.aria_contract_coverage}`,
    )
  }
  for (const key of [
    'aria_modal_without_trap',
    'role_menu_without_keyboard',
    'role_tablist_without_keyboard',
    'title_attribute_tooltips',
  ]) {
    if (metrics[key] > base[key]) {
      failures.push(`${key} increased: ${metrics[key]} > ${base[key]}`)
    }
  }
} else {
  failures.push('Missing aria-contract-baseline.json — run with --write-baseline')
}

if (asJson) {
  console.log(JSON.stringify({ metrics, baseline: base ?? null, failures, defects }, null, 2))
} else {
  console.log('UX.4 ARIA contract check')
  console.log(
    `  aria_contract_coverage: ${(metrics.aria_contract_coverage * 100).toFixed(2)}% (${metrics.aria_contract_satisfied}/${metrics.aria_contract_widgets})`,
  )
  console.log(`  aria_modal_without_trap: ${metrics.aria_modal_without_trap}`)
  console.log(`  role_menu_without_keyboard: ${metrics.role_menu_without_keyboard}`)
  console.log(`  role_tablist_without_keyboard: ${metrics.role_tablist_without_keyboard}`)
  console.log(`  title_attribute_tooltips: ${metrics.title_attribute_tooltips}`)
  if (failures.length) {
    console.error('\nRatchet failures:')
    for (const f of failures) console.error(`  - ${f}`)
    if (defects.length) {
      console.error('\nSample defects (first 25):')
      for (const d of defects.slice(0, 25)) {
        console.error(`  - ${d.file}: ${d.kind} ×${d.count}`)
      }
    }
  } else {
    console.log('  OK (no regression vs baseline)')
  }
}

process.exit(failures.length ? 1 : 0)
