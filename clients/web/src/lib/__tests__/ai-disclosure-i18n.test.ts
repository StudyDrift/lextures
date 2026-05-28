import { describe, expect, it } from 'vitest'
import { aiDisclosureI18n } from '../ai-disclosure-i18n'

describe('aiDisclosureI18n', () => {
  it('exposes opt-out and banner strings', () => {
    expect(aiDisclosureI18n.optOutLabel).toMatch(/opt out/i)
    expect(aiDisclosureI18n.bannerUnderstand).toBeTruthy()
  })
})
