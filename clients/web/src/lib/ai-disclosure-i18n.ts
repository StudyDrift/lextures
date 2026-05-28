/** i18n-style keys for AI disclosure UI (plan 10.17). */

export const aiDisclosureI18n = {
  pageTitle: 'AI usage disclosure',
  pageIntro:
    'Lextures uses third-party AI models via OpenRouter to power optional features. This page describes which models are used, what data is sent, and how to opt out.',
  optOutTitle: 'AI processing',
  optOutDescription:
    'When enabled, your course content and messages are not sent to external AI providers for tutoring, notebook answers, translations, or similar features.',
  optOutLabel: 'Opt out of AI processing',
  optOutSaved: 'AI preference saved.',
  bannerTitle: 'AI disclosure',
  bannerBody:
    'This feature sends your input to an AI model via OpenRouter for processing. You can opt out anytime in Settings → Account → AI processing.',
  bannerUnderstand: 'I understand',
  bannerOptOutLink: 'AI processing settings',
  fullDisclosureLink: 'Full AI disclosure',
  adminTitle: 'AI governance',
  adminIntro: 'Enable or disable AI features and restrict models for your organization.',
  adminSave: 'Save AI governance',
  adminSaved: 'AI governance settings saved.',
  featureDisabled: 'This AI feature is disabled by your organization.',
  processingDisabled: 'AI processing is disabled for this account.',
} as const
