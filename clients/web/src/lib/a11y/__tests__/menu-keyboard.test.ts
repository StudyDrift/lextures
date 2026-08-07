import { describe, expect, it, vi } from 'vitest'
import { focusFirstMenuitem, handleMenuKeyDown } from '../menu-keyboard'

function buildMenu(labels: string[]): HTMLElement {
  const menu = document.createElement('div')
  menu.setAttribute('role', 'menu')
  for (const label of labels) {
    const item = document.createElement('button')
    item.setAttribute('role', 'menuitem')
    item.textContent = label
    menu.appendChild(item)
  }
  document.body.appendChild(menu)
  return menu
}

describe('handleMenuKeyDown', () => {
  it('moves with ArrowDown/ArrowUp and wraps', () => {
    document.body.innerHTML = ''
    const menu = buildMenu(['A', 'B', 'C'])
    const items = menu.querySelectorAll<HTMLElement>('[role="menuitem"]')
    items[0]!.focus()
    const typeahead = { buffer: '', at: 0 }
    const onClose = vi.fn()

    handleMenuKeyDown(
      {
        key: 'ArrowDown',
        ctrlKey: false,
        metaKey: false,
        altKey: false,
        currentTarget: menu,
        preventDefault: vi.fn(),
      },
      { onClose },
      typeahead,
    )
    expect(document.activeElement).toBe(items[1])

    handleMenuKeyDown(
      {
        key: 'ArrowUp',
        ctrlKey: false,
        metaKey: false,
        altKey: false,
        currentTarget: menu,
        preventDefault: vi.fn(),
      },
      { onClose },
      typeahead,
    )
    expect(document.activeElement).toBe(items[0])
  })

  it('Escape calls onClose', () => {
    document.body.innerHTML = ''
    const menu = buildMenu(['A'])
    const onClose = vi.fn()
    handleMenuKeyDown(
      {
        key: 'Escape',
        ctrlKey: false,
        metaKey: false,
        altKey: false,
        currentTarget: menu,
        preventDefault: vi.fn(),
      },
      { onClose },
      { buffer: '', at: 0 },
    )
    expect(onClose).toHaveBeenCalled()
  })
})

describe('focusFirstMenuitem', () => {
  it('focuses the first item', () => {
    document.body.innerHTML = ''
    const menu = buildMenu(['First', 'Second'])
    focusFirstMenuitem(menu)
    expect(document.activeElement?.textContent).toBe('First')
  })
})
