import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useRef, useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { Menu } from '../menu'

function Harness({ onProfile }: { onProfile?: () => void }) {
  const [open, setOpen] = useState(true)
  const anchorRef = useRef<HTMLButtonElement>(null)
  return (
    <>
      <button ref={anchorRef} type="button">
        Open
      </button>
      <Menu
        open={open}
        onOpenChange={setOpen}
        anchorRef={anchorRef}
        aria-label="Demo menu"
        items={[
          { id: 'a', label: 'Alpha', textValue: 'Alpha', onSelect: onProfile },
          { id: 'b', label: 'Bravo', textValue: 'Bravo' },
          { id: 'c', label: 'Charlie', textValue: 'Charlie' },
        ]}
      />
    </>
  )
}

describe('UX.4 Menu', () => {
  it('focuses first item on open and moves with arrows (AC-3)', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    const first = await screen.findByRole('menuitem', { name: 'Alpha' })
    expect(first).toHaveFocus()
    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('menuitem', { name: 'Bravo' })).toHaveFocus()
    await user.keyboard('{End}')
    expect(screen.getByRole('menuitem', { name: 'Charlie' })).toHaveFocus()
    await user.keyboard('{Home}')
    expect(screen.getByRole('menuitem', { name: 'Alpha' })).toHaveFocus()
  })

  it('typeahead jumps to matching item', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await screen.findByRole('menuitem', { name: 'Alpha' })
    await user.keyboard('c')
    expect(screen.getByRole('menuitem', { name: 'Charlie' })).toHaveFocus()
  })

  it('Escape closes and returns focus to trigger', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await screen.findByRole('menuitem', { name: 'Alpha' })
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu')).toBeNull()
    expect(screen.getByRole('button', { name: 'Open' })).toHaveFocus()
  })

  it('Enter activates item', async () => {
    const user = userEvent.setup()
    const onProfile = vi.fn()
    render(<Harness onProfile={onProfile} />)
    await screen.findByRole('menuitem', { name: 'Alpha' })
    await user.keyboard('{Enter}')
    expect(onProfile).toHaveBeenCalled()
  })
})
