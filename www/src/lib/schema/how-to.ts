import type { JsonLdNode } from '../document-head'
import { absoluteUrl } from './ids'

export type HowToStep = {
  name: string
  text: string
}

/**
 * HowTo for procedural help articles (FR-12).
 * Emitted for LLM comprehension; HowTo rich results are retired.
 */
export function buildHowTo(opts: {
  path: string
  name: string
  description: string
  steps: HowToStep[]
  siteOrigin?: string
}): JsonLdNode | null {
  if (!opts.steps.length) return null
  return {
    '@type': 'HowTo',
    '@id': `${absoluteUrl(opts.path, opts.siteOrigin)}#howto`,
    name: opts.name,
    description: opts.description,
    step: opts.steps.map((s, i) => ({
      '@type': 'HowToStep',
      position: i + 1,
      name: s.name,
      text: s.text,
    })),
    inLanguage: 'en',
  }
}
