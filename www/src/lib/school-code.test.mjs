/**
 * Unit tests for school-code helpers used by /get-started.
 * Mirrors the TypeScript module logic for Node's test runner.
 */
import assert from 'node:assert/strict'
import { describe, it, mock } from 'node:test'

const SCHOOL_CODE_PATTERN = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/

const RESERVED_SCHOOL_CODES = new Set([
  'admin',
  'api',
  'app',
  'default',
  'demo',
  'login',
  'magic-link',
  'mfa',
  'self',
  'signup',
  'www',
])

function normalizeSchoolCode(raw) {
  return raw.trim().toLowerCase()
}

function schoolCodeError(code) {
  const normalized = normalizeSchoolCode(code)
  if (!normalized) return 'Enter your school code.'
  if (normalized.length < 2) return 'School codes are at least 2 characters.'
  if (normalized.length > 32) return 'School codes are at most 32 characters.'
  if (!SCHOOL_CODE_PATTERN.test(normalized)) {
    return 'Use lowercase letters, numbers, and hyphens only.'
  }
  if (RESERVED_SCHOOL_CODES.has(normalized)) {
    return 'That code is reserved.'
  }
  return null
}

function isValidSchoolCode(code) {
  return schoolCodeError(code) === null
}

const API_BASE = 'https://self.lextures.com'

function schoolLookupUrl(code) {
  const slug = normalizeSchoolCode(code)
  return `${API_BASE}/api/v1/public/orgs/by-slug/${encodeURIComponent(slug)}`
}

async function lookupSchoolCode(code, fetchImpl = fetch) {
  if (!isValidSchoolCode(code)) {
    return { ok: false, reason: 'invalid' }
  }
  const normalized = normalizeSchoolCode(code)
  try {
    const res = await fetchImpl(schoolLookupUrl(normalized), {
      headers: { Accept: 'application/json' },
    })
    if (res.status === 404) {
      return { ok: false, reason: 'not_found' }
    }
    if (!res.ok) {
      return { ok: false, reason: 'unreachable' }
    }
    const raw = await res.json().catch(() => ({}))
    return {
      ok: true,
      slug: raw.slug?.trim() || normalized,
      name: raw.name?.trim() || normalized,
    }
  } catch {
    return { ok: false, reason: 'unreachable' }
  }
}

describe('school-code validation', () => {
  it('normalizes and validates codes', () => {
    assert.equal(normalizeSchoolCode('  Green-Valley '), 'green-valley')
    assert.equal(isValidSchoolCode('green-valley'), true)
    assert.equal(isValidSchoolCode('ab'), true)
  })

  it('rejects invalid formats and reserved codes', () => {
    assert.match(schoolCodeError(''), /Enter/)
    assert.match(schoolCodeError('a'), /at least 2/)
    assert.match(schoolCodeError('Bad_Code'), /lowercase/)
    assert.match(schoolCodeError('www'), /reserved/)
    assert.match(schoolCodeError('self'), /reserved/)
  })

  it('builds lookup URL for the public org-by-slug API', () => {
    assert.equal(
      schoolLookupUrl('Example-U'),
      'https://self.lextures.com/api/v1/public/orgs/by-slug/example-u',
    )
  })
})

describe('lookupSchoolCode', () => {
  it('returns not_found on 404', async () => {
    const fetchImpl = mock.fn(async () => ({
      status: 404,
      ok: false,
      json: async () => ({ message: 'Organization not found.' }),
    }))
    const result = await lookupSchoolCode('example', fetchImpl)
    assert.deepEqual(result, { ok: false, reason: 'not_found' })
    assert.equal(fetchImpl.mock.callCount(), 1)
  })

  it('returns ok with slug/name on 200', async () => {
    const fetchImpl = mock.fn(async () => ({
      status: 200,
      ok: true,
      json: async () => ({ slug: 'green-valley', name: 'Green Valley USD' }),
    }))
    const result = await lookupSchoolCode('green-valley', fetchImpl)
    assert.deepEqual(result, {
      ok: true,
      slug: 'green-valley',
      name: 'Green Valley USD',
    })
  })

  it('returns unreachable on network failure', async () => {
    const fetchImpl = mock.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    const result = await lookupSchoolCode('green-valley', fetchImpl)
    assert.deepEqual(result, { ok: false, reason: 'unreachable' })
  })

  it('skips network for invalid codes', async () => {
    const fetchImpl = mock.fn(async () => ({ status: 200, ok: true, json: async () => ({}) }))
    const result = await lookupSchoolCode('www', fetchImpl)
    assert.deepEqual(result, { ok: false, reason: 'invalid' })
    assert.equal(fetchImpl.mock.callCount(), 0)
  })
})
