import { afterEach, describe, expect, it, vi } from 'vitest'
import { tutorMenuBlockReason, fetchTutorMenuBlocked } from '../tutor-api'

vi.mock('../api', () => ({
  authorizedFetch: (input: string, init?: RequestInit) => globalThis.fetch(input, init),
}))

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('tutorMenuBlockReason', () => {
  it('hides the tutor when the account opted out of the tutor or of AI processing', () => {
    expect(tutorMenuBlockReason(true, false)).toBe('tutor')
    expect(tutorMenuBlockReason(false, true)).toBe('processing')
    expect(tutorMenuBlockReason(null, true)).toBe('processing')
    expect(tutorMenuBlockReason(false, false)).toBeNull()
    expect(tutorMenuBlockReason(null, false)).toBeNull()
  })
})

describe('fetchTutorMenuBlocked', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('blocks when AI processing is off even if the tutor opt-out is false', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.endsWith('/ai-tutor-opt-out')) return json({ aiTutorOptOut: false })
      if (url.endsWith('/ai-opt-out')) return json({ aiProcessingOptOut: true })
      return json({}, 404)
    })
    await expect(fetchTutorMenuBlocked()).resolves.toBe(true)
  })

  it('allows the tutor when neither opt-out is set', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.endsWith('/ai-tutor-opt-out')) return json({ aiTutorOptOut: false })
      if (url.endsWith('/ai-opt-out')) return json({ aiProcessingOptOut: false })
      return json({}, 404)
    })
    await expect(fetchTutorMenuBlocked()).resolves.toBe(false)
  })
})
