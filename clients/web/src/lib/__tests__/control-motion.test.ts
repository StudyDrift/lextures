import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { distances, durations } from '../motion'
import {
  CONTROL_PRESS_SCALE,
  controlMotionClass,
  controlMotionDurationMs,
  hapticMapping,
  indicatorOffsetPx,
  indicatorTransition,
  loadingButtonState,
  pressClassName,
  shouldPressScale,
  shouldSlideIndicator,
  shouldValidationShake,
  validationClassName,
} from '../control-motion'

describe('AN.6 control motion helpers', () => {
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

  it('press scale matches AN.1 token and honors reduced-motion / kill-switch (FR-1 / AC-5)', () => {
    expect(CONTROL_PRESS_SCALE).toBe(distances.pressScale)
    expect(shouldPressScale({ enabled: true, reduceMotion: false })).toBe(true)
    expect(shouldPressScale({ enabled: true, reduceMotion: true })).toBe(false)
    expect(shouldPressScale({ enabled: false, reduceMotion: false })).toBe(false)
    expect(pressClassName({ enabled: true, reduceMotion: false })).toBe(controlMotionClass.pressable)
    expect(pressClassName({ enabled: true, reduceMotion: true })).toBe(controlMotionClass.pressReduced)
    expect(pressClassName({ enabled: false })).toBe('')
  })

  it('validation shake fires once path; reduced → pulse only (FR-4 / AC-4 / AC-5)', () => {
    expect(shouldValidationShake({ enabled: true, reduceMotion: false })).toBe(true)
    expect(shouldValidationShake({ enabled: true, reduceMotion: true })).toBe(false)
    expect(validationClassName(true, { enabled: true, reduceMotion: false })).toBe(
      controlMotionClass.shake,
    )
    expect(validationClassName(true, { enabled: true, reduceMotion: true })).toBe(
      controlMotionClass.shakePulse,
    )
    expect(validationClassName(false, { enabled: true, reduceMotion: false })).toBe('')
  })

  it('indicator offset slides between options and mirrors RTL (FR-3)', () => {
    const widths = [40, 50, 60]
    expect(indicatorOffsetPx({ index: 0, optionWidths: widths })).toBe(0)
    expect(indicatorOffsetPx({ index: 1, optionWidths: widths, gapPx: 4 })).toBe(44)
    expect(indicatorOffsetPx({ index: 2, optionWidths: widths })).toBe(90)
    // RTL: translateX still uses physical X; mirror so index 0 sits at the end.
    expect(indicatorOffsetPx({ index: 0, optionWidths: widths, dir: 'rtl' })).toBe(110)
    expect(indicatorOffsetPx({ index: 2, optionWidths: widths, dir: 'rtl' })).toBe(0)
    expect(shouldSlideIndicator({ enabled: true, reduceMotion: true })).toBe(false)
    expect(indicatorTransition({ enabled: true, reduceMotion: false })).toContain(`${durations.base}ms`)
    expect(indicatorTransition({ enabled: false })).toBeUndefined()
  })

  it('loading button reserves width and crossfades unless reduced (FR-6 / AC-6)', () => {
    const full = loadingButtonState({
      loading: true,
      labelWidthPx: 120,
      enabled: true,
      reduceMotion: false,
    })
    expect(full.ariaBusy).toBe(true)
    expect(full.minWidth).toBe(120)
    expect(full.showSpinner).toBe(true)
    expect(full.crossfade).toBe(true)

    const reduced = loadingButtonState({
      loading: true,
      labelWidthPx: 120,
      enabled: true,
      reduceMotion: true,
    })
    expect(reduced.crossfade).toBe(false)

    const idle = loadingButtonState({ loading: false, labelWidthPx: 120 })
    expect(idle.ariaBusy).toBeUndefined()
    expect(idle.minWidth).toBeUndefined()
  })

  it('haptics mapping is centralized (FR-5)', () => {
    expect(hapticMapping.tap).toBe('lightImpact')
    expect(hapticMapping.selection).toBe('selection')
    expect(hapticMapping.success).toBe('notificationSuccess')
    expect(hapticMapping.error).toBe('notificationError')
  })

  it('control duration collapses under kill-switch / reduced motion', () => {
    expect(controlMotionDurationMs({ enabled: true, reduceMotion: false })).toBe(durations.base)
    expect(controlMotionDurationMs({ enabled: true, reduceMotion: true })).toBe(durations.instant)
    expect(controlMotionDurationMs({ enabled: false })).toBe(0)
  })
})
