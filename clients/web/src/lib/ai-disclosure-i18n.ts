/** Anchor id of the AI settings block on the Account settings page. */
export const AI_SETTINGS_ANCHOR_ID = 'ai-settings'

/** Where "Manage AI settings" style links should point (Account settings, AI block). */
export const AI_SETTINGS_PATH = `/settings/account#${AI_SETTINGS_ANCHOR_ID}`

/** i18n-style keys for AI disclosure UI (plan 10.17 / AP.6). */

export const aiDisclosureI18n = {
  pageTitle: 'AI usage disclosure',
  pageIntro:
    'Lextures uses third-party AI models (via providers your institution configures) to power optional features. This page describes which models are used, what data is sent, and how to opt out.',
  optOutTitle: 'AI processing',
  optOutDescription:
    'When enabled, your course content and messages are not sent to external AI providers for tutoring, notebook answers, translations, or similar features. The AI tutor is hidden while this is on.',
  optOutLabel: 'Opt out of AI processing',
  optOutSaved: 'AI preference saved.',
  tutorOptOutLabel: 'Turn off the AI tutor for my account',
  tutorOptOutDescription:
    'Hides the AI tutor in your courses. Other AI features follow the AI processing setting above.',
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
