import { describe, it, expect, beforeEach } from 'vitest'
import { createFocusTrap, resolveFocusRestoreTarget } from '../focus-trap'

function buildContainer(html: string): HTMLElement {
  const div = document.createElement('div')
  div.innerHTML = html
  document.body.appendChild(div)
  return div
}

describe('createFocusTrap()', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('moves focus to the first focusable element on activate', () => {
    const container = buildContainer('<button id="b1">First</button><button id="b2">Second</button>')
    const trap = createFocusTrap(container)
    trap.activate()
    expect(document.activeElement?.id).toBe('b1')
    trap.deactivate()
  })

  it('restores focus to the previously focused element on deactivate', () => {
    const outside = document.createElement('button')
    outside.id = 'outside'
    document.body.appendChild(outside)
    outside.focus()

    const container = buildContainer('<button id="inner">Inner</button>')
    const trap = createFocusTrap(container)
    trap.activate()
    expect(document.activeElement?.id).toBe('inner')
    trap.deactivate()
    expect(document.activeElement?.id).toBe('outside')
  })

  it('wraps Tab from last to first element', () => {
    const container = buildContainer('<button id="b1">A</button><button id="b2">B</button>')
    const trap = createFocusTrap(container)
    trap.activate()

    const b2 = document.getElementById('b2')!
    b2.focus()

    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
    document.dispatchEvent(tabEvent)

    expect(document.activeElement?.id).toBe('b1')
    trap.deactivate()
  })

  it('wraps Shift+Tab from first to last element', () => {
    const container = buildContainer('<button id="b1">A</button><button id="b2">B</button>')
    const trap = createFocusTrap(container)
    trap.activate()

    const b1 = document.getElementById('b1')!
    b1.focus()

    const shiftTabEvent = new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
    })
    document.dispatchEvent(shiftTabEvent)

    expect(document.activeElement?.id).toBe('b2')
    trap.deactivate()
  })

  it('does not throw when container has no focusable children', () => {
    const container = buildContainer('<p>No focusable</p>')
    const trap = createFocusTrap(container)
    expect(() => trap.activate()).not.toThrow()
    trap.deactivate()
  })

  it('falls back to main when trigger unmounts (UX.4 restore chain)', () => {
    const main = document.createElement('main')
    main.id = 'main-content'
    document.body.appendChild(main)

    const trigger = document.createElement('button')
    trigger.id = 'trigger'
    document.body.appendChild(trigger)
    trigger.focus()

    const container = buildContainer('<button id="inner">Inner</button>')
    const trap = createFocusTrap(container)
    trap.activate()
    expect(document.activeElement?.id).toBe('inner')

    // Simulate trigger unmount (e.g. deleted row).
    trigger.remove()
    trap.deactivate()
    expect(document.activeElement?.id).toBe('main-content')
  })
})

describe('resolveFocusRestoreTarget()', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('returns preferred when still connected', () => {
    const btn = document.createElement('button')
    document.body.appendChild(btn)
    expect(resolveFocusRestoreTarget(btn)).toBe(btn)
  })

  it('returns main when preferred is detached', () => {
    const main = document.createElement('main')
    main.id = 'main-content'
    document.body.appendChild(main)
    const orphan = document.createElement('button')
    expect(resolveFocusRestoreTarget(orphan)).toBe(main)
  })
})
