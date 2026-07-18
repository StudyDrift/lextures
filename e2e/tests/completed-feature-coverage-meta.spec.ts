/**
 * E2E.4 — completed-feature coverage gate (no browser interaction).
 *
 * Fixture unit tests live in `npm run e2e:coverage:test` so they do not need
 * the API stack. Prefer `npm run e2e:coverage:check` as the lightweight CI gate.
 */
import { test, expect } from '@playwright/test'
import {
  REPO_ROOT,
  listEligibleCompletedStories,
  loadManifest,
  resolveManifestPath,
  validateCompletedFeatureCoverage,
} from '../lib/completed-feature-coverage.js'

test.describe('Completed feature coverage (E2E.4)', () => {
  test('manifest validates against the completed-doc tree', () => {
    const errors = validateCompletedFeatureCoverage({ repoRoot: REPO_ROOT })
    expect(errors, errors.join('\n')).toEqual([])
  })

  test('every eligible story has exactly one entry (AC-1)', () => {
    const manifest = loadManifest(resolveManifestPath(REPO_ROOT))
    const eligible = listEligibleCompletedStories(REPO_ROOT)
    const paths = new Set(manifest.entries.map((e) => e.path))
    expect(paths.size).toBe(manifest.entries.length)
    expect(eligible.length).toBe(manifest.entries.length)
    for (const p of eligible) expect(paths.has(p), p).toBe(true)
  })

  test('flagged entries expose six lifecycle dimensions (AC-3)', () => {
    const manifest = loadManifest(resolveManifestPath(REPO_ROOT))
    const flagged = manifest.entries.filter((e) => e.flags)
    expect(flagged.length).toBeGreaterThan(0)
    for (const entry of flagged) {
      for (const dim of [
        'settingsToggle',
        'disabledState',
        'enabledJourney',
        'authorization',
        'dependency',
        'rollback',
      ] as const) {
        const v = entry.flags![dim]
        expect(v === true || v === false || v === 'n/a', `${entry.id}.${dim}`).toBe(true)
      }
    }
  })

  test('owned missing rows include severity and milestone (AC-5)', () => {
    const manifest = loadManifest(resolveManifestPath(REPO_ROOT))
    for (const entry of manifest.entries.filter((e) => e.coverage === 'missing')) {
      expect(entry.owner.trim().length, entry.id).toBeGreaterThan(0)
      expect(entry.severity, entry.id).toBeTruthy()
      expect(entry.targetMilestone?.trim().length, entry.id).toBeGreaterThan(0)
    }
  })
})