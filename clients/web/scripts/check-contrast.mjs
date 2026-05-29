#!/usr/bin/env node
/**
 * Design-token contrast validation — WCAG 2.1 AA (plan 12.3).
 *
 * Reads contrast-config.json and validates every foreground/background
 * token pair against the required minimum contrast ratio. Exits 1 if any
 * pair falls below its threshold. Intended for CI; also runnable locally
 * via `npm run contrast:check`.
 */
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const configPath = resolve(__dirname, '..', 'contrast-config.json')

// ── WCAG relative-luminance helpers ──────────────────────────────────────────

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

// ── Validation ────────────────────────────────────────────────────────────────

const config = JSON.parse(readFileSync(configPath, 'utf8'))
const { tokens, pairs } = config

let failures = 0
let checked = 0

function check(theme, pair) {
  const fgHex = tokens[pair.foreground]
  const bgHex = tokens[pair.background]

  if (!fgHex) {
    console.error(`  ✗ [${theme}] Unknown token: "${pair.foreground}"`)
    failures++
    return
  }
  if (!bgHex) {
    console.error(`  ✗ [${theme}] Unknown token: "${pair.background}"`)
    failures++
    return
  }

  const ratio = contrastRatio(fgHex, bgHex)
  const threshold = pair.minRatio ?? 4.5
  const pass = ratio >= threshold
  checked++

  const label = `[${theme}] ${pair.foreground} on ${pair.background}`
  const ratioStr = ratio.toFixed(2) + ':1'
  const thresholdStr = threshold.toFixed(1) + ':1 (AA)'

  if (pass) {
    console.log(`  ✓ ${label} — ${ratioStr} ≥ ${thresholdStr}`)
  } else {
    console.error(`  ✗ ${label} — ${ratioStr} < ${thresholdStr} [FAILS] — ${pair.usage}`)
    failures++
  }
}

console.log('Design-token contrast validation (WCAG 2.1 AA)\n')

if (pairs.light?.length) {
  console.log('Light theme:')
  for (const pair of pairs.light) check('light', pair)
}

if (pairs.dark?.length) {
  console.log('\nDark theme:')
  for (const pair of pairs.dark) check('dark', pair)
}

const status = failures === 0 ? `✓ All ${checked} pairs pass` : `✗ ${failures}/${checked} pair(s) failed`
console.log(`\n${status}`)

if (failures > 0) {
  console.error(
    '\nFix failing pairs in clients/web/src/index.css and update contrast-config.json.\n' +
      'Reference: https://www.w3.org/TR/WCAG21/#contrast-minimum',
  )
  process.exit(1)
}
