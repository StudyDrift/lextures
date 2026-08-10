#!/usr/bin/env node
/**
 * UX.5 FR-5 / observability — drag_surfaces_without_alternative.
 *
 * Scans for @dnd-kit DndContext / useSortable usage outside the design-system
 * Reorderable helpers and requires each surface file (or a documented inventory
 * entry) to expose a single-pointer alternative marker:
 *   - MoveToPositionMenu / useClickToMove / data-click-to-move
 *   - or an allowlisted path in drag-surfaces-inventory.json
 *
 * Usage:
 *   node scripts/check-drag-alternatives.mjs
 *   node scripts/check-drag-alternatives.mjs --json
 *   node scripts/check-drag-alternatives.mjs --write-inventory
 */
import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'node:fs'
import { join, relative, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(__dirname, '..')
const srcRoot = join(webRoot, 'src')
const inventoryPath = join(webRoot, 'drag-surfaces-inventory.json')

const HARD_EXEMPT = [
  /src\/components\/ui\//,
  /src\/pages\/design\//,
  /\/__tests__\//,
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
  /src\/lib\/dnd\//,
  /src\/lib\/list-motion\.ts$/,
  /src\/lib\/reorderable\//,
]

const DND_MARKER =
  /\b(?:DndContext|useSortable|useDraggable|useDroppable)\b/
const ALTERNATIVE_MARKER =
  /\b(?:MoveToPositionMenu|useClickToMove|data-click-to-move|move-to-position-trigger|moveModuleToIndex|moveChildToIndex|moveItemToIndex)\b/

function walk(dir, pred) {
  const files = []
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      if (name === 'node_modules' || name === 'dist') continue
      files.push(...walk(path, pred))
    } else if (pred(name)) {
      files.push(path)
    }
  }
  return files
}

function isExempt(rel) {
  return HARD_EXEMPT.some((re) => re.test(rel.replace(/\\/g, '/')))
}

const asJson = process.argv.includes('--json')
const writeInventory = process.argv.includes('--write-inventory')

const inventory = existsSync(inventoryPath)
  ? JSON.parse(readFileSync(inventoryPath, 'utf8'))
  : { surfaces: [], notes: '' }

const allowlist = new Map(
  (inventory.surfaces || []).map((s) => [s.file.replace(/\\/g, '/'), s]),
)

const files = walk(srcRoot, (n) => n.endsWith('.tsx') || n.endsWith('.ts'))
/** @type {{ file: string; hasAlternative: boolean; status: string; note?: string }[]} */
const surfaces = []

for (const file of files) {
  const rel = relative(webRoot, file).replace(/\\/g, '/')
  if (isExempt(rel)) continue
  if (!rel.endsWith('.tsx')) continue
  const text = readFileSync(file, 'utf8')
  if (!DND_MARKER.test(text)) continue
  const hasAlternative = ALTERNATIVE_MARKER.test(text)
  const listed = allowlist.get(rel)
  const justified =
    listed &&
    (listed.alternative === 'documented' ||
      listed.alternative === 'keyboard-only' ||
      listed.alternative === 'click-to-move' ||
      listed.alternative === 'move-menu' ||
      listed.alternative === 'n/a-non-reorder')
  surfaces.push({
    file: rel,
    hasAlternative,
    status: hasAlternative ? 'ok' : justified ? 'allowlisted' : 'missing',
    note: listed?.note,
  })
}

const missing = surfaces.filter((s) => s.status === 'missing')
const metric = {
  drag_surfaces_total: surfaces.length,
  drag_surfaces_with_alternative: surfaces.filter((s) => s.status !== 'missing').length,
  drag_surfaces_without_alternative: missing.length,
}

if (writeInventory) {
  const next = {
    notes:
      'UX.5 drag inventory. alternative: click-to-move | move-menu | keyboard-only | documented | n/a-non-reorder',
    surfaces: surfaces.map((s) => ({
      file: s.file,
      alternative: s.hasAlternative
        ? 'click-to-move'
        : allowlist.get(s.file)?.alternative || 'documented',
      note: s.note || allowlist.get(s.file)?.note || '',
    })),
  }
  writeFileSync(inventoryPath, JSON.stringify(next, null, 2) + '\n')
  console.log(`Wrote ${inventoryPath} (${next.surfaces.length} surfaces)`)
  process.exit(0)
}

if (asJson) {
  console.log(JSON.stringify({ ...metric, surfaces, missing }, null, 2))
} else {
  console.log(
    `drag_surfaces_without_alternative=${metric.drag_surfaces_without_alternative} (total=${metric.drag_surfaces_total}, ok=${metric.drag_surfaces_with_alternative})`,
  )
  if (missing.length) {
    console.log('Missing single-pointer alternative:')
    for (const m of missing) console.log(`  - ${m.file}`)
  }
}

if (missing.length > 0) {
  process.exit(1)
}
