/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  __clearSettingsMatchCacheForTests,
  getMatchingSettingIds,
  getSettingById,
  getSuggestedPins,
  isValidSettingIdFormat,
  listSettingsForSurface,
  resolveSettingId,
  RETIRED_SETTING_IDS,
  sectionHasMatchingSettings,
  SETTING_ID_ALIASES,
  SETTING_ID_MAX_LENGTH,
  SETTINGS_REGISTRY,
  SETTINGS_SECTION_TITLES,
  settingMatchesQuery,
  SUGGESTED_PINS,
  type SettingsSurface,
} from '../settings-registry'

const QUIZ_SECTIONS = new Set(Object.keys(SETTINGS_SECTION_TITLES.quiz))
const ASSIGNMENT_SECTIONS = new Set(Object.keys(SETTINGS_SECTION_TITLES.assignment))

describe('settings-registry integrity', () => {
  it('has unique IDs', () => {
    const ids = SETTINGS_REGISTRY.map((d) => d.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('IDs match naming regex and length ≤ 96', () => {
    for (const d of SETTINGS_REGISTRY) {
      expect(isValidSettingIdFormat(d.id), d.id).toBe(true)
      expect(d.id.length).toBeLessThanOrEqual(SETTING_ID_MAX_LENGTH)
      expect(d.id.startsWith(`${d.surface}.`)).toBe(true)
      expect(d.id.split('.')[1]).toBe(d.section)
    }
  })

  it('every section is valid for its surface', () => {
    for (const d of SETTINGS_REGISTRY) {
      const allowed = d.surface === 'quiz' ? QUIZ_SECTIONS : ASSIGNMENT_SECTIONS
      expect(allowed.has(d.section), `${d.id} section ${d.section}`).toBe(true)
    }
  })

  it('aliases point at existing canonical IDs and not at retired', () => {
    for (const [from, to] of Object.entries(SETTING_ID_ALIASES)) {
      expect(getSettingById(to), `alias ${from} → ${to}`).toBeDefined()
      expect(RETIRED_SETTING_IDS.has(to)).toBe(false)
    }
  })

  it('required descriptor fields are non-empty', () => {
    for (const d of SETTINGS_REGISTRY) {
      expect(d.label.trim().length).toBeGreaterThan(0)
      expect(Array.isArray(d.keywords)).toBe(true)
      expect(typeof d.pinnable).toBe('boolean')
    }
  })

  it('every SUGGESTED_PINS id resolves and is pinnable on its surface (PS.4 FR-7)', () => {
    for (const surface of Object.keys(SUGGESTED_PINS) as SettingsSurface[]) {
      for (const id of SUGGESTED_PINS[surface]) {
        const canonical = resolveSettingId(id)
        expect(canonical, `suggested id unresolvable: ${id}`).not.toBeNull()
        const d = getSettingById(canonical!)
        expect(d, id).toBeDefined()
        expect(d!.surface).toBe(surface)
        expect(d!.pinnable).toBe(true)
      }
      const resolved = getSuggestedPins(surface)
      expect(resolved.length).toBe(SUGGESTED_PINS[surface].length)
    }
  })
})

describe('resolveSettingId', () => {
  it('returns canonical ID for known settings', () => {
    expect(resolveSettingId('quiz.presentation.lockdown-mode')).toBe('quiz.presentation.lockdown-mode')
    expect(resolveSettingId('assignment.scheduling.due-date')).toBe('assignment.scheduling.due-date')
  })

  it('returns null for unknown IDs', () => {
    expect(resolveSettingId('quiz.bogus.control')).toBeNull()
    expect(resolveSettingId('')).toBeNull()
  })

  it('returns null for retired IDs', () => {
    // RETIRED is empty in PS.1; verify the set type and empty behaviour.
    for (const id of RETIRED_SETTING_IDS) {
      expect(resolveSettingId(id)).toBeNull()
    }
  })
})

describe('search matching', () => {
  it('matches label, keyword, and section title', () => {
    const lockdown = getSettingById('quiz.presentation.lockdown-mode')!
    expect(settingMatchesQuery(lockdown, 'lockdown')).toBe(true)
    expect(settingMatchesQuery(lockdown, 'kiosk')).toBe(true)
    expect(settingMatchesQuery(lockdown, 'Presentation')).toBe(true)
    expect(settingMatchesQuery(lockdown, 'zzzz')).toBe(false)
  })

  it('getMatchingSettingIds is memoised for the same query', () => {
    __clearSettingsMatchCacheForTests()
    const a = getMatchingSettingIds('quiz', 'lockdown')
    const b = getMatchingSettingIds('quiz', 'lockdown')
    expect(a).toBe(b)
    expect(a.has('quiz.presentation.lockdown-mode')).toBe(true)
    // Dependent focus-loss control shares "lockdown" keyword (AC-2)
    expect(a.has('quiz.presentation.focus-loss-threshold')).toBe(true)
    // Must not false-positive on unrelated sections
    expect(a.has('quiz.access.access-code')).toBe(false)
  })

  it('does not match lockdown when searching access code', () => {
    __clearSettingsMatchCacheForTests()
    const matches = getMatchingSettingIds('quiz', 'access code')
    expect(matches.has('quiz.access.access-code')).toBe(true)
    expect(matches.has('quiz.presentation.lockdown-mode')).toBe(false)
  })

  it('sectionHasMatchingSettings hides empty sections', () => {
    expect(sectionHasMatchingSettings('quiz', 'presentation', 'lockdown')).toBe(true)
    expect(sectionHasMatchingSettings('quiz', 'scheduling', 'lockdown')).toBe(false)
    expect(sectionHasMatchingSettings('quiz', 'scheduling', '')).toBe(true)
    expect(sectionHasMatchingSettings('assignment', 'access', 'zzzz')).toBe(false)
  })

  it('empty query matches every surface entry', () => {
    __clearSettingsMatchCacheForTests()
    for (const surface of ['quiz', 'assignment'] as SettingsSurface[]) {
      const matches = getMatchingSettingIds(surface, '')
      expect(matches.size).toBe(listSettingsForSurface(surface).length)
    }
  })
})

describe('registry/DOM parity', () => {
  const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')

  function extractSettingIds(source: string): string[] {
    const ids: string[] = []
    const re = /settingId=["']([^"']+)["']/g
    let m: RegExpExecArray | null
    while ((m = re.exec(source))) {
      ids.push(m[1]!)
    }
    return ids
  }

  it('every SettingRow settingId in panels exists in the registry', () => {
    const quizSrc = readFileSync(
      resolve(root, 'components/quiz/quiz-page-settings-panel.tsx'),
      'utf8',
    )
    const assignmentSrc = readFileSync(
      resolve(root, 'components/assignment/assignment-page-settings-panel.tsx'),
      'utf8',
    )
    const used = [...extractSettingIds(quizSrc), ...extractSettingIds(assignmentSrc)]
    expect(used.length).toBeGreaterThan(20)
    for (const id of used) {
      expect(getSettingById(id), `missing registry entry for ${id}`).toBeDefined()
    }
  })

  it('every registry entry is referenced by its panel under some prop combination', () => {
    const quizSrc = readFileSync(
      resolve(root, 'components/quiz/quiz-page-settings-panel.tsx'),
      'utf8',
    )
    const assignmentSrc = readFileSync(
      resolve(root, 'components/assignment/assignment-page-settings-panel.tsx'),
      'utf8',
    )
    const quizIds = new Set(extractSettingIds(quizSrc))
    const assignmentIds = new Set(extractSettingIds(assignmentSrc))

    for (const d of SETTINGS_REGISTRY) {
      if (d.surface === 'quiz') {
        expect(quizIds.has(d.id), `quiz panel missing SettingRow for ${d.id}`).toBe(true)
      } else {
        expect(assignmentIds.has(d.id), `assignment panel missing SettingRow for ${d.id}`).toBe(true)
      }
    }
  })
})
