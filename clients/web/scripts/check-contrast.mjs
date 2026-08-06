#!/usr/bin/env node
/**
 * UX.1 — Semantic token contrast validation (WCAG 2.1 AA).
 *
 * Resolves semantic (fg, bg) pairings for every theme from the token value map
 * and fails if any pair is below AA (4.5:1 text, 3:1 UI). Also supports
 * --fixture for deliberate-fail tests (AC-2).
 *
 * Metrics: contrast_pairs_checked
 *
 * Usage:
 *   npm run contrast:check
 *   node scripts/check-contrast.mjs --fixture=failing
 */
import { readFileSync, existsSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')
const tokensPath = resolve(root, '../packages/tokens/tokens.json')
const legacyConfigPath = resolve(root, 'contrast-config.json')

// ── WCAG helpers ─────────────────────────────────────────────────────────────

function hexToRgb(hex) {
  const h = hex.replace('#', '')
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
}

function linearize(c) {
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

function luminance(hex) {
  const [r, g, b] = hexToRgb(hex).map((v) => linearize(v / 255))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function contrastRatio(hex1, hex2) {
  const l1 = luminance(hex1)
  const l2 = luminance(hex2)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

// ── Semantic pair definitions ────────────────────────────────────────────────

const AA_TEXT = 4.5
const AA_UI = 3.0

const SEMANTIC_PAIRS = [
  { fg: 'fg.default', bg: 'surface.base', minRatio: AA_TEXT, usage: 'body on page' },
  { fg: 'fg.default', bg: 'surface.raised', minRatio: AA_TEXT, usage: 'body on cards' },
  { fg: 'fg.muted', bg: 'surface.base', minRatio: AA_TEXT, usage: 'muted on page' },
  { fg: 'fg.muted', bg: 'surface.raised', minRatio: AA_TEXT, usage: 'muted on cards' },
  { fg: 'fg.subtle', bg: 'surface.raised', minRatio: AA_TEXT, usage: 'placeholder' },
  { fg: 'fg.onAccent', bg: 'accent.solid', minRatio: AA_TEXT, usage: 'primary button' },
  { fg: 'fg.default', bg: 'surface.sunken', minRatio: AA_TEXT, usage: 'text on sunken' },
  { fg: 'status.info.fg', bg: 'status.info.surface', minRatio: AA_TEXT, usage: 'info' },
  { fg: 'status.success.fg', bg: 'status.success.surface', minRatio: AA_TEXT, usage: 'success' },
  { fg: 'status.warning.fg', bg: 'status.warning.surface', minRatio: AA_TEXT, usage: 'warning' },
  { fg: 'status.danger.fg', bg: 'status.danger.surface', minRatio: AA_TEXT, usage: 'danger' },
  { fg: 'status.accent.fg', bg: 'status.accent.surface', minRatio: AA_TEXT, usage: 'accent status' },
  // Decorative hairline borders are not SC 1.4.11 controls; use border.strong / focus for UI.
  { fg: 'border.strong', bg: 'surface.raised', minRatio: AA_UI, usage: 'strong UI border' },
  { fg: 'focus.ring', bg: 'surface.raised', minRatio: AA_UI, usage: 'focus ring' },
]

const THEMES = ['light', 'dark', 'high-contrast-light', 'high-contrast-dark']

function resolvePath(obj, path) {
  const parts = path.split('.')
  let cur = obj
  for (const p of parts) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = cur[p]
  }
  if (cur && typeof cur === 'object' && cur.$value != null) return cur.$value
  return typeof cur === 'string' ? cur : undefined
}

function loadTokenThemes() {
  if (!existsSync(tokensPath)) {
    console.error(`Missing ${tokensPath}. Run tokens export first.`)
    process.exit(2)
  }
  const doc = JSON.parse(readFileSync(tokensPath, 'utf8'))
  return doc.themes ?? doc
}

// ── Main ─────────────────────────────────────────────────────────────────────

const args = process.argv.slice(2)
const fixtureFail = args.some((a) => a === '--fixture=failing' || a === '--fixture')

console.log('UX.1 semantic token contrast validation (WCAG 2.1 AA)\n')

const themes = loadTokenThemes()
let failures = 0
let checked = 0

for (const theme of THEMES) {
  const map = themes[theme]
  if (!map) {
    console.error(`  ✗ Missing theme "${theme}" in tokens.json`)
    failures++
    continue
  }
  console.log(`${theme}:`)
  for (const pair of SEMANTIC_PAIRS) {
    let fgHex = resolvePath(map, pair.fg)
    let bgHex = resolvePath(map, pair.bg)
    if (fixtureFail && pair.fg === 'fg.subtle' && theme === 'light') {
      // Deliberate fail: near-white on white
      fgHex = '#f0f0f0'
      bgHex = '#ffffff'
    }
    if (!fgHex || !bgHex) {
      console.error(`  ✗ [${theme}] unresolved ${pair.fg} on ${pair.bg}`)
      failures++
      continue
    }
    // Skip non-hex (oklch) — tokens.json stores resolved hex for validation
    if (!/^#[0-9a-f]{6}$/i.test(fgHex) || !/^#[0-9a-f]{6}$/i.test(bgHex)) {
      console.error(`  ✗ [${theme}] non-hex value for ${pair.fg}/${pair.bg}: ${fgHex} / ${bgHex}`)
      failures++
      continue
    }
    const ratio = contrastRatio(fgHex, bgHex)
    checked++
    const pass = ratio >= pair.minRatio
    const label = `[${theme}] ${pair.fg} on ${pair.bg}`
    if (pass) {
      console.log(`  ✓ ${label} — ${ratio.toFixed(2)}:1 ≥ ${pair.minRatio}:1`)
    } else {
      console.error(
        `  ✗ ${label} — ${ratio.toFixed(2)}:1 < ${pair.minRatio}:1 [FAILS] — ${pair.usage}`,
      )
      failures++
    }
  }
  console.log('')
}

// Legacy allowlist still checked if present (deprecation cross-check)
if (existsSync(legacyConfigPath)) {
  try {
    const legacy = JSON.parse(readFileSync(legacyConfigPath, 'utf8'))
    let legacyFail = 0
    for (const theme of ['light', 'dark']) {
      for (const pair of legacy.pairs?.[theme] ?? []) {
        const fg = legacy.tokens[pair.foreground]
        const bg = legacy.tokens[pair.background]
        if (!fg || !bg) continue
        const ratio = contrastRatio(fg, bg)
        if (ratio < (pair.minRatio ?? 4.5)) legacyFail++
      }
    }
    if (legacyFail > 0) {
      console.warn(`Legacy contrast-config.json: ${legacyFail} pair(s) still fail (informational).`)
    }
  } catch {
    /* ignore */
  }
}

console.log(`contrast_pairs_checked=${checked}`)
const status =
  failures === 0
    ? `✓ All ${checked} pairs pass`
    : `✗ ${failures}/${checked} pair(s) failed`
console.log(status)

if (failures > 0) {
  console.error(
    '\nFix semantic token values in clients/web/src/styles/tokens/ and regenerate tokens.json.\n' +
      'Reference: https://www.w3.org/TR/WCAG21/#contrast-minimum',
  )
  process.exit(1)
}
