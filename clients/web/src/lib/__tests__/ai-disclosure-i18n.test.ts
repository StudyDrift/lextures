import { describe, expect, it } from 'vitest'
import { aiDisclosureI18n, aiDisclosureProviderPhrase } from '../ai-disclosure-i18n'

describe('aiDisclosureI18n', () => {
  it('exposes opt-out and banner strings', () => {
    expect(aiDisclosureI18n.optOutLabel).toMatch(/opt out/i)
    expect(aiDisclosureI18n.bannerUnderstand).toBeTruthy()
  })

  it('is provider-agnostic (no OpenRouter-only copy)', () => {
    expect(aiDisclosureI18n.pageIntro).not.toMatch(/via OpenRouter/i)
    expect(aiDisclosureI18n.bannerBody).not.toMatch(/via OpenRouter/i)
  })

  it('formats active provider phrase', () => {
    expect(aiDisclosureProviderPhrase('Anthropic')).toBe('Anthropic')
    expect(aiDisclosureProviderPhrase('')).toBe('an AI provider')
    expect(aiDisclosureProviderPhrase(undefined)).toBe('an AI provider')
  })
})
