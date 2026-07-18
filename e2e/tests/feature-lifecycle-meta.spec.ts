/**
 * E2E.3 — unit-level validation of the feature lifecycle dependency manifest.
 * Fails on missing Priority 1/2 families, invalid disabled contracts, or parent cycles.
 */
import { test, expect } from '@playwright/test'
import {
  FEATURE_LIFECYCLE_FAMILIES,
  detectParentCycles,
  priorityFamilies,
  validateFeatureLifecycleManifest,
} from '../lib/feature-lifecycle-manifest.js'

test.describe('Feature lifecycle manifest metadata', () => {
  test('manifest validates with no dependency cycles', () => {
    const errors = validateFeatureLifecycleManifest()
    expect(errors, errors.join('; ')).toEqual([])
    expect(detectParentCycles()).toEqual([])
  })

  test('Priority 1 families cover collaboration, credentials, commerce, and AI shards', () => {
    const p1 = priorityFamilies(1)
    expect(p1.length).toBeGreaterThanOrEqual(7)
    const shards = new Set(p1.map((f) => f.shard))
    expect(shards.has('collaboration')).toBe(true)
    expect(shards.has('credentials')).toBe(true)
    expect(shards.has('commerce-api')).toBe(true)
    expect(shards.has('ai')).toBe(true)
  })

  test('every family with edges documents parent authority or a known gap', () => {
    for (const family of FEATURE_LIFECYCLE_FAMILIES) {
      for (const edge of family.edges) {
        if (!edge.parentAuthoritative) {
          expect(edge.knownGap?.trim().length, family.id).toBeGreaterThan(0)
        }
      }
    }
  })

  test('disabled status contracts are explicit (no permissive unions)', () => {
    for (const family of FEATURE_LIFECYCLE_FAMILIES) {
      for (const probe of family.probes) {
        expect([403, 404, 501, 503]).toContain(probe.authDisabledStatus)
      }
    }
  })

  test('interactive quizzes lists eight child controls under hosting/course parents', () => {
    const iq = FEATURE_LIFECYCLE_FAMILIES.find((f) => f.id === 'interactive-quizzes')
    expect(iq).toBeTruthy()
    expect(iq!.children.map((c) => c.key).sort()).toEqual(
      [
        'ffIqAiGeneration',
        'ffIqGradebookPush',
        'ffIqGuestJoin',
        'ffIqHomework',
        'ffIqPublicKitCatalog',
        'ffIqStudentPaced',
        'ffIqTeamMode',
      ].sort(),
    )
    // Hosting is a master alongside the course flag (FR-3).
    expect(iq!.masterFlags.some((m) => m.key === 'ffIqLiveHosting')).toBe(true)
  })
})
