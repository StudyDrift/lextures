import { describe, expect, it } from 'vitest'
import { activityAssignmentItemId, sortAssignmentsByTitle } from '../activity-node-data'

describe('activity node data', () => {
  it('falls back to the current assignment when unset', () => {
    expect(activityAssignmentItemId({}, 'item-1')).toBe('item-1')
    expect(activityAssignmentItemId({ assignmentItemId: '' }, 'item-1')).toBe('item-1')
  })

  it('uses the stored assignment id when present', () => {
    expect(activityAssignmentItemId({ assignmentItemId: 'other-item' }, 'item-1')).toBe('other-item')
  })

  it('sorts assignments alphanumerically by title', () => {
    const sorted = sortAssignmentsByTitle([
      { id: '2', title: 'Lab 10' },
      { id: '1', title: 'Lab 2' },
      { id: '3', title: 'Essay' },
    ])
    expect(sorted.map((item) => item.title)).toEqual(['Essay', 'Lab 2', 'Lab 10'])
  })
})