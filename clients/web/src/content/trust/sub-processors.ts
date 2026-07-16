export type DpaStatus = 'signed' | 'in-review' | 'not-applicable'

/** How an AI vendor appears on the trust list relative to BYOK. */
export type AiProcessingMode = 'platform_default' | 'when_configured'

export type SubProcessor = {
  name: string
  service: string
  dataCategories: string[]
  headquarters: string
  dataRegion: string
  dpaStatus: DpaStatus
  privacyUrl: string
  /**
   * When set, this row is an optional AI vendor: it is a Lextures sub-processor
   * only when the platform (or customer BYOK) routes traffic to it.
   */
  aiProcessingMode?: AiProcessingMode
}

/** Effective date of this list — update when adding/removing vendors. */
export const SUB_PROCESSORS_EFFECTIVE_DATE = '2026-07-15'

/**
 * Legal note for AI / BYOK (AP.7 / S07 alignment).
 * Customer-configured BYOK endpoints are the customer’s processors for that traffic;
 * Lextures lists vendors here for platform-default routing and when those vendors
 * are configured as AI providers on the instance.
 */
export const AI_SUBPROCESSOR_BYOK_NOTE =
  'Optional AI features may send prompts to third-party model providers. Vendors marked “when configured” are Lextures sub-processors only when that provider is enabled for the instance (platform key or customer bring-your-own-key). A customer’s direct BYOK connection to their own Azure OpenAI, Anthropic, OpenAI, Bedrock, or Vertex account is processing under the customer’s agreement with that vendor — not automatically a Lextures sub-processor. OpenRouter is listed when used as a routing gateway. See Settings → Intelligence and the AI usage disclosure for which providers are active on a given deployment.'

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
    service:
      'AI language model (tutor, grading assistance, content generation) — when configured as an AI provider',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://www.anthropic.com/privacy',
    aiProcessingMode: 'when_configured',
  },
  {
    name: 'OpenAI',
    service: 'AI language model (tutor, content generation) — when configured as an AI provider',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'signed',
    privacyUrl: 'https://openai.com/policies/privacy-policy',
    aiProcessingMode: 'when_configured',
  },
  {
    name: 'OpenRouter',
    service:
      'AI model routing gateway — when configured as an AI provider (not used on BYOK-only deployments that omit OpenRouter)',
    dataCategories: ['course content', 'anonymized student queries'],
    headquarters: 'US',
    dataRegion: 'US',
    dpaStatus: 'in-review',
    privacyUrl: 'https://openrouter.ai/privacy',
    aiProcessingMode: 'when_configured',
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
