import { describe, expect, it } from 'vitest'
import {
  extractTocEntries,
  LEGAL_VERSIONS,
  PRIVACY_POLICY,
  slugifyHeading,
  TERMS_OF_SERVICE,
} from '../legal-documents'

describe('legal-documents', () => {
  it('exports matching versions for privacy and terms', () => {
    expect(PRIVACY_POLICY.version).toBe(LEGAL_VERSIONS.privacy_policy.version)
    expect(TERMS_OF_SERVICE.version).toBe(LEGAL_VERSIONS.terms_of_service.version)
  })

  it('privacy policy markdown includes required compliance keywords', () => {
    const body = PRIVACY_POLICY.bodyMarkdown.toLowerCase()
    expect(body).toContain('ferpa')
    expect(body).toContain('gdpr')
    expect(body).toContain('coppa')
    expect(body).toContain('ccpa')
    expect(body).toContain('anthropic')
    expect(body).toContain('openai')
    expect(body).toContain('openrouter')
    expect(body).toContain('privacy@lextures.com')
  })

  it('terms markdown includes required sections', () => {
    const body = TERMS_OF_SERVICE.bodyMarkdown.toLowerCase()
    expect(body).toContain('acceptable use')
    expect(body).toContain('dmca')
    expect(body).toContain('arbitration')
    expect(body).toContain('ai-generated')
  })

  it('slugifyHeading produces stable anchor ids', () => {
    expect(slugifyHeading('Your Rights Under GDPR')).toBe('your-rights-under-gdpr')
  })

  it('extractTocEntries finds level-2 headings', () => {
    const toc = extractTocEntries('## One\n\n## Two\n')
    expect(toc).toEqual([
      { id: 'one', title: 'One' },
      { id: 'two', title: 'Two' },
    ])
  })
})
