import { afterEach, describe, expect, it } from 'vitest'
import { applyDocumentScrollMode, isStandalonePublicRoute } from '../standalone-public-routes'

describe('isStandalonePublicRoute', () => {
  it('matches trust pages', () => {
    expect(isStandalonePublicRoute('/trust')).toBe(true)
  })

  it('matches auth pages', () => {
    expect(isStandalonePublicRoute('/login')).toBe(true)
    expect(isStandalonePublicRoute('/login/magic-link')).toBe(true)
    expect(isStandalonePublicRoute('/signup')).toBe(true)
  })

  it('does not match LMS shell routes or marketing legal pages', () => {
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
  afterEach(() => {
    document.documentElement.style.overflow = 'hidden'
  })

  it('enables document scroll on standalone public pages', () => {
    applyDocumentScrollMode('/trust')
    expect(document.documentElement.style.overflow).toBe('auto')
  })

  it('locks document scroll in the LMS shell', () => {
    applyDocumentScrollMode('/courses')
    expect(document.documentElement.style.overflow).toBe('hidden')
  })
})
