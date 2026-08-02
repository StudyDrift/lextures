import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ScheduleDatetimeField, RelativeScheduleBanner } from '../schedule-datetime-field'

describe('ScheduleDatetimeField', () => {
  it('renders datetime-local in fixed mode', () => {
    render(
      <ScheduleDatetimeField
        id="due"
        label="Due date"
        value=""
        onChange={() => {}}
        scheduleMode="fixed"
      />,
    )
    expect(screen.getByLabelText('Due date')).toHaveAttribute('type', 'datetime-local')
  })

  it('renders amount+unit controls in relative mode', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const anchor = new Date(2026, 0, 1, 0, 0, 0, 0).toISOString()
    render(
      <ScheduleDatetimeField
        id="due"
        label="Due date"
        relativeLabel="Due after enrollment"
        value=""
        onChange={onChange}
        scheduleMode="relative"
        relativeAnchorAt={anchor}
        defaultTime="23:59"
      />,
    )
    expect(screen.getByLabelText('Due after enrollment')).toHaveAttribute('type', 'number')
    expect(screen.getByLabelText('Due after enrollment unit')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Due after enrollment'), '7')
    expect(onChange).toHaveBeenCalled()
    const last = onChange.mock.calls.at(-1)?.[0] as string
    expect(last).toMatch(/T23:59$/)
  })

  it('falls back to datetime-local when relative without anchor', () => {
    render(
      <ScheduleDatetimeField
        id="due"
        label="Due date"
        value=""
        onChange={() => {}}
        scheduleMode="relative"
        relativeAnchorAt={null}
      />,
    )
    expect(screen.getByLabelText('Due date')).toHaveAttribute('type', 'datetime-local')
  })
})

describe('RelativeScheduleBanner', () => {
  it('only shows for relative mode with an anchor', () => {
    const { rerender } = render(
      <RelativeScheduleBanner scheduleMode="fixed" relativeAnchorAt="2026-01-01T00:00:00Z" />,
    )
    expect(screen.queryByText(/relative dates/i)).not.toBeInTheDocument()
    rerender(
      <RelativeScheduleBanner scheduleMode="relative" relativeAnchorAt="2026-01-01T00:00:00Z" />,
    )
    expect(screen.getByText(/relative dates/i)).toBeInTheDocument()
  })
})
