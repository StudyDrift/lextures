import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  initialsFromName,
  nameFieldsFromProfile,
  parseAccountProfile,
  profileName,
  shortcutHint,
} from '../top-bar-utils'

describe('nameFieldsFromProfile', () => {
  it('uses first and last name when present', () => {
    expect(
      nameFieldsFromProfile({
        firstName: 'Ada',
        lastName: 'Lovelace',
        displayName: 'Display Only',
      }),
    ).toEqual({ firstName: 'Ada', lastName: 'Lovelace' })
  })

  it('splits display name when first and last are empty', () => {
    expect(
      nameFieldsFromProfile({
        displayName: 'Chase Willden',
      }),
    ).toEqual({ firstName: 'Chase', lastName: 'Willden' })
  })

  it('puts a single-word display name in first name', () => {
    expect(nameFieldsFromProfile({ displayName: 'Madonna' })).toEqual({
      firstName: 'Madonna',
      lastName: '',
    })
  })
})

describe('profileName', () => {
  it('returns Profile when profile is null', () => {
    expect(profileName(null)).toBe('Profile')
  })

  it('prefers first + last name', () => {
    expect(
      profileName({
        email: 'a@b.com',
        firstName: '  Ada ',
        lastName: ' Lovelace ',
      }),
    ).toBe('Ada Lovelace')
  })

  it('falls back to display name then email', () => {
    expect(
      profileName({
        email: 'only@example.com',
        displayName: 'Display',
      }),
    ).toBe('Display')
    expect(profileName({ email: 'e@mail.test' })).toBe('e@mail.test')
  })
})

describe('parseAccountProfile', () => {
  it('reads avatarUrl and snake_case avatar_url', () => {
    expect(
      parseAccountProfile({
        email: 'a@b.com',
        firstName: 'Ada',
        avatarUrl: 'https://example.com/a.png',
      }),
    ).toEqual({
      email: 'a@b.com',
      displayName: null,
      firstName: 'Ada',
      lastName: null,
      avatarUrl: 'https://example.com/a.png',
    })
    expect(
      parseAccountProfile({
        email: 'a@b.com',
        avatar_url: ' data:image/png;base64,abc ',
      })?.avatarUrl,
    ).toBe('data:image/png;base64,abc')
  })

  it('returns null for missing email', () => {
    expect(parseAccountProfile({ avatarUrl: 'https://x.test/a.png' })).toBeNull()
  })
})

describe('initialsFromName', () => {
  it('returns U for empty or whitespace', () => {
    expect(initialsFromName('')).toBe('U')
    expect(initialsFromName('   ')).toBe('U')
  })

  it('uses one letter for a single word', () => {
    expect(initialsFromName('Madonna')).toBe('M')
  })

  it('uses first letters of first two words', () => {
    expect(initialsFromName('Jean-Luc Picard')).toBe('JP')
  })
})

describe('shortcutHint', () => {
  const nav = globalThis.navigator

  afterEach(() => {
    vi.stubGlobal('navigator', nav)
  })

  it('returns Ctrl+K for non-Apple platforms', () => {
    vi.stubGlobal('navigator', {
      ...nav,
      platform: 'Win32',
      userAgent: 'Mozilla/5.0 Windows',
    })
    expect(shortcutHint()).toBe('Ctrl+K')
  })

  it('returns ⌘K for Mac-like platforms', () => {
    vi.stubGlobal('navigator', {
      ...nav,
      platform: 'MacIntel',
      userAgent: 'Mozilla/5.0 Macintosh',
    })
    expect(shortcutHint()).toBe('⌘K')
  })
})
