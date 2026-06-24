#!/usr/bin/env node
/**
 * Codemod: replace bare Tailwind `transition` (maps to transition-property: all)
 * with property-specific utilities (plan 22.1, rule 14).
 *
 * Usage: node scripts/fix-bare-transitions.mjs [--write]
 */
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(__dirname, '..', 'src')
const write = process.argv.includes('--write')

/** @param {string} classStr */
function inferTransition(classStr) {
  const hasTransform =
    /\b(?:rotate-|scale-|translate-|group-open:rotate|group-hover:scale|group-focus:scale)/.test(classStr) ||
    /\$\{[^}]*rotate/.test(classStr)
  const hasOpacity =
    /\b(?:group-hover:opacity|group-focus-within:opacity|opacity-0|opacity-100|focus-visible:opacity)/.test(
      classStr,
    )
  const hasShadow = /\bhover:shadow|shadow-card-hover|hover-shadow-card/.test(classStr)
  const hasFilter = /\bblur-/.test(classStr)
  const hasColor =
    /\bhover:(?:bg|text|border)-/.test(classStr) ||
    /\bfocus:(?:border|ring|bg|text)-/.test(classStr) ||
    /\bfocus-visible:(?:ring|bg|text)-/.test(classStr) ||
    /\b(?:bg|text|border)-.*(?:hover:|focus:)/.test(classStr)

  const props = []
  if (hasTransform) props.push('transform')
  if (hasOpacity) props.push('opacity')
  if (hasShadow) props.push('box-shadow')
  if (hasFilter) props.push('filter')
  if (hasColor) {
    props.push('background-color', 'color', 'border-color')
  }

  const unique = [...new Set(props)]
  if (unique.length === 0) return 'transition-colors'
  if (unique.length === 1 && unique[0] === 'transform') return 'transition-transform'
  if (unique.length === 1 && unique[0] === 'opacity') return 'transition-opacity'
  if (unique.length === 1 && unique[0] === 'box-shadow') return 'transition-[box-shadow]'
  if (unique.length === 1 && unique[0] === 'filter') return 'transition-[filter]'
  if (
    unique.length === 2 &&
    unique.includes('transform') &&
    unique.includes('opacity')
  ) {
    return 'transition-[transform,opacity]'
  }
  if (
    unique.length === 2 &&
    unique.includes('opacity') &&
    unique.includes('background-color')
  ) {
    return 'transition-[opacity,background-color,color]'
  }
  return `transition-[${unique.join(',')}]`
}

/** Replace bare `transition` tokens in a class string segment. */
function fixClassString(classStr) {
  return classStr.replace(/(^|\s)(motion-safe:)?transition(\s|$)/g, (match, before, motionPrefix, after) => {
    const prefix = motionPrefix ?? ''
    const replacement = inferTransition(classStr)
    return `${before}${prefix}${replacement}${after}`
  })
}

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

let totalReplacements = 0
let filesChanged = 0

for (const file of walk(srcRoot)) {
  const original = readFileSync(file, 'utf8')
  let next = original
  let fileReplacements = 0

  // Template literals in className={`...`}
  next = next.replace(/className=\{`([^`]*)`\}/g, (full, classStr) => {
    const fixed = fixClassString(classStr)
    if (fixed !== classStr) fileReplacements++
    return `className={\`${fixed}\`}`
  })

  // Static className="..."
  next = next.replace(/className="([^"]*)"/g, (full, classStr) => {
    const fixed = fixClassString(classStr)
    if (fixed !== classStr) fileReplacements++
    return `className="${fixed}"`
  })

  // cn(...) and similar string literals passed as first arg — conservative: only quoted strings on own lines
  next = next.replace(/'([^']*\btransition\b[^']*)'/g, (full, classStr) => {
    if (/transition-(?:colors|transform|opacity|\[)/.test(classStr) && !/(?:^|\s)transition(?:\s|$)/.test(classStr)) {
      return full
    }
    const fixed = fixClassString(classStr)
    if (fixed !== classStr) fileReplacements++
    return `'${fixed}'`
  })

  if (next !== original) {
    totalReplacements += fileReplacements
    filesChanged++
    if (write) {
      writeFileSync(file, next, 'utf8')
    } else {
      console.log(`Would update ${file} (${fileReplacements} segment(s))`)
    }
  }
}

const mode = write ? 'Updated' : 'Would update'
console.log(`\n${mode} ${filesChanged} file(s), ~${totalReplacements} class segment(s).`)
if (!write) {
  console.log('Re-run with --write to apply changes.')
}
