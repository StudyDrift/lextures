#!/usr/bin/env node
/**
 * UX.2 AC-9 — every core barrel export must appear in the component gallery
 * (imported or rendered). Gallery imports satisfy "≥1 importer outside ui/".
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const barrelPath = join(webRoot, 'src/components/ui/index.ts')
const galleryDir = join(webRoot, 'src/pages/design')
const galleryEntryPath = join(galleryDir, 'components-gallery.tsx')

const barrel = readFileSync(barrelPath, 'utf8')
// Page shell + demos (and helpers) under /design — split for file-size budgets.
const galleryFiles = readdirSync(galleryDir)
  .filter((n) => n.endsWith('.tsx') && (n.startsWith('components-gallery') || n === 'gallery-block.tsx'))
  .map((n) => join(galleryDir, n))
const gallery = galleryFiles.map((p) => readFileSync(p, 'utf8')).join('\n')
const galleryEntry = readFileSync(galleryEntryPath, 'utf8')

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
  // UX.6 utilities / context (not visual gallery demos)
  'FieldContext',
  'useFieldContext',
  'mergeDescribedBy',
  // UX.5 hook (gallery demos use MoveToPositionMenu / ClickToMoveDropZone)
  'useClickToMove',
])

function isTypeName(n) {
  return (
    n.endsWith('Props') ||
    n.endsWith('Tone') ||
    n.endsWith('Variant') ||
    n.endsWith('Option') ||
    n.endsWith('Options') ||
    n.endsWith('Item') ||
    n.endsWith('Action') ||
    n.endsWith('Handle') ||
    n.endsWith('Value') ||
    n === 'ControlSize' ||
    n === 'MenuItem' ||
    n === 'BreadcrumbItem' ||
    n === 'ComboboxOption' ||
    n === 'SegmentedOption' ||
    n === 'DescriptionItem' ||
    n === 'EmptyStateAction' ||
    n === 'ReorderableItemMeta'
  )
}

const components = [...exportNames].filter((n) => !SKIP.has(n) && !isTypeName(n))

const failures = []

const hasUiImport =
  /from ['"]\.\.\/\.\.\/components\/ui['"]/.test(gallery) ||
  /from ['"]\.\.\/\.\.\/components\/ui\/index['"]/.test(gallery)
if (!hasUiImport) {
  failures.push('Gallery must import from components/ui barrel (entry or demos)')
}
if (!galleryEntry.includes('ComponentsGalleryDemos') && !galleryEntry.includes("from '../../components/ui'")) {
  failures.push('components-gallery.tsx must wire demos or import ui barrel directly')
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
