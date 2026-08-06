#!/usr/bin/env node
/**
 * UX.1 — Idempotent codemod: dual light/dark palette pairs → semantic tokens.
 *
 * Maps the most common className patterns to semantic utilities.
 * Emits unmapped report to stdout (and optionally --report=path).
 *
 * Usage:
 *   node scripts/codemod-semantic-tokens.mjs --dry-run
 *   node scripts/codemod-semantic-tokens.mjs --dir=src/components/layout
 *   node scripts/codemod-semantic-tokens.mjs --write
 */
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')

const write = process.argv.includes('--write')
const dryRun = !write || process.argv.includes('--dry-run')
const dirArg = process.argv.find((a) => a.startsWith('--dir='))
const targetDir = dirArg
  ? resolve(webRoot, dirArg.slice('--dir='.length))
  : join(webRoot, 'src')

/**
 * Ordered replacements: longer / more specific first.
 * Each entry: [pattern (global), replacement]
 */
const REPLACEMENTS = [
  // Text dual pairs
  [/text-slate-900\s+dark:text-neutral-50\b/g, 'text-fg-default'],
  [/text-slate-900\s+dark:text-neutral-100\b/g, 'text-fg-default'],
  [/text-slate-800\s+dark:text-neutral-100\b/g, 'text-fg-default'],
  [/text-slate-800\s+dark:text-neutral-200\b/g, 'text-fg-default'],
  [/text-slate-700\s+dark:text-neutral-200\b/g, 'text-fg-default'],
  [/text-slate-700\s+dark:text-neutral-300\b/g, 'text-fg-muted'],
  [/text-slate-600\s+dark:text-neutral-300\b/g, 'text-fg-muted'],
  [/text-slate-600\s+dark:text-neutral-400\b/g, 'text-fg-muted'],
  [/text-slate-500\s+dark:text-neutral-400\b/g, 'text-fg-muted'],
  [/text-slate-500\s+dark:text-neutral-500\b/g, 'text-fg-subtle'],
  [/text-slate-400\s+dark:text-neutral-500\b/g, 'text-fg-subtle'],
  [/text-slate-400\s+dark:text-neutral-400\b/g, 'text-fg-subtle'],

  // Background dual pairs
  [/bg-white\s+dark:bg-neutral-900\b/g, 'bg-surface-raised'],
  [/bg-white\s+dark:bg-neutral-950\b/g, 'bg-surface-raised'],
  [/bg-white\s+dark:bg-neutral-800\b/g, 'bg-surface-raised'],
  [/bg-slate-50\s+dark:bg-neutral-950\b/g, 'bg-surface-base'],
  [/bg-slate-50\s+dark:bg-neutral-900\b/g, 'bg-surface-base'],
  [/bg-slate-100\s+dark:bg-neutral-800\b/g, 'bg-surface-sunken'],
  [/bg-slate-100\s+dark:bg-neutral-900\b/g, 'bg-surface-sunken'],
  [/bg-slate-50\s+dark:bg-neutral-800\b/g, 'bg-surface-sunken'],

  // Border dual pairs
  [/border-slate-200\s+dark:border-neutral-700\b/g, 'border-border-default'],
  [/border-slate-200\s+dark:border-neutral-600\b/g, 'border-border-default'],
  [/border-slate-100\s+dark:border-neutral-800\b/g, 'border-border-subtle'],
  [/border-slate-300\s+dark:border-neutral-600\b/g, 'border-border-strong'],
  [/border-slate-300\s+dark:border-neutral-700\b/g, 'border-border-strong'],

  // Single-token defaults (only when no dark: sibling remains on same class string — applied carefully)
  // Standalone common text (after dual pairs removed)
  [/(?<![\w-])text-slate-900(?![\w-])/g, 'text-fg-default'],
  [/(?<![\w-])text-slate-800(?![\w-])/g, 'text-fg-default'],
  [/(?<![\w-])text-slate-700(?![\w-])/g, 'text-fg-muted'],
  [/(?<![\w-])text-slate-600(?![\w-])/g, 'text-fg-muted'],
  [/(?<![\w-])text-slate-500(?![\w-])/g, 'text-fg-muted'],
  [/(?<![\w-])text-slate-400(?![\w-])/g, 'text-fg-subtle'],
  [/(?<![\w-])text-neutral-100(?![\w-])/g, 'text-fg-default'],
  [/(?<![\w-])text-neutral-200(?![\w-])/g, 'text-fg-default'],
  [/(?<![\w-])text-neutral-300(?![\w-])/g, 'text-fg-muted'],
  [/(?<![\w-])text-neutral-400(?![\w-])/g, 'text-fg-muted'],

  [/(?<![\w-])bg-white(?![\w/-])/g, 'bg-surface-raised'],
  [/(?<![\w-])bg-slate-50(?![\w/-])/g, 'bg-surface-base'],
  [/(?<![\w-])bg-slate-100(?![\w/-])/g, 'bg-surface-sunken'],
  [/(?<![\w-])bg-neutral-900(?![\w/-])/g, 'bg-surface-raised'],
  [/(?<![\w-])bg-neutral-950(?![\w/-])/g, 'bg-surface-base'],
  [/(?<![\w-])bg-neutral-800(?![\w/-])/g, 'bg-surface-overlay'],

  [/(?<![\w-])border-slate-200(?![\w/-])/g, 'border-border-default'],
  [/(?<![\w-])border-slate-100(?![\w/-])/g, 'border-border-subtle'],
  [/(?<![\w-])border-slate-300(?![\w/-])/g, 'border-border-strong'],
  [/(?<![\w-])border-neutral-700(?![\w/-])/g, 'border-border-default'],
  [/(?<![\w-])border-neutral-600(?![\w/-])/g, 'border-border-default'],
  [/(?<![\w-])border-neutral-800(?![\w/-])/g, 'border-border-subtle'],

  // Status colours (common)
  [/text-red-600\s+dark:text-red-400\b/g, 'text-danger-fg'],
  [/text-red-700\s+dark:text-red-300\b/g, 'text-danger-fg'],
  [/(?<![\w-])text-red-600(?![\w-])/g, 'text-danger-fg'],
  [/(?<![\w-])text-red-700(?![\w-])/g, 'text-danger-fg'],
  [/text-green-700\s+dark:text-green-400\b/g, 'text-success-fg'],
  [/(?<![\w-])text-green-700(?![\w-])/g, 'text-success-fg'],
  [/text-amber-700\s+dark:text-amber-300\b/g, 'text-warning-fg'],
  [/(?<![\w-])text-amber-700(?![\w-])/g, 'text-warning-fg'],
  [/text-indigo-600\s+dark:text-indigo-400\b/g, 'text-accent-fg'],
  [/text-indigo-700\s+dark:text-indigo-300\b/g, 'text-accent-fg'],
  [/(?<![\w-])text-indigo-600(?![\w-])/g, 'text-accent-fg'],
  [/(?<![\w-])text-indigo-700(?![\w-])/g, 'text-accent-fg'],
  [/bg-indigo-600\s+hover:bg-indigo-700\b/g, 'bg-accent-solid hover:bg-accent'],
  [/(?<![\w-])bg-indigo-600(?![\w/-])/g, 'bg-accent-solid'],
  [/(?<![\w-])bg-indigo-700(?![\w/-])/g, 'bg-accent'],
  [/bg-red-50\s+dark:bg-red-950\/\d+\b/g, 'bg-danger-surface'],
  [/bg-green-50\s+dark:bg-green-950\/\d+\b/g, 'bg-success-surface'],
  [/bg-amber-50\s+dark:bg-amber-950\/\d+\b/g, 'bg-warning-surface'],
  [/bg-indigo-50\s+dark:bg-indigo-950\/\d+\b/g, 'bg-accent-surface'],

  // Clean leftover dark: colour variants that only mirrored the light token
  [/\s*dark:text-neutral-(?:50|100|200|300|400|500)\b/g, ''],
  [/\s*dark:bg-neutral-(?:800|900|950)\b/g, ''],
  [/\s*dark:border-neutral-(?:600|700|800)\b/g, ''],
]

const UNMAPPED_RE =
  /\b(?:bg|text|border|ring)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-(?:50|100|200|300|400|500|600|700|800|900|950)\b/g

function walk(dir) {
  const out = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      if (name === 'node_modules' || name === '__tests__') continue
      out.push(...walk(path))
    } else if (name.endsWith('.tsx')) {
      out.push(path)
    }
  }
  return out
}

const files = walk(targetDir)
let changedFiles = 0
let totalReplacements = 0
/** @type {Record<string, number>} */
const unmapped = {}

for (const file of files) {
  let text = readFileSync(file, 'utf8')
  const original = text
  let fileHits = 0
  for (const [re, rep] of REPLACEMENTS) {
    const before = text
    text = text.replace(re, rep)
    if (text !== before) {
      const m = before.match(re)
      fileHits += m ? m.length : 1
    }
  }
  // Collapse double spaces in class strings (best-effort)
  text = text.replace(/className="([^"]*)"/g, (_, cls) => {
    return `className="${cls.replace(/\s{2,}/g, ' ').trim()}"`
  })
  text = text.replace(/className=\{`([^`]*)`\}/g, (_, cls) => {
    return `className={\`${cls.replace(/\s{2,}/g, ' ').trim()}\`}`
  })

  const leftovers = text.match(UNMAPPED_RE) ?? []
  for (const lit of leftovers) {
    unmapped[lit] = (unmapped[lit] ?? 0) + 1
  }

  if (text !== original) {
    changedFiles++
    totalReplacements += fileHits
    if (!dryRun) {
      writeFileSync(file, text)
    }
  }
}

console.log(
  `codemod ${dryRun ? '(dry-run)' : '(write)'}: ${changedFiles} files, ~${totalReplacements} replacements`,
)
console.log('\nUnmapped literals (top 40):')
const sorted = Object.entries(unmapped).sort((a, b) => b[1] - a[1])
for (const [lit, n] of sorted.slice(0, 40)) {
  console.log(`  ${n}\t${lit}`)
}
console.log(`\nunmapped_total=${sorted.reduce((s, [, n]) => s + n, 0)}`)
