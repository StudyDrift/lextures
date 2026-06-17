import { describe, expect, it } from 'vitest'
import { buildLinkedInCertificationUrl } from '../linkedin-share'

describe('buildLinkedInCertificationUrl', () => {
  it('builds LinkedIn certification URL with required params', () => {
    const url = buildLinkedInCertificationUrl({
      name: 'Intro to Data Science',
      organizationName: 'Lextures',
      issueYear: 2026,
      issueMonth: 6,
      certUrl: 'https://app.example.com/verify/abc',
      certId: 'abc',
    })
    const parsed = new URL(url)
    expect(parsed.origin + parsed.pathname).toBe('https://www.linkedin.com/profile/add')
    expect(parsed.searchParams.get('startTask')).toBe('CERTIFICATION_NAME')
    expect(parsed.searchParams.get('name')).toBe('Intro to Data Science')
    expect(parsed.searchParams.get('organizationName')).toBe('Lextures')
    expect(parsed.searchParams.get('issueYear')).toBe('2026')
    expect(parsed.searchParams.get('issueMonth')).toBe('6')
    expect(parsed.searchParams.get('certUrl')).toBe('https://app.example.com/verify/abc')
    expect(parsed.searchParams.get('certId')).toBe('abc')
  })

  it('uses organizationId when provided', () => {
    const url = buildLinkedInCertificationUrl({
      name: 'Course',
      organizationName: 'Lextures',
      organizationId: '12345',
      issueYear: 2026,
      issueMonth: 1,
      certUrl: 'https://example.com/v',
      certId: 'id',
    })
    const parsed = new URL(url)
    expect(parsed.searchParams.get('organizationId')).toBe('12345')
    expect(parsed.searchParams.get('organizationName')).toBeNull()
  })
})