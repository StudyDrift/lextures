import { describe, expect, it } from 'vitest'
import { applyDocumentScrollMode, isStandalonePublicRoute } from '../standalone-public-routes'

describe('isStandalonePublicRoute', () => {
  it('returns true for trust center', () => {
    expect(isStandalonePublicRoute('/trust')).toBe(true)
  })

  it('returns true for login routes', () => {
    expect(isStandalonePublicRoute('/login')).toBe(true)
    expect(isStandalonePublicRoute('/login/magic-link')).toBe(true)
    expect(isStandalonePublicRoute('/signup')).toBe(true)
  })

  it('returns true for verify routes', () => {
    expect(isStandalonePublicRoute('/verify/abc')).toBe(true)
  })

  it('returns false for authenticated app routes', () => {
    expect(isStandalonePublicRoute('/')).toBe(false)
    expect(isStandalonePublicRoute('/courses')).toBe(false)
    expect(isStandalonePublicRoute('/privacy-centre')).toBe(false)
    expect(isStandalonePublicRoute('/privacy')).toBe(false)
    expect(isStandalonePublicRoute('/terms')).toBe(false)
    expect(isStandalonePublicRoute('/security')).toBe(false)
    expect(isStandalonePublicRoute('/accessibility')).toBe(false)
    expect(isStandalonePublicRoute('/accessibility/vpat')).toBe(false)
    expect(isStandalonePublicRoute('/settings/account')).toBe(false)
  })
})

describe('applyDocumentScrollMode', () => {
  it('sets overflow auto on public routes', () => {
    applyDocumentScrollMode('/trust')
    expect(document.documentElement.style.overflow).toBe('auto')
  })
})
