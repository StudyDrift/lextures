import { describe, expect, it } from 'vitest'
import {
  parseLegacyFieldErrors,
  parseValidationErrorResponse,
  readFieldAddressableErrors,
} from '../parse-validation-error'
import { readApiErrorMessage, readApiFieldErrors } from '../../errors'

describe('validation error parsing (UX.6)', () => {
  it('parses the validation_failed envelope', () => {
    const raw = {
      error: 'validation_failed',
      message: 'Fix the highlighted fields',
      fields: [
        { path: 'phoneNumber', code: 'invalid_phone', message: 'Enter a valid phone number.' },
      ],
    }
    const parsed = parseValidationErrorResponse(raw)
    expect(parsed?.fields).toHaveLength(1)
    expect(parsed?.fields[0]?.path).toBe('phoneNumber')
    expect(readApiErrorMessage(raw)).toBe('Fix the highlighted fields')
    expect(readApiFieldErrors(raw)).toHaveLength(1)
  })

  it('returns null for legacy nested error shape', () => {
    const raw = { error: { code: 'INVALID_INPUT', message: 'Bad request' } }
    expect(parseValidationErrorResponse(raw)).toBeNull()
    expect(readApiErrorMessage(raw)).toBe('Bad request')
    expect(readApiFieldErrors(raw)).toEqual([])
  })

  it('parses legacy content-tools field errors', () => {
    const raw = {
      error: {
        code: 'VALIDATION',
        message: 'Invalid',
        errors: [{ path: 'title', message: 'Required' }],
      },
    }
    expect(parseLegacyFieldErrors(raw)).toEqual([
      { path: 'title', code: 'custom', message: 'Required' },
    ])
    expect(readFieldAddressableErrors(raw).fields).toHaveLength(1)
  })
})
