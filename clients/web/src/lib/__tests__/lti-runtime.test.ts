import { describe, expect, it } from 'vitest'
import { ltiLaunchErrorMessage, ltiNotEnabledMessage } from '../lti-runtime'

describe('ltiLaunchErrorMessage', () => {
  it('replaces the disabled-runtime error with a plain sentence', () => {
    expect(ltiLaunchErrorMessage(new Error('LTI is not enabled on this server.'))).toBe(
      ltiNotEnabledMessage,
    )
  })

  it('keeps other server messages', () => {
    expect(ltiLaunchErrorMessage(new Error('External tool is not active.'))).toBe(
      'External tool is not active.',
    )
  })

  it('falls back when the error has no message', () => {
    expect(ltiLaunchErrorMessage(new Error('Request failed'))).toBe('Could not load this LTI link.')
    expect(ltiLaunchErrorMessage('nope')).toBe('Could not load this LTI link.')
  })
})
