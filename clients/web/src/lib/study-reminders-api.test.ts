import { describe, expect, it } from 'vitest'
import { formatReminderTimeLabel } from './study-reminders-api'

describe('formatReminderTimeLabel', () => {
  it('formats HH:MM without throwing', () => {
    const label = formatReminderTimeLabel('19:00')
    expect(label.length).toBeGreaterThan(0)
  })
})
