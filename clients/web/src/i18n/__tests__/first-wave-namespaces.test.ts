import { beforeAll, describe, expect, it } from 'vitest'
import { i18n } from '../index'

describe('first-wave namespace bundles (plan W01)', () => {
  beforeAll(async () => {
    await i18n.loadLanguages(['en', 'es', 'fr'])
  })

  it('resolves parent dashboard title in Spanish', async () => {
    await i18n.changeLanguage('es')
    expect(i18n.t('parent.dashboard.title', { ns: 'parent' })).toBe('Panel familiar')
  })

  it('resolves billing checkout cancel title in French', async () => {
    await i18n.changeLanguage('fr')
    expect(i18n.t('billing.checkout.cancel.title', { ns: 'billing' })).toBe('Paiement annulé')
  })

  it('resolves onboarding welcome title in English', async () => {
    await i18n.changeLanguage('en')
    expect(i18n.t('onboarding.welcome.title', { ns: 'onboarding', lng: 'en' })).toBe('Welcome to Lextures')
  })

  it('resolves dashboard title in English', async () => {
    await i18n.changeLanguage('en')
    expect(i18n.t('dashboard.title', { ns: 'dashboard', lng: 'en' })).toBe('Dashboard')
  })
})
