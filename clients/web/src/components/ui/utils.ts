/**
 * UX.2 — shared class-name helper and size tokens for the core library.
 * Keep this module free of React so tree-shaking stays cheap.
 */

export type ControlSize = 'sm' | 'md' | 'lg'

/** Minimum 24×24 CSS px target (WCAG 2.2 SC 2.5.8) on every size. */
export const sizeClasses: Record<ControlSize, string> = {
  sm: 'min-h-6 min-w-6 px-2.5 py-1 text-xs',
  md: 'min-h-9 min-w-9 px-4 py-2 text-sm',
  lg: 'min-h-11 min-w-11 px-5 py-2.5 text-base',
}

export const iconSizeClasses: Record<ControlSize, string> = {
  sm: 'min-h-6 min-w-6 h-6 w-6 p-1',
  md: 'min-h-9 min-w-9 h-9 w-9 p-2',
  lg: 'min-h-11 min-w-11 h-11 w-11 p-2.5',
}

/** Token-driven focus ring meeting SC 1.4.11 / 2.4.7. */
export const focusRingClass =
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-border-focus focus-visible:ring-offset-2 focus-visible:ring-offset-surface-base'

export const controlBaseClass =
  'inline-flex items-center justify-center gap-2 rounded-xl font-semibold disabled:cursor-not-allowed disabled:opacity-50'

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}
