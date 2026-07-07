#!/usr/bin/env node
/**
 * W05 — forbid raw UUID prefixes in instructor/reviewer-facing LMS pages.
 *
 * Usage: node scripts/check-entity-labels.mjs
 */
import { readFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const targets = [
  'src/pages/lms/moderation-dashboard.tsx',
  'src/pages/lms/peer-review-summary-page.tsx',
  'src/pages/lms/course-module-assignment-page.tsx',
]

const RAW_ID_PREFIX = /\.slice\(\s*0\s*,\s*8\s*\)/

/** @type {{ file: string; line: number; text: string }[]} */
const violations = []

for (const rel of targets) {
  const path = join(__dirname, '..', rel)
  const lines = readFileSync(path, 'utf8').split('\n')
  lines.forEach((text, index) => {
    if (!RAW_ID_PREFIX.test(text)) return
    violations.push({ file: rel, line: index + 1, text: text.trim() })
  })
}

if (violations.length === 0) {
  console.log('✓ Entity label check passed (no raw id.slice(0, 8) in W05 surfaces).')
  process.exit(0)
}

console.error(`✗ Found ${violations.length} raw UUID prefix pattern(s):\n`)
for (const v of violations) {
  console.error(`  ${v.file}:${v.line}`)
  console.error(`    ${v.text}`)
}
console.error('\nFix: use formatEntityLabel() / <EntityLabel> with API-provided names or neutral fallbacks.')
process.exit(1)