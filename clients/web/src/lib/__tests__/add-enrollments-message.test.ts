import { describe, expect, it } from 'vitest'
import { formatAddEnrollmentsMessage } from '../add-enrollments-message'

const KID_ID = 'a14762d8-c505-4cf3-8e60-2d4e40a9a003'

describe('formatAddEnrollmentsMessage', () => {
  it('names a managed learner instead of printing the raw user id', () => {
    const msg = formatAddEnrollmentsMessage({ added: [KID_ID] }, new Map([[KID_ID, 'QA Test Kid']]))
    expect(msg).toBe('Added QA Test Kid.')
    expect(msg).not.toContain(KID_ID)
  })

  it('names already-enrolled and not-found learners too', () => {
    const names = new Map([[KID_ID, 'QA Test Kid']])
    expect(formatAddEnrollmentsMessage({ alreadyEnrolled: [KID_ID], notFound: [KID_ID] }, names)).toBe(
      'Already enrolled: QA Test Kid. No account for: QA Test Kid.',
    )
  })

  it('never leaks a raw uuid when the learner is not in the loaded list', () => {
    expect(formatAddEnrollmentsMessage({ added: [KID_ID] })).toBe('Added a learner.')
  })

  it('echoes emails as typed', () => {
    expect(
      formatAddEnrollmentsMessage({ added: ['a@example.com', 'b@example.com'], notFound: ['c@example.com'] }),
    ).toBe('Added a@example.com, b@example.com. No account for: c@example.com.')
  })

  it('says Done when nothing changed', () => {
    expect(formatAddEnrollmentsMessage({})).toBe('Done.')
  })
})
