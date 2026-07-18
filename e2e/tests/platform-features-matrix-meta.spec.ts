/**
 * E2E.2 — unit-level validation of the platform feature matrix metadata.
 * Fails when PLATFORM_FEATURE_DEFINITIONS gains a key without manifest classification (AC-5).
 */
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'
import {
  PLATFORM_FEATURE_CATEGORIES,
  PLATFORM_FEATURE_MATRIX,
  UI_SAMPLE_PLATFORM_FEATURES,
  parsePlatformFeatureDefinitionKeys,
  validatePlatformFeatureMatrix,
} from '../lib/platform-feature-matrix.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const definitionsPath = path.join(
  repoRoot,
  'clients/web/src/components/settings/platform-feature-definitions.ts',
)

test.describe('Platform feature matrix metadata', () => {
  test('matrix keys and labels are unique with required fields', () => {
    const definitionKeys = parsePlatformFeatureDefinitionKeys(readFileSync(definitionsPath, 'utf8'))
    const errors = validatePlatformFeatureMatrix(definitionKeys)
    expect(errors, errors.join('; ')).toEqual([])
  })

  test('every PLATFORM_FEATURE_DEFINITIONS key has manifest classification', () => {
    const definitionKeys = parsePlatformFeatureDefinitionKeys(readFileSync(definitionsPath, 'utf8'))
    const manifestKeys = new Set(PLATFORM_FEATURE_MATRIX.map((e) => e.key))
    const missing = definitionKeys.filter((key) => !manifestKeys.has(key))
    expect(missing, `missing manifest entries: ${missing.join(', ')}`).toEqual([])
  })

  test('UI samples cover each category exactly once', () => {
    expect(UI_SAMPLE_PLATFORM_FEATURES.length).toBe(PLATFORM_FEATURE_CATEGORIES.length)
    const cats = UI_SAMPLE_PLATFORM_FEATURES.map((e) => e.category)
    expect(new Set(cats).size).toBe(cats.length)
  })

  test('settings-only flags document why they omit runtime payload fields', () => {
    const settingsOnly = PLATFORM_FEATURE_MATRIX.filter((e) => e.runtimeKey == null)
    expect(settingsOnly.length).toBeGreaterThan(0)
    for (const entry of settingsOnly) {
      expect(entry.settingsOnlyRationale?.trim().length, entry.key).toBeGreaterThan(0)
    }
  })
})
