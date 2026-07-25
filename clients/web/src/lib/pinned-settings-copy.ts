/**
 * Copy + constants for pinned editor settings (PS.3 / PS.4).
 * English literals today, matching the assignment/quiz panels.
 */

import type { SettingsSurface } from './settings-registry'

/** Soft UI cap (under PS.2 schema cap of 12). */
export const PINNED_SETTINGS_UI_CAP = 8

/** Debounce for PUT coalescing (FR-16). */
export const PINNED_SETTINGS_SAVE_DEBOUNCE_MS = 500

/** localStorage key for first-run hint dismissal (PS.3 FR-20; replaced by suggestions in PS.4). */
export const PINNED_HINT_DISMISSED_KEY = 'lextures.pinned-settings.hint-dismissed'

/** localStorage key prefix for per-surface suggestion dismissal (PS.4 FR-5). */
export const SUGGESTIONS_DISMISSED_KEY_PREFIX = 'lextures.pinned-settings.suggestions-dismissed'

export function suggestionsDismissedKey(surface: SettingsSurface): string {
  return `${SUGGESTIONS_DISMISSED_KEY_PREFIX}.${surface}`
}

export const pinnedSettingsCopy = {
  title: 'Pinned',
  hint: "Pin the settings you use most — they'll show up here",
  dismissHint: 'Dismiss pin hint',
  pinAction: (label: string) => `Pin ${label} to top`,
  unpinAction: (label: string) => `Unpin ${label}`,
  capReached: (max: number) =>
    `You can pin up to ${max} settings. Unpin one to add another.`,
  sectionHint: (count: number) =>
    count === 1 ? '1 pinned to top' : `${count} pinned to top`,
  saveFailed: "Couldn't save your pinned settings",
  announcePinned: (label: string, index: number, total: number) =>
    `${label} pinned, position ${index} of ${total}`,
  announceUnpinned: (label: string) => `${label} unpinned`,
  announceMoved: (label: string, index: number, total: number) =>
    `${label} moved to position ${index} of ${total}`,
  reorderHelp: 'Press Alt plus up or down arrow to reorder',
  reorderHandle: (label: string) => `Reorder ${label}`,
  /** PS.4 suggested-pins strip. */
  suggestions: {
    heading: 'Suggested pins',
    intro: 'Settings other instructors keep at the top',
    pinAction: (label: string) => `Pin ${label} to top`,
    dismiss: 'Not now',
    dismissAria: 'Dismiss pin suggestions',
  },
} as const
