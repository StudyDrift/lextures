import { describe, expect, it } from 'vitest'
import { canvasAccessTokenSettingsUrl } from '../canvas-url'

describe('canvasAccessTokenSettingsUrl', () => {
  it('returns null for empty input', () => {
    expect(canvasAccessTokenSettingsUrl('')).toBeNull()
    expect(canvasAccessTokenSettingsUrl('   ')).toBeNull()
  })

  it('builds settings URL from a full https base URL', () => {
    expect(canvasAccessTokenSettingsUrl('https://example.instructure.com')).toBe(
      'https://example.instructure.com/profile/settings',
    )
  })

  it('strips trailing slashes and extra path segments', () => {
    expect(canvasAccessTokenSettingsUrl('https://example.instructure.com/')).toBe(
      'https://example.instructure.com/profile/settings',
    )
    expect(canvasAccessTokenSettingsUrl('https://example.instructure.com/courses/123')).toBe(
      'https://example.instructure.com/profile/settings',
    )
  })

  it('adds https when the scheme is omitted', () => {
    expect(canvasAccessTokenSettingsUrl('example.instructure.com')).toBe(
      'https://example.instructure.com/profile/settings',
    )
  })

  it('returns null for invalid URLs', () => {
    expect(canvasAccessTokenSettingsUrl('not a url')).toBeNull()
    expect(canvasAccessTokenSettingsUrl('https://')).toBeNull()
  })
})