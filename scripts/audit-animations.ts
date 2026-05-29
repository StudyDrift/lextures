#!/usr/bin/env tsx
/**
 * Plan 12.7 — Animation audit script.
 *
 * Scans all .tsx and .css files under clients/web/src/ for bare Tailwind animation/transition
 * utilities that are not guarded by the motion-safe: or motion-reduce: variant.
 *
 * Exit code 0 → clean. Exit code 1 → violations found.
 *
 * Usage: tsx scripts/audit-animations.ts [--ci]
 */

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const ROOT = join(import.meta.dirname ?? __dirname, '..')
const SRC = join(ROOT, 'clients/web/src')

// Tailwind classes that require a motion guard when used in component markup.
// These patterns match bare utilities (no motion-safe: / motion-reduce: prefix).
const BARE_ANIMATE_RE =
  /(?<![:\w])(animate-(?!none\b)|transition-(?!none\b)|duration-\d|ease-(?:in|out|in-out|linear)\b)/g

// Classes that are acceptable without a motion guard:
//   - motion-safe:animate-* / motion-reduce:animate-*
//   - In @media (prefers-reduced-motion) blocks (CSS only)
const GUARDED_RE = /motion-(?:safe|reduce):(?:animate|transition|duration|ease)-/

type Violation = {
  file: string
  line: number
  col: number
  match: string
}

function walkDir(dir: string): string[] {
  const results: string[] = []
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const stat = statSync(full)
    if (stat.isDirectory()) {
      results.push(...walkDir(full))
    } else if (full.endsWith('.tsx') || full.endsWith('.css')) {
      results.push(full)
    }
  }
  return results
}

function auditFile(filePath: string): Violation[] {
  const src = readFileSync(filePath, 'utf-8')
  const violations: Violation[] = []
  const lines = src.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]

    // Skip lines in @media (prefers-reduced-motion) blocks (CSS)
    if (/prefers-reduced-motion/.test(line)) continue
    // Skip lines that already have a motion-safe/motion-reduce guard
    if (GUARDED_RE.test(line)) continue
    // Skip comment-only lines
    if (/^\s*(\/\/|\/\*|\*|#)/.test(line)) continue

    let match: RegExpExecArray | null
    BARE_ANIMATE_RE.lastIndex = 0
    while ((match = BARE_ANIMATE_RE.exec(line)) !== null) {
      violations.push({
        file: relative(ROOT, filePath),
        line: i + 1,
        col: match.index + 1,
        match: match[0],
      })
    }
  }
  return violations
}

function main() {
  const files = walkDir(SRC)
  const all: Violation[] = []
  for (const f of files) {
    all.push(...auditFile(f))
  }

  if (all.length === 0) {
    console.log('✓ Animation audit passed — no bare animate/transition utilities found.')
    process.exit(0)
  }

  console.error(`\n✗ Animation audit failed — ${all.length} violation(s) found:\n`)
  for (const v of all) {
    console.error(`  ${v.file}:${v.line}:${v.col}  bare "${v.match}" (use motion-safe: or motion-reduce: prefix)`)
  }
  console.error(
    '\nFix: prefix with motion-safe: (e.g. motion-safe:animate-spin) or wrap in ' +
      '@media (prefers-reduced-motion: no-preference) { ... } in CSS.\n',
  )
  process.exit(1)
}

main()
