export type DpaStatus = 'signed' | 'in-review' | 'not-applicable'

export type SubProcessor = {
  name: string
  service: string
  dataCategories: string[]
  headquarters: string
  dataRegion: string
  dpaStatus: DpaStatus
  privacyUrl: string
}

/** Effective date of this list — update when adding/removing vendors. */
export const SUB_PROCESSORS_EFFECTIVE_DATE = '2026-05-21'

export const SUB_PROCESSORS: SubProcessor[] = [
  {
    name: 'Amazon Web Services (AWS)',
    service: 'Cloud infrastructure, compute, and managed databases',
    dataCategories: ['all user data', 'course content', 'media files'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://aws.amazon.com/privacy/',
  },
  {
    name: 'Anthropic',
    service: 'AI language model (AI tutor, grading assistance, content generation)',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://www.anthropic.com/privacy',
  },
  {
    name: 'OpenAI',
    service: 'AI language model (AI tutor, content generation)',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://openai.com/policies/privacy-policy',
  },
  {
    name: 'OpenRouter',
    service: 'AI model routing and orchestration',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'in-review',
    privacyUrl: 'https://openrouter.ai/privacy',
  },
  {
    name: 'Postmark (ActiveCampaign)',
    service: 'Transactional email delivery (notifications, password resets)',
    dataCategories: ['email address', 'display name'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://activecampaign.com/legal/privacy-policy/',
  },
  {
    name: 'Cloudflare',
    service: 'CDN, DDoS protection, and DNS',
    dataCategories: ['IP address', 'request metadata'],
    headquarters: 'US',
    dataRegion: 'Global (edge)',
    dpaStatus: 'signed',
    privacyUrl: 'https://www.cloudflare.com/privacypolicy/',
  },
  {
    name: 'Sentry',
    service: 'Application error monitoring and performance tracing',
    dataCategories: ['error traces', 'anonymized usage metadata'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://sentry.io/privacy/',
  },
  {
    name: 'Stripe',
    service: 'Payment processing (institutional billing)',
    dataCategories: ['billing contact', 'payment method metadata'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://stripe.com/privacy',
  },
]
