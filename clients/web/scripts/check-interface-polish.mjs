#!/usr/bin/env node
/**
 * CI check for make-interfaces-feel-better patterns (plan 22.1).
 * Fails on bare Tailwind `transition` (transition-property: all).
 *
 * Usage: npm run interface-polish:check
 */
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(__dirname, '..', 'src')

const BARE_TRANSITION =
  /(?:^|[\s"'`])(?:motion-safe:)?transition(?:[\s"'`]|$)/

function walk(dir) {
  /** @type {string[]} */
  const files = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      files.push(...walk(path))
    } else if (/\.tsx?$/.test(name)) {
      files.push(path)
    }
  }
  return files
}

/** @type {{ file: string; line: number; text: string }[]} */
const violations = []

for (const file of walk(srcRoot)) {
  const lines = readFileSync(file, 'utf8').split('\n')
  lines.forEach((text, index) => {
    if (!BARE_TRANSITION.test(text)) return
    // Allow specific transition utilities on the same line
    if (/transition-(?:colors|transform|opacity|\[)/.test(text)) {
      // Still flag if bare `transition` also appears
      const stripped = text
        .replace(/transition-(?:colors|transform|opacity|\[[^\]]+\])/g, '')
        .replace(/motion-safe:transition-(?:colors|transform|opacity|\[[^\]]+\])/g, '')
      if (!/(?:^|[\s"'`])(?:motion-safe:)?transition(?:[\s"'`]|$)/.test(stripped)) return
    }
    violations.push({
      file: relative(join(__dirname, '..'), file),
      line: index + 1,
      text: text.trim().slice(0, 120),
    })
  })
}

if (violations.length === 0) {
  console.log('✓ Interface polish check passed (no bare `transition` utilities).')
  process.exit(0)
}

console.error(`✗ Found ${violations.length} bare \`transition\` utility(ies):\n`)
for (const v of violations.slice(0, 40)) {
  console.error(`  ${v.file}:${v.line}`)
  console.error(`    ${v.text}`)
}
if (violations.length > 40) {
  console.error(`  … and ${violations.length - 40} more`)
}
console.error('\nFix: run `node scripts/fix-bare-transitions.mjs --write` or use transition-colors / transition-transform.')
process.exit(1)
