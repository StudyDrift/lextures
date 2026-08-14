import { describe, expect, it } from 'vitest'
import { lintMetadataFromArticle, normalizeMarketingFinding } from '../marketing-finding'

describe('normalizeMarketingFinding', () => {
  it('maps camelCase lint fields', () => {
    expect(normalizeMarketingFinding({
      rule: 'fm.cluster',
      severity: 'error',
      message: 'Required metadata field is missing.',
      line: 0,
      path: 'cluster',
    })).toEqual({
      rule: 'fm.cluster',
      severity: 'error',
      message: 'Required metadata field is missing.',
      path: 'cluster',
    })
  })

  it('maps PascalCase lint fields from the live API', () => {
    expect(normalizeMarketingFinding({
      Rule: 'struct.faq-count',
      Severity: 'warn',
      Message: 'FAQ must contain 3–6 questions; found 0.',
      Line: 1,
      Column: 1,
    })).toEqual({
      rule: 'struct.faq-count',
      severity: 'warning',
      message: 'FAQ must contain 3–6 questions; found 0.',
      line: 1,
      column: 1,
    })
  })
})

describe('lintMetadataFromArticle', () => {
  it('sends the validator field names, not the write payload', () => {
    expect(lintMetadataFromArticle({
      title: 'Hello',
      description: 'A description',
      authorSlug: 'chase',
      cluster: 'learning',
      primaryQuestion: 'How do I start?',
      keywords: ['home'],
      locale: 'en',
      updatedAt: '2026-08-13T17:19:00Z',
    })).toEqual({
      title: 'Hello',
      description: 'A description',
      author: 'chase',
      cluster: 'learning',
      primaryQuestion: 'How do I start?',
      keywords: ['home'],
      locale: 'en',
      updated: '2026-08-13',
    })
  })
})
