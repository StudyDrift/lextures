#!/usr/bin/env node
/**
 * TD.2 — web source file basenames must be kebab-case.
 *
 * Scope: clients/web/src  (all .ts / .tsx files)
 * Pattern: /^[a-z0-9.-]+\.(ts|tsx)$/
 * Allowlist: scripts/allowlists/file-naming.txt (owners TD.11 / TD.14)
 *
 * Usage:
 *   node scripts/check-file-naming.mjs
 *   node scripts/check-file-naming.mjs --report
 *   node scripts/check-file-naming.mjs --self-test
 *   STRUCTURE_CHECKS_WARN=1 node scripts/check-file-naming.mjs
 */
import { readdirSync, readFileSync, statSync, mkdtempSync, writeFileSync, mkdirSync, rmSync } from 'node:fs'
import { join, relative, basename, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { tmpdir } from 'node:os'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = process.env.ROOT_OVERRIDE || join(__dirname, '..')
const ALLOW_FILE = process.env.ALLOW_OVERRIDE || join(REPO_ROOT, 'scripts/allowlists/file-naming.txt')
const SRC_ROOT = join(REPO_ROOT, 'clients/web/src')
const KEBAB = /^[a-z0-9.-]+\.(ts|tsx)$/

const args = new Set(process.argv.slice(2))
const REPORT = args.has('--report')
const SELF_TEST = args.has('--self-test')
const WARN = process.env.STRUCTURE_CHECKS_WARN === '1'

if (args.has('-h') || args.has('--help')) {
  console.log(`TD.2 web file naming (kebab-case).
Usage: node scripts/check-file-naming.mjs [--report] [--self-test]`)
  process.exit(0)
}

function loadAllowlist(file) {
  try {
    return new Set(
      readFileSync(file, 'utf8')
        .split(/\r?\n/)
        .map((l) => l.trim())
        .filter((l) => l && !l.startsWith('#')),
    )
  } catch {
    return new Set()
  }
}

function walk(dir, out = []) {
  let entries
  try {
    entries = readdirSync(dir, { withFileTypes: true })
  } catch {
    return out
  }
  for (const ent of entries) {
    const p = join(dir, ent.name)
    if (ent.isDirectory()) {
      walk(p, out)
    } else if (ent.isFile() && /\.tsx?$/.test(ent.name)) {
      out.push(p)
    }
  }
  return out
}

function check(root, allowFile) {
  const allow = loadAllowlist(allowFile)
  const src = join(root, 'clients/web/src')
  const files = walk(src)
  /** @type {string[]} */
  const violations = []
  for (const abs of files) {
    const rel = relative(root, abs).split('\\').join('/')
    const base = basename(abs)
    if (KEBAB.test(base)) continue
    if (allow.has(rel)) continue
    violations.push(rel)
  }
  return { files: files.length, violations, allowSize: allow.size }
}

function selfTest() {
  let failures = 0
  const tmp = mkdtempSync(join(tmpdir(), 'td2-naming-'))
  const src = join(tmp, 'clients/web/src/components')
  mkdirSync(src, { recursive: true })
  writeFileSync(join(src, 'good-name.tsx'), 'export {}')
  writeFileSync(join(src, 'BadName.tsx'), 'export {}')
  writeFileSync(join(src, 'AllowedThing.tsx'), 'export {}')
  const allow = join(tmp, 'allow.txt')
  writeFileSync(allow, 'clients/web/src/components/AllowedThing.tsx\n')

  process.env.ROOT_OVERRIDE = tmp
  process.env.ALLOW_OVERRIDE = allow
  // Re-run logic inline
  const { violations } = check(tmp, allow)
  if (!violations.includes('clients/web/src/components/BadName.tsx')) {
    console.error('FAIL: expected BadName.tsx')
    failures++
  } else {
    console.log('OK: BadName.tsx rejected')
  }
  if (violations.includes('clients/web/src/components/AllowedThing.tsx')) {
    console.error('FAIL: allowlisted file reported')
    failures++
  } else {
    console.log('OK: allowlisted suppressed')
  }
  if (violations.includes('clients/web/src/components/good-name.tsx')) {
    console.error('FAIL: good-name should pass')
    failures++
  } else {
    console.log('OK: kebab-case accepted')
  }
  rmSync(tmp, { recursive: true, force: true })
  if (failures) {
    console.error(`self-test FAILED (${failures})`)
    process.exit(1)
  }
  console.log('self-test PASSED')
  process.exit(0)
}

if (SELF_TEST) selfTest()

const { files, violations, allowSize } = check(REPO_ROOT, ALLOW_FILE)

for (const rel of violations.sort()) {
  console.error(
    `${rel}: basename must be kebab-case [a-z0-9.-]+.(ts|tsx) (rule: file-naming; owner: TD.11/TD.14)`,
  )
  console.error('  Fix: rename the file; update imports. See docs/ARCHITECTURE_CONVENTIONS.md §5.')
}

console.log(
  `file-naming: checked ${files} files; unallowlisted violations: ${violations.length}`,
)
console.log(`  file-naming   remaining allowlist entries: ${allowSize}`)

if (REPORT) process.exit(0)

if (violations.length === 0) process.exit(0)
if (WARN) {
  console.error(
    `WARN: file-naming: ${violations.length} violation(s) (STRUCTURE_CHECKS_WARN=1; not failing)`,
  )
  process.exit(0)
}
process.exit(1)
