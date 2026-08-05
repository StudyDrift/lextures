/// <reference types="node" />
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  FOCUS_ANCHOR_ALIASES,
  FOCUS_ANCHORS,
  FOCUS_ANCHOR_ID_MAX_LENGTH,
  isValidFocusAnchorIdFormat,
  parseCompositeAnchor,
  resolveFocusAnchor,
  RETIRED_FOCUS_ANCHOR_IDS,
} from '../focus-anchors'
import { resolveSettingId } from '../settings-registry'
import {
  cssEscape,
  hrefForTarget,
  FOCUS_ENTITY_QUERY_PARAM,
  FOCUS_QUERY_PARAM,
} from '../use-focus-anchor'

const here = dirname(fileURLToPath(import.meta.url))
const webSrc = resolve(here, '../..')
const repoRoot = resolve(webSrc, '../../..')

describe('focus-anchors integrity', () => {
  it('has unique IDs', () => {
    const ids = FOCUS_ANCHORS.map((a) => a.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('IDs match naming regex and length bound', () => {
    for (const a of FOCUS_ANCHORS) {
      expect(isValidFocusAnchorIdFormat(a.id), a.id).toBe(true)
      expect(a.id.length).toBeLessThanOrEqual(FOCUS_ANCHOR_ID_MAX_LENGTH)
      expect(a.route.trim().length).toBeGreaterThan(0)
      expect(a.label.trim().length).toBeGreaterThan(0)
      expect(a.labelKey.trim().length).toBeGreaterThan(0)
      expect(['control', 'region', 'entity']).toContain(a.kind)
    }
  })

  it('aliases point at existing anchors or resolvable IDs', () => {
    for (const [from, to] of Object.entries(FOCUS_ANCHOR_ALIASES)) {
      if (from === to) continue
      const resolved = resolveFocusAnchor(to)
      expect(resolved, `alias ${from} → ${to}`).not.toBeNull()
      expect(RETIRED_FOCUS_ANCHOR_IDS.has(to)).toBe(false)
    }
  })
})

describe('resolveFocusAnchor', () => {
  it('resolves registry IDs', () => {
    expect(resolveFocusAnchor('course.general.dates')?.id).toBe('course.general.dates')
    expect(resolveFocusAnchor('modules.item')?.kind).toBe('entity')
  })

  it('resolves aliases', () => {
    expect(resolveFocusAnchor('course.general.hero')?.id).toBe('course.general.hero-image')
    expect(resolveFocusAnchor('grading.groups')?.id).toBe('course.grading.groups')
    expect(resolveFocusAnchor('resend')?.id).toBe('enrollments.invitations')
  })

  it('resolves PS.1 setting IDs through the settings registry', () => {
    const full = 'assignment.outcomes-mapping.editor'
    expect(resolveSettingId(full)).toBe(full)
    const a = resolveFocusAnchor(full)
    expect(a).not.toBeNull()
    expect(a!.id).toBe(full)
    expect(a!.fromSettingsRegistry).toBe(true)
    expect(a!.container).toEqual({ type: 'accordion', id: 'outcomes-mapping' })
  })

  it('returns null for unknown / retired', () => {
    expect(resolveFocusAnchor('does.not.exist')).toBeNull()
    expect(resolveFocusAnchor('')).toBeNull()
  })

  it('parses composite entity forms', () => {
    expect(parseCompositeAnchor('modules.item:abc-123')).toEqual({
      baseId: 'modules.item',
      entityId: 'abc-123',
    })
    expect(parseCompositeAnchor('module:uuid-here')).toEqual({
      baseId: 'modules.module',
      entityId: 'uuid-here',
    })
    expect(resolveFocusAnchor('module:uuid-here')?.id).toBe('modules.module')
  })
})

describe('hrefForTarget', () => {
  it('substitutes params and appends focus', () => {
    const href = hrefForTarget(
      {
        route: '/courses/{courseCode}/settings/general',
        anchor: 'course.general.dates',
      },
      { courseCode: 'BIO101' },
    )
    expect(href.startsWith('/courses/BIO101/settings/general?')).toBe(true)
    const qs = new URLSearchParams(href.split('?')[1])
    expect(qs.get(FOCUS_QUERY_PARAM)).toBe('course.general.dates')
    expect(qs.get(FOCUS_ENTITY_QUERY_PARAM)).toBeNull()
  })

  it('attaches focusEntity for entity anchors', () => {
    const href = hrefForTarget(
      {
        route: '/courses/{courseCode}/modules',
        anchor: 'modules.item',
        entityKey: 'item-1',
      },
      { courseCode: 'C1' },
    )
    const qs = new URLSearchParams(href.split('?')[1])
    expect(qs.get(FOCUS_QUERY_PARAM)).toBe('modules.item')
    expect(qs.get(FOCUS_ENTITY_QUERY_PARAM)).toBe('item-1')
  })

  it('ignores unknown anchors (plain navigation)', () => {
    const href = hrefForTarget(
      {
        route: '/courses/{courseCode}/settings/general',
        anchor: 'does.not.exist',
      },
      { courseCode: 'X' },
    )
    expect(href).toBe('/courses/X/settings/general')
    expect(href.includes('focus=')).toBe(false)
  })

  it('substitutes itemId for editor routes', () => {
    const href = hrefForTarget(
      {
        route: '/courses/{courseCode}/modules/assignment/{itemId}',
        anchor: 'assignment.outcomes-mapping',
        entityKey: 'asg-9',
      },
      { courseCode: 'CHEM' },
    )
    expect(href.startsWith('/courses/CHEM/modules/assignment/asg-9?')).toBe(true)
    expect(href).toContain('focus=assignment.outcomes-mapping')
  })
})

describe('cssEscape', () => {
  it('escapes selector metacharacters', () => {
    const evil = 'a.b"x'
    const escaped = cssEscape(evil)
    expect(escaped).not.toBe('')
    // Must not throw when used in a selector construction
    expect(() => document.querySelector(`[data-x="${escaped}"]`)).not.toThrow()
  })
})

describe('catalog integrity (CC.8 FR-18)', () => {
  it('every server catalog anchor resolves in focus-anchors or PS.1', () => {
    const fixturePath = resolve(
      repoRoot,
      'server/internal/service/coursechecklist/testdata/catalog_targets.json',
    )
    const rows = JSON.parse(readFileSync(fixturePath, 'utf8')) as Array<{
      itemId: string
      route: string
      anchor?: string
    }>
    expect(rows.length).toBeGreaterThan(50)
    const missing: string[] = []
    for (const row of rows) {
      if (!row.anchor) continue
      if (!resolveFocusAnchor(row.anchor)) {
        missing.push(`${row.itemId} → ${row.anchor}`)
      }
    }
    expect(missing, `unresolved catalog anchors:\n${missing.join('\n')}`).toEqual([])
  })
})

describe('parity (CC.8 FR-19)', () => {
  function walkTsx(dir: string, out: string[] = []): string[] {
    for (const name of readdirSync(dir)) {
      if (name === 'node_modules' || name === '__tests__' || name === 'generated') continue
      const full = join(dir, name)
      const st = statSync(full)
      if (st.isDirectory()) walkTsx(full, out)
      else if (/\.(tsx|ts)$/.test(name)) out.push(full)
    }
    return out
  }

  it('every non-entity registry anchor is referenced in source as data-focus-anchor or via section panels', () => {
    const files = walkTsx(webSrc)
    const corpus = files.map((f) => readFileSync(f, 'utf8')).join('\n')

    // Section-level editor anchors are emitted as assignment.{section} / quiz.{section}
    // on SettingsAccordion details elements.
    const missing: string[] = []
    for (const a of FOCUS_ANCHORS) {
      if (a.kind === 'entity') {
        // Entity bases must appear; per-entity values are dynamic.
        if (
          !corpus.includes(`data-focus-anchor="${a.id}"`) &&
          !corpus.includes(`'${a.id}'`) &&
          !corpus.includes(`"${a.id}"`)
        ) {
          // Still require the id string somewhere in web src (registry counts only if also attached)
          if (!corpus.includes(`data-focus-anchor="${a.id}"`)) {
            missing.push(a.id)
          }
        }
        continue
      }
      // Control / region: require data-focus-anchor="id" somewhere
      if (!corpus.includes(`data-focus-anchor="${a.id}"`)) {
        // Editor section anchors may use template `assignment.${sectionId}`
        if (
          (a.id.startsWith('assignment.') || a.id.startsWith('quiz.')) &&
          (corpus.includes('data-focus-anchor={sectionId ? `assignment.') ||
            corpus.includes('data-focus-anchor={sectionId ? `quiz.'))
        ) {
          continue
        }
        missing.push(a.id)
      }
    }
    expect(missing, `anchors not attached in source:\n${missing.join('\n')}`).toEqual([])
  })
})
