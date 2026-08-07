#!/usr/bin/env node
/**
 * UX.3 — mechanical map of raw Tailwind type sizes → semantic roles.
 * Size-preserving intent (plan rollout step 2).
 *
 *   text-[10px]|text-[11px] → text-overline
 *   text-xs                 → text-caption
 *   text-sm                 → text-body-sm
 *   text-base               → text-body
 *   text-lg                 → text-body-lg
 *   text-xl                 → text-subtitle
 *   text-2xl                → text-title
 *   text-3xl                → text-title-lg
 *   text-4xl+               → text-display
 *
 * Usage:
 *   npm run type:codemod:dry -- --dir=src/components/a11y
 *   npm run type:codemod -- --dir=src/components/a11y
 */
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')

const write = process.argv.includes('--write')
const dryRun = !write || process.argv.includes('--dry-run')
const dirArg = process.argv.find((a) => a.startsWith('--dir='))
const targetDir = dirArg ? resolve(webRoot, dirArg.slice('--dir='.length)) : join(webRoot, 'src')

const REPLACEMENTS = [
  [/\btext-\[(?:10|11)px\]/g, 'text-overline'],
  [/\btext-\[0\.625rem\]/g, 'text-overline'],
  [/\btext-\[0\.6875rem\]/g, 'text-overline'],
  [/\btext-xs\b/g, 'text-caption'],
  [/\btext-sm\b/g, 'text-body-sm'],
  [/\btext-base\b/g, 'text-body'],
  [/\btext-lg\b/g, 'text-body-lg'],
  [/\btext-xl\b/g, 'text-subtitle'],
  [/\btext-2xl\b/g, 'text-title'],
  [/\btext-3xl\b/g, 'text-title-lg'],
  [/\btext-(?:4xl|5xl|6xl|7xl|8xl|9xl)\b/g, 'text-display'],
  [/\btext-\[12px\]/g, 'text-overline'],
  [/\btext-\[13px\]/g, 'text-caption'],
  [/\btext-\[14px\]/g, 'text-body-sm'],
  [/\btext-\[15px\]/g, 'text-body-sm'],
  [/\btext-\[16px\]/g, 'text-body'],
  [/\btext-\[18px\]/g, 'text-body-lg'],
  [/\btext-\[20px\]/g, 'text-subtitle'],
  [/\btext-\[22px\]/g, 'text-title'],
  [/\btext-\[24px\]/g, 'text-title'],
  [/\btext-\[28px\]/g, 'text-title-lg'],
  [/\btext-\[32px\]/g, 'text-title-lg'],
  [/\btext-\[36px\]/g, 'text-display'],
]

const HARD_EXEMPT = [
  /src\/pages\/typeface-page\.tsx$/,
  /src\/pages\/design\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
]

function walk(dir) {
  const files = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      if (name === 'node_modules' || name === 'dist') continue
      files.push(...walk(path))
    } else if (name.endsWith('.tsx')) {
      files.push(path)
    }
  }
  return files
}

let filesChanged = 0
let replacements = 0

for (const file of walk(targetDir)) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (HARD_EXEMPT.some((re) => re.test(rel))) continue
  let text = readFileSync(file, 'utf8')
  let fileHits = 0
  for (const [re, to] of REPLACEMENTS) {
    text = text.replace(re, () => {
      fileHits++
      return to
    })
  }
  if (fileHits === 0) continue
  replacements += fileHits
  filesChanged++
  if (!dryRun) {
    writeFileSync(file, text)
  }
  console.log(`${dryRun ? '[dry] ' : ''}${rel}: ${fileHits}`)
}

console.log(
  `\n${dryRun ? 'Would change' : 'Changed'} ${filesChanged} files, ${replacements} replacements.`,
)
if (dryRun) {
  console.log('Re-run with --write to apply.')
}
