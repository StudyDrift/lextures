import { describe, it, expect, beforeEach, vi } from 'vitest'
import { announce } from '../announcer'

describe('announce()', () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <div id="a11y-polite-announcer" role="status" aria-live="polite" aria-atomic="true"></div>
      <div id="a11y-assertive-announcer" role="alert" aria-live="assertive" aria-atomic="true"></div>
    `
  })

  it('updates the polite live region', async () => {
    vi.useFakeTimers()
    announce('Page loaded')
    // text cleared immediately
    expect(document.getElementById('a11y-polite-announcer')!.textContent).toBe('')
    // rAF fires
    await vi.runAllTimersAsync()
    expect(document.getElementById('a11y-polite-announcer')!.textContent).toBe('Page loaded')
    vi.useRealTimers()
  })

  it('updates the assertive live region', async () => {
    vi.useFakeTimers()
    announce('Error occurred', 'assertive')
    await vi.runAllTimersAsync()
    expect(document.getElementById('a11y-assertive-announcer')!.textContent).toBe('Error occurred')
    vi.useRealTimers()
  })

  it('defaults to polite when no politeness specified', async () => {
    vi.useFakeTimers()
    announce('Default message')
    await vi.runAllTimersAsync()
    expect(document.getElementById('a11y-polite-announcer')!.textContent).toBe('Default message')
    expect(document.getElementById('a11y-assertive-announcer')!.textContent).toBe('')
    vi.useRealTimers()
  })

  it('does nothing when live region element is absent', () => {
    document.body.innerHTML = ''
    expect(() => announce('orphaned')).not.toThrow()
  })
})
