/** i18n-style keys for AI disclosure UI (plan 10.17 / AP.6). */

export const aiDisclosureI18n = {
  pageTitle: 'AI usage disclosure',
  pageIntro:
    'Lextures uses third-party AI models (via providers your institution configures) to power optional features. This page describes which models are used, what data is sent, and how to opt out.',
  optOutTitle: 'AI processing',
  optOutDescription:
    'When enabled, your course content and messages are not sent to external AI providers for tutoring, notebook answers, translations, or similar features.',
  optOutLabel: 'Opt out of AI processing',
  optOutSaved: 'AI preference saved.',
  bannerTitle: 'AI disclosure',
  bannerBody:
    'This feature sends your input to an AI model for processing. You can opt out anytime in Settings → Account → AI processing.',
  bannerUnderstand: 'I understand',
  bannerOptOutLink: 'AI processing settings',
  fullDisclosureLink: 'Full AI disclosure',
  adminTitle: 'Governance',
  adminIntro: 'Enable or disable AI features and restrict models for your organization.',
  adminSave: 'Save AI governance',
  adminSaved: 'AI governance settings saved.',
  featureDisabled: 'This AI feature is disabled by your organization.',
  processingDisabled: 'AI processing is disabled for this account.',
} as const

/** Formats the active-provider clause for the in-app disclosure banner (AP.6 FR-5). */
export function aiDisclosureProviderPhrase(providerLabel?: string): string {
  const label = providerLabel?.trim()
  if (!label) return 'an AI provider'
  return label
}
