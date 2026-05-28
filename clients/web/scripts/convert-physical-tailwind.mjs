#!/usr/bin/env node
/**
 * Converts common physical Tailwind utilities to logical equivalents (plan 11.2).
 * Run from clients/web: node scripts/convert-physical-tailwind.mjs
 */
import fs from 'node:fs'
import path from 'node:path'

const root = path.join(import.meta.dirname, '..', 'src')

const replacements = [
  [/\btext-left\b/g, 'text-start'],
  [/\btext-right\b/g, 'text-end'],
  [/\bml-(\d+)/g, 'ms-$1'],
  [/\bmr-(\d+)/g, 'me-$1'],
  [/\bpl-(\d+)/g, 'ps-$1'],
  [/\bpr-(\d+)/g, 'pe-$1'],
  [/\bpl-(\[)/g, 'ps-$1'],
  [/\bpr-(\[)/g, 'pe-$1'],
  [/\bml-(\[)/g, 'ms-$1'],
  [/\bmr-(\[)/g, 'me-$1'],
  [/\bleft-(\d+)/g, 'start-$1'],
  [/\bright-(\d+)/g, 'end-$1'],
  [/\bleft-(\[)/g, 'start-$1'],
  [/\bright-(\[)/g, 'end-$1'],
  [/\bborder-l\b/g, 'border-s'],
  [/\bborder-r\b/g, 'border-e'],
  [/\brounded-l\b/g, 'rounded-s'],
  [/\brounded-r\b/g, 'rounded-e'],
  [/\bscroll-pl-/g, 'scroll-ps-'],
  [/\bscroll-pr-/g, 'scroll-pe-'],
]

function walk(dir, files = []) {
  for (const name of fs.readdirSync(dir)) {
    const p = path.join(dir, name)
    const st = fs.statSync(p)
    if (st.isDirectory()) walk(p, files)
    else if (/\.(tsx?|css)$/.test(name)) files.push(p)
  }
  return files
}

let changed = 0
for (const file of walk(root)) {
  const before = fs.readFileSync(file, 'utf8')
  let next = before
  for (const [re, rep] of replacements) next = next.replace(re, rep)
  if (next !== before) {
    fs.writeFileSync(file, next)
    changed++
  }
}
console.log(`Updated ${changed} files under src/`)
