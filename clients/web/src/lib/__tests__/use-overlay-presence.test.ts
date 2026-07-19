import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useOverlayPresence } from '../use-overlay-presence'

describe('useOverlayPresence', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    document.documentElement.classList.remove('reduced-motion')
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({
        matches: false,
        media: '',
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => true,
      })),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('keeps mounted through exit then unmounts', () => {
    const { result, rerender } = renderHook(
      ({ open }: { open: boolean }) =>
        useOverlayPresence({ open, kind: 'dialog', enabled: true }),
      { initialProps: { open: true } },
    )
    expect(result.current.mounted).toBe(true)
    expect(result.current.phase).toBe('open')

    rerender({ open: false })
    expect(result.current.phase).toBe('closing')
    expect(result.current.mounted).toBe(true)

    act(() => {
      vi.runAllTimers()
    })
    expect(result.current.phase).toBe('closed')
    expect(result.current.mounted).toBe(false)
  })

  it('re-opens mid-exit (AC-6)', () => {
    const { result, rerender } = renderHook(
      ({ open }: { open: boolean }) =>
        useOverlayPresence({ open, kind: 'dialog', enabled: true }),
      { initialProps: { open: true } },
    )
    rerender({ open: false })
    expect(result.current.phase).toBe('closing')
    rerender({ open: true })
    expect(result.current.phase).toBe('opening')
    expect(result.current.mounted).toBe(true)
    act(() => {
      vi.runAllTimers()
    })
    expect(result.current.phase).toBe('open')
  })

  it('calls onExitStart when closing begins', () => {
    const onExitStart = vi.fn()
    const { rerender } = renderHook(
      ({ open }: { open: boolean }) =>
        useOverlayPresence({ open, kind: 'dialog', enabled: true, onExitStart }),
      { initialProps: { open: true } },
    )
    rerender({ open: false })
    expect(onExitStart).toHaveBeenCalledTimes(1)
  })
})
