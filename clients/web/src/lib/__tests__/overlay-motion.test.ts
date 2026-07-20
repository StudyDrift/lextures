import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { distances, durations } from '../motion'
import {
  dialogEnterKeyframes,
  dialogExitKeyframes,
  overlayClassNames,
  overlayDurationMs,
  overlayEntered,
  overlayMounted,
  OVERLAY_SHEET_DISMISS_THRESHOLD,
  shouldDismissSheetDrag,
  transitionOverlay,
  type OverlayPhase,
} from '../overlay-motion'

describe('transitionOverlay', () => {
  it('walks closed → opening → open → closing → closed', () => {
    let phase: OverlayPhase = 'closed'
    phase = transitionOverlay(phase, 'requestOpen')
    expect(phase).toBe('opening')
    phase = transitionOverlay(phase, 'enterComplete')
    expect(phase).toBe('open')
    phase = transitionOverlay(phase, 'requestClose')
    expect(phase).toBe('closing')
    phase = transitionOverlay(phase, 'exitComplete')
    expect(phase).toBe('closed')
  })

  it('re-opens mid-exit without stranding (AC-6)', () => {
    let phase: OverlayPhase = 'closing'
    phase = transitionOverlay(phase, 'requestOpen')
    expect(phase).toBe('opening')
    expect(overlayMounted(phase)).toBe(true)
    expect(overlayEntered(phase)).toBe(true)
  })

  it('closes mid-enter', () => {
    expect(transitionOverlay('opening', 'requestClose')).toBe('closing')
  })

  it('is idempotent for duplicate open/close', () => {
    expect(transitionOverlay('open', 'requestOpen')).toBe('open')
    expect(transitionOverlay('closing', 'requestClose')).toBe('closing')
    expect(transitionOverlay('closed', 'requestClose')).toBe('closed')
  })
})

describe('overlayDurationMs / classNames', () => {
  beforeEach(() => {
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
    document.documentElement.classList.remove('reduced-motion')
    vi.unstubAllGlobals()
  })

  it('uses bubble enter / exit durations for dialogs', () => {
    expect(
      overlayDurationMs({ kind: 'dialog', exiting: false, reduceMotion: false }),
    ).toBe(durations.base)
    expect(
      overlayDurationMs({ kind: 'dialog', exiting: true, reduceMotion: false }),
    ).toBe(durations.fast)
  })

  it('fades only under reduced motion (FR-7 / AC-5)', () => {
    expect(
      overlayDurationMs({ kind: 'drawer', exiting: false, reduceMotion: true }),
    ).toBeLessThanOrEqual(100)
    const classes = overlayClassNames({
      kind: 'dialog',
      phase: 'opening',
      reduceMotion: true,
    })
    expect(classes.panel).toContain('lx-overlay-fade-in')
    expect(classes.panel).not.toContain('dialog-in')
  })

  it('disables motion when kill-switch is off', () => {
    expect(
      overlayDurationMs({ kind: 'dialog', exiting: false, enabled: false }),
    ).toBe(0)
    const classes = overlayClassNames({
      kind: 'dialog',
      phase: 'open',
      enabled: false,
    })
    expect(classes.durationMs).toBe(0)
  })

  it('assigns edge-aware sheet classes', () => {
    const end = overlayClassNames({
      kind: 'drawer',
      phase: 'opening',
      edge: 'end',
      reduceMotion: false,
    })
    expect(end.panel).toBe('lx-overlay-sheet-end-in')
    const bottom = overlayClassNames({
      kind: 'sheet',
      phase: 'closing',
      edge: 'bottom',
      reduceMotion: false,
    })
    expect(bottom.panel).toBe('lx-overlay-sheet-bottom-out')
  })
})

describe('dialog keyframes', () => {
  it('scales from enterScaleFrom on enter and exit', () => {
    const enter = dialogEnterKeyframes()
    expect(enter[0]?.transform).toContain(`scale(${distances.enterScaleFrom})`)
    expect(enter[1]?.opacity).toBe(1)
    const exit = dialogExitKeyframes()
    expect(exit[1]?.transform).toContain(`scale(${distances.enterScaleFrom})`)
  })
})

describe('shouldDismissSheetDrag', () => {
  it('dismisses past threshold or with high velocity (AC-2)', () => {
    expect(OVERLAY_SHEET_DISMISS_THRESHOLD).toBeGreaterThan(0.2)
    expect(shouldDismissSheetDrag(100, 400)).toBe(false)
    expect(shouldDismissSheetDrag(120, 400)).toBe(true)
    expect(shouldDismissSheetDrag(10, 400, 0.9)).toBe(true)
    expect(shouldDismissSheetDrag(10, 0)).toBe(false)
  })
})
