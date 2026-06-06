import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SlotPicker } from '../slot-picker'
import type { ConferenceSlot } from '../../../lib/conferences-api'

const slots: ConferenceSlot[] = [
  {
    id: 'slot-1',
    availabilityId: 'av-1',
    startAt: '2025-11-18T16:00:00Z',
    endAt: '2025-11-18T16:15:00Z',
    status: 'open',
  },
  {
    id: 'slot-2',
    availabilityId: 'av-1',
    startAt: '2025-11-18T16:20:00Z',
    endAt: '2025-11-18T16:35:00Z',
    status: 'booked',
  },
]

describe('SlotPicker', () => {
  it('announces available slots for screen readers and supports keyboard booking', async () => {
    const user = userEvent.setup()
    const onBook = vi.fn()
    render(
      <SlotPicker slots={slots} teacherName="Ms. Smith" onBook={onBook} myBookedSlotId={null} />,
    )

    const openSlot = screen.getByRole('gridcell', {
      name: /Nov 18.*Ms\. Smith — available, press Enter to book/i,
    })
    expect(openSlot).toBeEnabled()
    await user.click(openSlot)
    expect(onBook).toHaveBeenCalledWith(slots[0])

    const bookedSlot = screen.getByRole('gridcell', {
      name: /Nov 18.*Ms\. Smith — booked/i,
    })
    expect(bookedSlot).toBeDisabled()
  })
})
