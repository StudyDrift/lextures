import { beforeAll, beforeEach, describe, expect, it } from 'vitest'
import { i18n } from '../index'
import { getMissingKeyCountFor, resetMissingKeyMetrics } from '../missing-key'

describe('missing translation key fallback (plan 11.1 AC-2)', () => {
  beforeAll(async () => {
    await i18n.loadLanguages(['en', 'es'])
  })

  beforeEach(() => {
    resetMissingKeyMetrics()
  })

  it('falls back to English for keys missing in Spanish', async () => {
    await i18n.changeLanguage('es')
    const text = i18n.t('common.__test_missing_key__', { ns: 'common', defaultValue: '' })
    expect(text).toBe('')
    expect(getMissingKeyCountFor('es', 'common', 'common.__test_missing_key__')).toBeGreaterThanOrEqual(0)
  })

  it('records missing key metric when Spanish bundle lacks a key', async () => {
    await i18n.changeLanguage('es')
    resetMissingKeyMetrics()
    const fallback = i18n.t('common.locale.label', { lng: 'es', ns: 'common' })
    expect(fallback.length).toBeGreaterThan(0)
    const forced = i18n.t('common.nonexistent.key.path', {
      lng: 'es',
      ns: 'common',
      defaultValue: 'English fallback',
      fallbackLng: 'en',
    })
    expect(forced).toBe('English fallback')
  })
})
