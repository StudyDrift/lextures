export {
  createLocaleFormatters,
  type LocaleFormatters,
} from './create-formatters'
export {
  detectBrowserLocale,
  detectBrowserTimeZone,
  toDate,
  toIsoDateTime,
} from './resolve'
export {
  detectBrowserTimezone,
  formatDeadlineDisplay,
  formatUtcOffsetLabel,
  resolveDisplayTimezone
} from './timezone'

import { createLocaleFormatters, type LocaleFormatters } from './create-formatters'
import { detectBrowserLocale, detectBrowserTimeZone } from './resolve'

let activeFormatters: LocaleFormatters = createLocaleFormatters({
  locale: detectBrowserLocale(),
  timeZone: detectBrowserTimeZone(),
})

export function setActiveLocaleFormatters(formatters: LocaleFormatters): void {
  activeFormatters = formatters
}

export function getActiveLocaleFormatters(): LocaleFormatters {
  return activeFormatters
}

export function formatDate(
  input: Parameters<LocaleFormatters['formatDate']>[0],
  options?: Parameters<LocaleFormatters['formatDate']>[1],
): string {
  return activeFormatters.formatDate(input, options)
}

export function formatDateTime(
  input: Parameters<LocaleFormatters['formatDateTime']>[0],
  options?: Parameters<LocaleFormatters['formatDateTime']>[1],
): string {
  return activeFormatters.formatDateTime(input, options)
}

export function formatRelativeTime(
  input: Parameters<LocaleFormatters['formatRelativeTime']>[0],
  now?: Date,
): string {
  return activeFormatters.formatRelativeTime(input, now)
}

export function formatNumber(
  value: number,
  options?: Intl.NumberFormatOptions,
): string {
  return activeFormatters.formatNumber(value, options)
}
