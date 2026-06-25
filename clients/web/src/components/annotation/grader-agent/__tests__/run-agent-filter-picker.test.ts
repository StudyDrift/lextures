import { describe, expect, it } from 'vitest'
import { defaultRunAgentFilterState, runFilterFromState } from '../run-agent-filter-picker'

describe('runFilterFromState', () => {
  it('returns undefined for all-course target', () => {
    expect(runFilterFromState(defaultRunAgentFilterState)).toBeUndefined()
  })

  it('maps section target', () => {
    expect(
      runFilterFromState({
        ...defaultRunAgentFilterState,
        target: 'section',
        sectionId: 'sec-1',
      }),
    ).toEqual({ sectionId: 'sec-1' })
  })

  it('maps selected submissions', () => {
    expect(
      runFilterFromState({
        ...defaultRunAgentFilterState,
        target: 'selected',
        selectedSubmissionIds: ['a', 'b'],
      }),
    ).toEqual({ submissionIds: ['a', 'b'] })
  })
})
