import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useRef, useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { Menu } from '../menu'
import { menuPositionStyle } from '../menu-position'

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

  it('clamps a right-edge trigger so the open menu stays in the viewport', () => {
    const viewport = { width: 1024, height: 768 }
    vi.stubGlobal('innerWidth', viewport.width)
    vi.stubGlobal('innerHeight', viewport.height)
    const originalOffset = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetWidth')
    const originalHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetHeight')
    Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
      configurable: true,
      get() {
        return (this as HTMLElement).getAttribute('role') === 'menu' ? 220 : 90
      },
    })
    Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
      configurable: true,
      get() {
        return (this as HTMLElement).getAttribute('role') === 'menu' ? 260 : 32
      },
    })
    const originalRect = HTMLElement.prototype.getBoundingClientRect
    HTMLElement.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if ((this as HTMLElement).getAttribute('role') === 'menu') {
        return DOMRect.fromRect({ x: 910, y: 76, width: 220, height: 260 })
      }
      return DOMRect.fromRect({ x: 910, y: 40, width: 90, height: 32 })
    }
    try {
      render(<Harness />)
      const menu = screen.getByRole('menu')
      const left = Number.parseFloat(menu.style.left)
      expect(left).toBeGreaterThanOrEqual(8)
      expect(left + 220).toBeLessThanOrEqual(viewport.width - 8)
    } finally {
      vi.unstubAllGlobals()
      HTMLElement.prototype.getBoundingClientRect = originalRect
      if (originalOffset) Object.defineProperty(HTMLElement.prototype, 'offsetWidth', originalOffset)
      if (originalHeight) Object.defineProperty(HTMLElement.prototype, 'offsetHeight', originalHeight)
    }
  })

  it('flips to the start/end that keeps the menu on screen', () => {
    const viewport = { width: 1024, height: 768 }
    const nearRight = { top: 40, right: 1000, bottom: 72, left: 910, width: 90 }
    const menu = { width: 200, height: 240 }

    const startNearRight = menuPositionStyle(nearRight, menu, 'bottom-start', viewport)
    expect(startNearRight.left).toBeGreaterThanOrEqual(8)
    expect(Number(startNearRight.left) + menu.width).toBeLessThanOrEqual(viewport.width - 8)

    const endNearRight = menuPositionStyle(nearRight, menu, 'bottom-end', viewport)
    expect(endNearRight.left).toBe(800)
    expect(endNearRight.right).toBeUndefined()
  })

  it('flips above the trigger when there is no room below', () => {
    const viewport = { width: 1024, height: 400 }
    const nearBottom = { top: 320, right: 200, bottom: 352, left: 40, width: 160 }
    const menu = { width: 180, height: 220 }
    const style = menuPositionStyle(nearBottom, menu, 'bottom-start', viewport)
    expect(Number(style.top) + menu.height).toBeLessThanOrEqual(viewport.height - 8)
    expect(Number(style.top)).toBeLessThan(nearBottom.top)
  })
})
