import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getMissingKeyMetrics, recordMissingTranslationKey, resetMissingKeyMetrics } from '../missing-key'

describe('missing translation key metrics (plan W01 FR-5)', () => {
  beforeEach(() => {
    resetMissingKeyMetrics()
  })

  it('records missing key without throwing by default', () => {
    recordMissingTranslationKey({ locale: 'es', namespace: 'parent', key: 'parent.dashboard.title' })
    expect(getMissingKeyMetrics().size).toBe(1)
  })

  it('throws when VITE_I18N_FAIL_ON_MISSING is enabled', () => {
    vi.stubEnv('VITE_I18N_FAIL_ON_MISSING', '1')
    expect(() =>
      recordMissingTranslationKey({ locale: 'fr', namespace: 'dashboard', key: 'dashboard.title' }),
    ).toThrow(/missing_translation_key/)
    vi.unstubAllEnvs()
  })
})
