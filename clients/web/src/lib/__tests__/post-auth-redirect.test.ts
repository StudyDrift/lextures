import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setCachedAccountTypeFromUser } from '../auth'
import {
  authHandoffHref,
  pickPostAuthPath,
  returnPathFromAuthLocation,
  sanitizeReturnPath,
} from '../post-auth-redirect'

function memoryStorage(): Storage {
  const store = new Map<string, string>()
  return {
    get length() {
      return store.size
    },
    clear: () => store.clear(),
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    key: (i: number) => [...store.keys()][i] ?? null,
    removeItem: (k: string) => {
      store.delete(k)
    },
    setItem: (k: string, v: string) => {
      store.set(k, String(v))
    },
  } as Storage
}

describe('post-auth-redirect', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', memoryStorage())
  })

  afterEach(() => {
    setCachedAccountTypeFromUser(undefined)
    vi.unstubAllGlobals()
  })

  it('keeps in-app paths including query strings', () => {
    expect(sanitizeReturnPath('/marketplace/ai-essentials-c-hupcnf')).toBe(
      '/marketplace/ai-essentials-c-hupcnf',
    )
    expect(sanitizeReturnPath('/marketplace/demo?coupon=LAUNCH25&ref=www')).toBe(
      '/marketplace/demo?coupon=LAUNCH25&ref=www',
    )
  })

  it('rejects open redirects and auth-loop paths', () => {
    expect(sanitizeReturnPath('https://evil.example')).toBe('/')
    expect(sanitizeReturnPath('//evil.example')).toBe('/')
    expect(sanitizeReturnPath('/\\evil.example')).toBe('/')
    expect(sanitizeReturnPath('/login')).toBe('/')
    expect(sanitizeReturnPath('/signup?next=/marketplace/x')).toBe('/')
    expect(sanitizeReturnPath('/onboarding')).toBe('/')
  })

  it('prefers location state over next query', () => {
    expect(
      returnPathFromAuthLocation({
        search: '?next=%2Fcourses',
        state: { from: '/marketplace/ai-essentials-c-hupcnf' },
      }),
    ).toBe('/marketplace/ai-essentials-c-hupcnf')
    expect(returnPathFromAuthLocation({ search: '?next=%2Fmarketplace%2Fdemo', state: null })).toBe(
      '/marketplace/demo',
    )
  })

  it('builds a login handoff that survives refresh', () => {
    expect(authHandoffHref('/login', '/marketplace/demo?coupon=X')).toBe(
      '/login?next=%2Fmarketplace%2Fdemo%3Fcoupon%3DX',
    )
    expect(authHandoffHref('/login', '/login')).toBe('/login')
  })

  it('sends parent accounts to /parent unless the destination is already a parent path', () => {
    setCachedAccountTypeFromUser({ accountType: 'parent' })
    expect(pickPostAuthPath('/marketplace/demo')).toBe('/parent')
    expect(pickPostAuthPath('/parent/conferences')).toBe('/parent/conferences')
  })
})
