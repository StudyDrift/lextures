#!/usr/bin/env node
/**
 * CI check for make-interfaces-feel-better patterns (plan 22.1) and AN.1 motion tokens.
 *
 * Fails on:
 *   - bare Tailwind `transition` (transition-property: all)
 *   - CSS that animates layout properties (width/height/top/left/margin)
 *
 * Warns (exit 0) on:
 *   - raw duration literals in feature TS/TSX (grace period — flip to fail later)
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

/** Layout props that must not be animated (AN.1 FR-9 / perf budget). */
const LAYOUT_ANIM_PROPS =
  /(?:^|[^\w-])(?:width|height|top|left|right|bottom|margin|margin-(?:top|right|bottom|left|inline|block)|padding|padding-(?:top|right|bottom|left))\s*:/i

const LAYOUT_TRANSITION_PROP =
  /transition(?:-property)?\s*:\s*[^;]*(?:width|height|top|left|right|bottom|margin|padding)/i

/** Raw ms / s duration literals in feature code (prefer durations.* from lib/motion). */
const RAW_DURATION_LITERAL =
  /(?:duration(?:Ms|Millis)?\s*[:=]\s*|animation(?:Duration)?\s*[:=]\s*|transitionDuration\s*[:=]\s*)(\d{2,4})\b/

const ALLOWED_DURATION_FILES = new Set([
  'src/lib/motion.ts',
  'src/lib/__tests__/motion.test.ts',
  'src/lib/route-transition.ts',
  'src/lib/__tests__/route-transition.test.ts',
])

const ALLOWED_LAYOUT_CSS = new Set([
  // Token / foundation files may mention layout props outside animations.
])

function walk(dir, pred) {
  /** @type {string[]} */
  const files = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      files.push(...walk(path, pred))
    } else if (pred(name)) {
      files.push(path)
    }
  }
  return files
}

/** @type {{ file: string; line: number; text: string }[]} */
const transitionViolations = []
/** @type {{ file: string; line: number; text: string }[]} */
const layoutViolations = []
/** @type {{ file: string; line: number; text: string }[]} */
const durationWarnings = []

for (const file of walk(srcRoot, (name) => /\.tsx?$/.test(name))) {
  const rel = relative(join(__dirname, '..'), file)
  const lines = readFileSync(file, 'utf8').split('\n')
  lines.forEach((text, index) => {
    const trimmed = text.trim()
    // Skip comments — the word "transition" in prose is not a Tailwind utility.
    if (trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      return
    }
    // Skip CSS/style object properties (`transition: …`), which are not Tailwind utilities.
    if (/^transition\s*:/.test(trimmed)) {
      return
    }
    if (BARE_TRANSITION.test(text)) {
      if (/transition-(?:colors|transform|opacity|\[)/.test(text)) {
        const stripped = text
          .replace(/transition-(?:colors|transform|opacity|\[[^\]]+\])/g, '')
          .replace(/motion-safe:transition-(?:colors|transform|opacity|\[[^\]]+\])/g, '')
        if (!/(?:^|[\s"'`])(?:motion-safe:)?transition(?:[\s"'`]|$)/.test(stripped)) {
          // ok — only named transition utilities
        } else {
          transitionViolations.push({
            file: rel,
            line: index + 1,
            text: text.trim().slice(0, 120),
          })
        }
      } else {
        transitionViolations.push({
          file: rel,
          line: index + 1,
          text: text.trim().slice(0, 120),
        })
      }
    }

    if (!ALLOWED_DURATION_FILES.has(rel) && !rel.includes('__tests__') && !rel.includes('.test.')) {
      const m = text.match(RAW_DURATION_LITERAL)
      if (m) {
        const ms = Number(m[1])
        // Ignore common non-motion numbers (HTTP timeouts, etc.) by requiring small animation range.
        if (ms >= 50 && ms <= 2000) {
          durationWarnings.push({
            file: rel,
            line: index + 1,
            text: text.trim().slice(0, 120),
          })
        }
      }
    }
  })
}

for (const file of walk(srcRoot, (name) => /\.css$/.test(name))) {
  const rel = relative(join(__dirname, '..'), file)
  if (ALLOWED_LAYOUT_CSS.has(rel)) continue
  const lines = readFileSync(file, 'utf8').split('\n')

  let keyframesDepth = 0
  lines.forEach((text, index) => {
    if (/@keyframes\b/.test(text)) {
      keyframesDepth = 1
    } else if (keyframesDepth > 0) {
      const opens = (text.match(/\{/g) || []).length
      const closes = (text.match(/\}/g) || []).length
      keyframesDepth += opens - closes
      if (keyframesDepth < 0) keyframesDepth = 0
    }

    if (keyframesDepth > 0 && LAYOUT_ANIM_PROPS.test(text) && !/^\s*\/\*/.test(text)) {
      layoutViolations.push({
        file: rel,
        line: index + 1,
        text: text.trim().slice(0, 120),
      })
    }
    if (LAYOUT_TRANSITION_PROP.test(text)) {
      layoutViolations.push({
        file: rel,
        line: index + 1,
        text: text.trim().slice(0, 120),
      })
    }
  })
}

let failed = false

if (transitionViolations.length === 0) {
  console.log('✓ No bare `transition` utilities.')
} else {
  failed = true
  console.error(`✗ Found ${transitionViolations.length} bare \`transition\` utility(ies):\n`)
  for (const v of transitionViolations.slice(0, 40)) {
    console.error(`  ${v.file}:${v.line}`)
    console.error(`    ${v.text}`)
  }
  if (transitionViolations.length > 40) {
    console.error(`  … and ${transitionViolations.length - 40} more`)
  }
  console.error('\nFix: run `node scripts/fix-bare-transitions.mjs --write` or use transition-colors / transition-transform.')
}

if (layoutViolations.length === 0) {
  console.log('✓ No layout-property animations in CSS (AN.1).')
} else {
  failed = true
  console.error(`\n✗ Found ${layoutViolations.length} layout-property animation(s) — use transform/opacity only:\n`)
  for (const v of layoutViolations.slice(0, 40)) {
    console.error(`  ${v.file}:${v.line}`)
    console.error(`    ${v.text}`)
  }
  console.error('\nUse motion tokens from src/lib/motion.ts / --dur-* / --ease-* instead.')
}

if (durationWarnings.length === 0) {
  console.log('✓ No raw duration literals in feature code (AN.1).')
} else {
  console.warn(
    `\n⚠ AN.1 warn-only: ${durationWarnings.length} raw duration literal(s) in feature code (prefer durations.* from @/lib/motion):\n`,
  )
  for (const v of durationWarnings.slice(0, 25)) {
    console.warn(`  ${v.file}:${v.line}`)
    console.warn(`    ${v.text}`)
  }
  if (durationWarnings.length > 25) {
    console.warn(`  … and ${durationWarnings.length - 25} more`)
  }
  console.warn('\n(Grace period: warnings only; will fail CI after the next release cycle.)')
}

if (failed) process.exit(1)
console.log('\n✓ Interface polish check passed.')
process.exit(0)
