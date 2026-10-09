import { describe, expect, it } from 'vitest'
import { messageFromApiErrorBody, readApiErrorMessage } from '../errors'

describe('readApiErrorMessage', () => {
  it('reads nested API error message', () => {
    expect(
      readApiErrorMessage({
        error: { code: 'EMAIL_TAKEN', message: 'This email is already registered.' },
      }),
    ).toBe('This email is already registered.')
  })

  it('falls back to top-level message', () => {
    expect(readApiErrorMessage({ message: 'Bad request' })).toBe('Bad request')
  })

  it('returns a generic label when shape is unknown', () => {
    expect(readApiErrorMessage({})).toBe('Request failed')
  })
})

describe('messageFromApiErrorBody', () => {
  it('reads the JSON error envelope instead of the raw body', () => {
    const raw = JSON.stringify({
      error: { code: 'FORBIDDEN', message: 'AI processing is disabled for this account.' },
    })
    expect(messageFromApiErrorBody(raw, 'Error 403')).toBe(
      'AI processing is disabled for this account.',
    )
  })

  it('keeps plain-text bodies and uses the fallback when empty', () => {
    expect(messageFromApiErrorBody('tutor unavailable', 'Error 500')).toBe('tutor unavailable')
    expect(messageFromApiErrorBody('  ', 'Error 500')).toBe('Error 500')
    expect(messageFromApiErrorBody('{}', 'Error 403')).toBe('Error 403')
  })
})
