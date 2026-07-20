/**
 * AN.6 — Control micro-interactions (press, toggle/tabs, validation, loading, haptics).
 *
 * Pure helpers + a web no-op haptics hook. Feature code imports from here —
 * never scatter press/shake/haptic literals.
 */

import { useCallback } from 'react'
import { distances, durations, easings, prefersReducedMotion } from './motion'

/** Press scale applied to tappable controls (FR-1). */
export const CONTROL_PRESS_SCALE = distances.pressScale

/** Validation shake travel in px (horizontal, RTL-safe). */
export const VALIDATION_SHAKE_PX = 6

/** Haptic kinds mapped for mobile; web is a no-op (FR-5). */
export type HapticKind = 'tap' | 'selection' | 'success' | 'error'

export type ControlMotionOptions = {
  /** Feature kill-switch (`ff_motion_controls`). Default true. */
  enabled?: boolean
  reduceMotion?: boolean
}

function resolveReduce(opts?: ControlMotionOptions): boolean {
  return opts?.reduceMotion ?? prefersReducedMotion()
}

function resolveEnabled(opts?: ControlMotionOptions): boolean {
  return opts?.enabled !== false
}

/**
 * Whether press scale should run (FR-1 / FR-7 / AC-5).
 * Reduced motion → false (opacity/color only); kill-switch → false.
 */
export function shouldPressScale(opts?: ControlMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (resolveReduce(opts)) return false
  return true
}

/**
 * Whether validation shake should run (FR-4 / FR-7).
 * Reduced motion / kill-switch → color/icon only.
 */
export function shouldValidationShake(opts?: ControlMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (resolveReduce(opts)) return false
  return true
}

/**
 * Whether tab/segment indicators should slide (FR-3 / FR-7).
 * Reduced motion → cut/crossfade (no slide).
 */
export function shouldSlideIndicator(opts?: ControlMotionOptions): boolean {
  if (!resolveEnabled(opts)) return false
  if (resolveReduce(opts)) return false
  return true
}

/**
 * Compute the left offset (px) for a sliding pill/underline indicator.
 * Options are measured left→right in LTR; pass `dir: 'rtl'` to mirror.
 */
export function indicatorOffsetPx(opts: {
  index: number
  optionWidths: number[]
  gapPx?: number
  dir?: 'ltr' | 'rtl'
}): number {
  const { index, optionWidths, gapPx = 0, dir = 'ltr' } = opts
  if (optionWidths.length === 0) return 0
  const clamped = Math.min(Math.max(0, index), optionWidths.length - 1)
  let offset = 0
  for (let i = 0; i < clamped; i++) {
    offset += optionWidths[i]! + gapPx
  }
  if (dir === 'rtl') {
    const total = optionWidths.reduce((sum, w) => sum + w, 0) + gapPx * (optionWidths.length - 1)
    const activeWidth = optionWidths[clamped]!
    return total - offset - activeWidth
  }
  return offset
}

/** Duration for press settle / validation shake (≤ base). */
export function controlMotionDurationMs(opts?: ControlMotionOptions): number {
  if (!resolveEnabled(opts)) return 0
  if (resolveReduce(opts)) return durations.instant
  return durations.base
}

/** CSS class tokens for control motion surfaces. */
export const controlMotionClass = {
  pressable: 'lx-control-press',
  pressReduced: 'lx-control-press-reduced',
  shake: 'lx-control-shake',
  shakePulse: 'lx-control-pulse',
  loading: 'lx-control-loading',
  indicator: 'lx-control-indicator',
  checkIn: 'lx-control-check-in',
} as const

/**
 * Resolve press className for a control (FR-1 / FR-7).
 * When reduced: opacity feedback class only; when disabled: empty.
 */
export function pressClassName(opts?: ControlMotionOptions): string {
  if (!resolveEnabled(opts)) return ''
  if (resolveReduce(opts)) return controlMotionClass.pressReduced
  return controlMotionClass.pressable
}

/**
 * Resolve validation feedback class (shake vs color pulse).
 */
export function validationClassName(invalid: boolean, opts?: ControlMotionOptions): string {
  if (!invalid) return ''
  if (!resolveEnabled(opts)) return ''
  if (resolveReduce(opts)) return controlMotionClass.shakePulse
  return controlMotionClass.shake
}

/** Transition string for indicator slide (bubble/standard). */
export function indicatorTransition(opts?: ControlMotionOptions): string | undefined {
  if (!shouldSlideIndicator(opts)) return undefined
  return `transform ${durations.base}ms ${easings.bubble}, width ${durations.base}ms ${easings.standard}`
}

/**
 * Loading-button layout: reserve min-width and crossfade label↔spinner (FR-6 / AC-6).
 */
export function loadingButtonState(opts: {
  loading: boolean
  labelWidthPx?: number
  enabled?: boolean
  reduceMotion?: boolean
}): {
  ariaBusy: boolean | undefined
  minWidth: number | undefined
  showSpinner: boolean
  crossfade: boolean
} {
  const enabled = opts.enabled !== false
  const reduce = opts.reduceMotion ?? prefersReducedMotion()
  return {
    ariaBusy: opts.loading || undefined,
    minWidth: opts.loading && opts.labelWidthPx ? opts.labelWidthPx : undefined,
    showSpinner: opts.loading,
    crossfade: enabled && !reduce,
  }
}

/**
 * Web haptics helper — no-op on web; same API surface as mobile (FR-5).
 * Never delays the control action (FR-9).
 */
export function useHaptics(): { trigger: (kind: HapticKind) => void } {
  const trigger = useCallback((_kind: HapticKind) => {
    // Web: intentional no-op. Mobile platforms map kinds to system feedback.
  }, [])
  return { trigger }
}

/** Pure mapping table for tests / mobile ports (FR-5). */
export const hapticMapping: Record<HapticKind, string> = {
  tap: 'lightImpact',
  selection: 'selection',
  success: 'notificationSuccess',
  error: 'notificationError',
}
