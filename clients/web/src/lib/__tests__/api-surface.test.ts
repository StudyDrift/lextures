/**
 * TD.1 FR-10 — export-surface guard for `src/lib/*-api.ts` modules.
 *
 * Pins the set of exported symbols per API module so TD.11/TD.12 module splits
 * cannot silently drop an export. Intentional changes: update the golden via
 * UPDATE_GOLDEN=1 (or delete the golden and re-run with the env var set).
 */
import { readdirSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const LIB_DIR = join(import.meta.dirname, '..')
const GOLDEN_PATH = join(import.meta.dirname, 'api-surface.golden.json')

function listApiModules(): string[] {
  return readdirSync(LIB_DIR)
    .filter((f) => f.endsWith('-api.ts'))
    .sort()
}

/** Extract top-level export names from a TypeScript module source (heuristic, AST-free). */
export function extractExportNames(source: string): string[] {
  const names = new Set<string>()

  // export function / async function / class / const / let / var / type / interface / enum
  const decl =
    /^export\s+(?:declare\s+)?(?:async\s+)?(?:function\*?|class|const|let|var|type|interface|enum)\s+([A-Za-z_$][\w$]*)/gm
  for (const m of source.matchAll(decl)) {
    names.add(m[1])
  }

  // export { a, b as c }
  const named = /^export\s*\{([^}]+)\}/gm
  for (const m of source.matchAll(named)) {
    for (const part of m[1].split(',')) {
      const trimmed = part.trim()
      if (!trimmed || trimmed === 'type' || trimmed.startsWith('type ')) {
        // export { type Foo } or empty
        const typeName = trimmed.replace(/^type\s+/, '').trim()
        if (typeName) {
          const asParts = typeName.split(/\s+as\s+/)
          names.add(asParts[asParts.length - 1].trim())
        }
        continue
      }
      const asParts = trimmed.replace(/^type\s+/, '').split(/\s+as\s+/)
      const exported = asParts[asParts.length - 1].trim()
      if (exported) names.add(exported)
    }
  }

  // export default — record as "default"
  if (/^export\s+default\b/m.test(source)) {
    names.add('default')
  }

  return [...names].sort()
}

function buildSurface(): Record<string, string[]> {
  const out: Record<string, string[]> = {}
  for (const file of listApiModules()) {
    const src = readFileSync(join(LIB_DIR, file), 'utf8')
    out[file] = extractExportNames(src)
  }
  return out
}

function updateGoldenRequested(): boolean {
  const v = (process.env.UPDATE_GOLDEN ?? '').trim().toLowerCase()
  return v === '1' || v === 'true' || v === 'yes'
}

describe('API module export surface (TD.1 FR-10)', () => {
  it('matches committed export sets for each *-api.ts module', () => {
    const live = buildSurface()
    const modules = Object.keys(live)
    expect(modules.length).toBeGreaterThan(50)

    if (updateGoldenRequested()) {
      mkdirSync(join(import.meta.dirname), { recursive: true })
      writeFileSync(GOLDEN_PATH, `${JSON.stringify(live, null, 2)}\n`, 'utf8')
      expect(Object.keys(live).length).toBeGreaterThan(0)
      return
    }

    if (!existsSync(GOLDEN_PATH)) {
      expect.fail(
        `missing ${GOLDEN_PATH}\nCreate with: UPDATE_GOLDEN=1 npm test -- src/lib/__tests__/api-surface.test.ts`,
      )
    }

    const golden = JSON.parse(readFileSync(GOLDEN_PATH, 'utf8')) as Record<string, string[]>
    const goldenModules = Object.keys(golden).sort()
    const liveModules = modules.sort()

    const addedModules = liveModules.filter((m) => !golden[m])
    const removedModules = goldenModules.filter((m) => !live[m])

    const exportDiffs: string[] = []
    for (const m of liveModules) {
      if (!golden[m]) continue
      const want = new Set(golden[m])
      const got = new Set(live[m])
      for (const name of got) {
        if (!want.has(name)) exportDiffs.push(`+ ${m}: ${name}`)
      }
      for (const name of want) {
        if (!got.has(name)) exportDiffs.push(`- ${m}: ${name}`)
      }
    }

    if (addedModules.length || removedModules.length || exportDiffs.length) {
      const lines = [
        'API export surface diverged from api-surface.golden.json',
        ...addedModules.map((m) => `  + module ${m}`),
        ...removedModules.map((m) => `  - module ${m}`),
        ...exportDiffs.map((d) => `  ${d}`),
        'If intentional: UPDATE_GOLDEN=1 npm test -- src/lib/__tests__/api-surface.test.ts',
      ]
      expect.fail(lines.join('\n'))
    }
  })

  it('extractExportNames finds named exports', () => {
    const src = `
export function fetchFoo() {}
export const BAR = 1
export type Baz = string
export { qux as quux }
`
    expect(extractExportNames(src)).toEqual(['BAR', 'Baz', 'fetchFoo', 'quux'])
  })
})

