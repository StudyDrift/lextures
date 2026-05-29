import { describe, expect, it } from 'vitest'
import { LEGAL_VERSIONS, PRIVACY_POLICY, TERMS_OF_SERVICE } from '../legal-documents'
import { MARKETING_SITE_URLS } from '../marketing-site'

describe('legal-documents', () => {
  it('exports matching versions for privacy and terms', () => {
    expect(PRIVACY_POLICY.version).toBe(LEGAL_VERSIONS.privacy_policy.version)
    expect(TERMS_OF_SERVICE.version).toBe(LEGAL_VERSIONS.terms_of_service.version)
  })

  it('points legal pages at the marketing site', () => {
    expect(PRIVACY_POLICY.url).toBe(MARKETING_SITE_URLS.privacy)
    expect(TERMS_OF_SERVICE.url).toBe(MARKETING_SITE_URLS.terms)
  })
})
