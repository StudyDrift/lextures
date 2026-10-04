import { describe, expect, it } from 'vitest'
import { isApiErrorPayloadTitle } from '../structure-title'

describe('isApiErrorPayloadTitle', () => {
  it('rejects the reorder error body and allows real page names', () => {
    expect(
      isApiErrorPayloadTitle(
        '{"error":{"code":"INTERNAL","message":"Failed to reorder course structure."}}',
      ),
    ).toBe(true)
    expect(isApiErrorPayloadTitle('  {"error":{"code":"INVALID_INPUT","message":"Nope."}}\n')).toBe(
      true,
    )
    expect(isApiErrorPayloadTitle('What Makes a Strong AI-Augmented Workflow')).toBe(false)
    expect(isApiErrorPayloadTitle('Failed to reorder course structure.')).toBe(false)
  })
})
