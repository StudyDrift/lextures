import { describe, expect, it } from 'vitest'
import { weekdayColumnForDue } from '../student-todo-utils'

describe('weekdayColumnForDue', () => {
  it('maps a Tuesday 9:15 due date to Monday with default PFT', () => {
    // Tuesday 2026-06-30 09:15 local
    const dueAt = new Date(2026, 5, 30, 9, 15, 0, 0).toISOString()
    expect(weekdayColumnForDue(dueAt)).toBe('mon')
  })

  it('maps afternoon due dates to the previous weekday', () => {
    const dueAt = new Date(2026, 5, 30, 15, 0, 0, 0).toISOString()
    expect(weekdayColumnForDue(dueAt)).toBe('mon')
  })

  it('uses the current weekday when no due date is set', () => {
    // Friday 2026-06-26
    const now = new Date(2026, 5, 26, 12, 0, 0, 0)
    expect(weekdayColumnForDue(null, now)).toBe('fri')
  })
})