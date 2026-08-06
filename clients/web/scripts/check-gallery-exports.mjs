#!/usr/bin/env node
/**
 * UX.2 AC-9 — every core barrel export must appear in the component gallery
 * (imported or rendered). Gallery imports satisfy "≥1 importer outside ui/".
 */
import { readFileSync } from 'node:fs'
import { join, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const barrelPath = join(webRoot, 'src/components/ui/index.ts')
const galleryPath = join(webRoot, 'src/pages/design/components-gallery.tsx')

const barrel = readFileSync(barrelPath, 'utf8')
const gallery = readFileSync(galleryPath, 'utf8')

const exportNames = new Set()
for (const line of barrel.split('\n')) {
  if (!line.includes('export {')) continue
  const m = line.match(/export \{([^}]+)\}/)
  if (!m) continue
  for (const part of m[1].split(',')) {
    let cleaned = part.replace(/\btype\b/g, '').trim()
    if (!cleaned) continue
    const asMatch = cleaned.match(/^(\w+)\s+as\s+(\w+)$/)
    if (asMatch) {
      exportNames.add(asMatch[1])
      exportNames.add(asMatch[2])
    } else {
      cleaned = cleaned.replace(/,/g, '').trim()
      if (/^\w+$/.test(cleaned)) exportNames.add(cleaned)
    }
  }
}

const SKIP = new Set([
  'cx',
  'sizeClasses',
  'focusRingClass',
  'toast',
  'toastSaveOk',
  'toastMutationError',
  'toastWithUndo',
])

function isTypeName(n) {
  return (
    n.endsWith('Props') ||
    n.endsWith('Tone') ||
    n.endsWith('Variant') ||
    n.endsWith('Option') ||
    n.endsWith('Item') ||
    n.endsWith('Action') ||
    n === 'ControlSize' ||
    n === 'MenuItem' ||
    n === 'BreadcrumbItem' ||
    n === 'ComboboxOption' ||
    n === 'SegmentedOption' ||
    n === 'DescriptionItem' ||
    n === 'EmptyStateAction'
  )
}

const components = [...exportNames].filter((n) => !SKIP.has(n) && !isTypeName(n))

const failures = []

if (!gallery.includes("from '../../components/ui'") && !gallery.includes('from "../../components/ui"')) {
  failures.push('Gallery must import from components/ui barrel')
}

for (const name of components) {
  const mentioned =
    new RegExp(`\\b${name}\\b`).test(gallery) || gallery.includes(`<${name}`)
  if (!mentioned) {
    failures.push(`Gallery missing component: ${name}`)
  }
}

if (failures.length) {
  console.error('Gallery export coverage FAIL:')
  for (const f of failures) console.error(`  - ${f}`)
  process.exit(1)
}

console.log(`OK — ${components.length} core components covered by gallery.`)
process.exit(0)
