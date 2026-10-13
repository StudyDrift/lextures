import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { KidRecipientPicker } from '../kid-recipient-picker'

describe('KidRecipientPicker (#775)', () => {
  it('offers only the allowed recipients and no free-text address', () => {
    const onChange = vi.fn()
    render(
      <KidRecipientPicker
        recipients={[
          { id: 'p1', displayName: 'Mom', kind: 'parent' },
          { id: 't1', displayName: 'Ms. Lee', kind: 'teacher', courseTitles: ['Science'] },
        ]}
        value="p1"
        onChange={onChange}
      />,
    )
    expect(screen.getAllByRole('radio')).toHaveLength(2)
    expect(screen.getByLabelText('Mom (parent)')).toBeChecked()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Ms. Lee (teacher, Science)'))
    expect(onChange).toHaveBeenCalledWith('t1')
  })

  it('explains when nobody is set up yet', () => {
    render(<KidRecipientPicker recipients={[]} value="" onChange={() => undefined} />)
    expect(screen.getByText(/ask your parent to set up messaging/i)).toBeInTheDocument()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
  })
})
