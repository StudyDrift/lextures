import { StrictMode } from 'react'
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { StaggerReveal } from '../load-reveal'

type MediaQueryListStub = {
  matches: boolean
  media: string
  onchange: null
  addEventListener: (type: string, listener: () => void) => void
  removeEventListener: (type: string, listener: () => void) => void
  addListener: () => void
  removeListener: () => void
  dispatchEvent: () => boolean
}

function stubMatchMedia(matches: boolean) {
  const listeners = new Set<() => void>()
  const mql: MediaQueryListStub = {
    matches,
    media: '(prefers-reduced-motion: reduce)',
    onchange: null,
    addEventListener: (_type, listener) => {
      listeners.add(listener)
    },
    removeEventListener: (_type, listener) => {
      listeners.delete(listener)
    },
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  }
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => {
      mql.media = query
      return mql
    }),
  )
}

describe('StaggerReveal', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    stubMatchMedia(false)
  })

  afterEach(() => {
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('plays the entrance on first paint when ready (does not strand opacity 0)', () => {
    render(
      <StaggerReveal index={0}>
        <a href="/inbox">Inbox</a>
      </StaggerReveal>,
    )
    const root = screen.getByText('Inbox').parentElement
    expect(root).toHaveAttribute('data-lx-reveal', 'in')
    expect(root?.className).not.toMatch(/lx-stagger-reveal-pending/)
    expect(root?.className).toMatch(/lx-stagger-reveal/)
  })

  it('still reveals under React Strict Mode double-invoke', () => {
    render(
      <StrictMode>
        <StaggerReveal index={0}>
          <a href="/inbox">Inbox</a>
        </StaggerReveal>
      </StrictMode>,
    )
    const root = screen.getByText('Inbox').parentElement
    expect(root).toHaveAttribute('data-lx-reveal', 'in')
    expect(root?.className).not.toMatch(/lx-stagger-reveal-pending/)
  })

  it('stays pending until ready, then reveals without a cancelled rAF', () => {
    const { rerender } = render(
      <StaggerReveal index={0} ready={false}>
        <a href="/inbox">Inbox</a>
      </StaggerReveal>,
    )
    expect(screen.getByText('Inbox').parentElement).toHaveAttribute('data-lx-reveal', 'pending')

    rerender(
      <StaggerReveal index={0} ready>
        <a href="/inbox">Inbox</a>
      </StaggerReveal>,
    )
    expect(screen.getByText('Inbox').parentElement).toHaveAttribute('data-lx-reveal', 'in')
  })

  it('skips motion classes when the kill-switch is off', () => {
    render(
      <StaggerReveal index={0} enabled={false}>
        <a href="/inbox">Inbox</a>
      </StaggerReveal>,
    )
    const root = screen.getByText('Inbox').parentElement
    expect(root).toHaveAttribute('data-lx-reveal', 'off')
    expect(root?.className ?? '').not.toMatch(/lx-stagger-reveal/)
  })
})
