import type { JsonLdNode } from '../document-head'
import { faqId } from './ids'

export type FaqPair = {
  question: string
  /** Must match visible on-page answer text verbatim (FR-13). */
  answer: string
}

export function buildFaqPage(
  path: string,
  pairs: FaqPair[],
  siteOrigin?: string,
): JsonLdNode | null {
  if (!pairs.length) return null
  return {
    '@type': 'FAQPage',
    '@id': faqId(path, siteOrigin),
    mainEntity: pairs.map(({ question, answer }) => ({
      '@type': 'Question',
      name: question,
      acceptedAnswer: {
        '@type': 'Answer',
        text: answer,
      },
    })),
  }
}
