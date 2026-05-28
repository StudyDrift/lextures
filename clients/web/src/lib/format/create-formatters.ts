import { FormatterCache, stableOptionsKey } from './cache'
import { detectBrowserLocale, resolveLocale, resolveTimeZone, toDate } from './resolve'

export type LocaleFormatOptions = {
  locale?: string | null
  timeZone?: string | null
}

const dateTimeCache = new FormatterCache<Intl.DateTimeFormat>()
const numberCache = new FormatterCache<Intl.NumberFormat>()
const relativeCache = new FormatterCache<Intl.RelativeTimeFormat>()

function resolvedLocale(opts: LocaleFormatOptions): string {
  return resolveLocale(opts.locale ?? null)
}

function resolvedTimeZone(opts: LocaleFormatOptions): string {
  return resolveTimeZone(opts.timeZone ?? null)
}

function getDateTimeFormat(
  locale: string,
  timeZone: string,
  options: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  const key = `${locale}|${timeZone}|${stableOptionsKey(options)}`
  return dateTimeCache.get(key, () => {
    try {
      return new Intl.DateTimeFormat(locale, { ...options, timeZone })
    } catch {
      if (import.meta.env.DEV) {
        console.warn(`[format] DateTimeFormat failed for ${locale}/${timeZone}`)
      }
      return new Intl.DateTimeFormat(detectBrowserLocale(), { ...options, timeZone: 'UTC' })
    }
  })
}

function getNumberFormat(locale: string, options: Intl.NumberFormatOptions): Intl.NumberFormat {
  const key = `${locale}|${stableOptionsKey(options)}`
  return numberCache.get(key, () => {
    try {
      return new Intl.NumberFormat(locale, options)
    } catch {
      return new Intl.NumberFormat('en-US', options)
    }
  })
}

function getRelativeFormat(locale: string): Intl.RelativeTimeFormat {
  const key = locale
  return relativeCache.get(key, () => {
    try {
      return new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
    } catch {
      return new Intl.RelativeTimeFormat('en-US', { numeric: 'auto' })
    }
  })
}

export type LocaleFormatters = ReturnType<typeof createLocaleFormatters>

export function createLocaleFormatters(opts: LocaleFormatOptions = {}) {
  const locale = resolvedLocale(opts)
  const timeZone = resolvedTimeZone(opts)

  function formatDate(
    input: string | Date | null | undefined,
    options: Intl.DateTimeFormatOptions = { dateStyle: 'medium' },
  ): string {
    const d = toDate(input)
    if (!d) return '—'
    return getDateTimeFormat(locale, timeZone, options).format(d)
  }

  function formatDateTime(
    input: string | Date | null | undefined,
    options: Intl.DateTimeFormatOptions = {
      dateStyle: 'medium',
      timeStyle: 'short',
    },
  ): string {
    const d = toDate(input)
    if (!d) return '—'
    return getDateTimeFormat(locale, timeZone, options).format(d)
  }

  function formatRelativeTime(
    input: string | Date | null | undefined,
    now: Date = new Date(),
  ): string {
    const d = toDate(input)
    if (!d) return '—'
    const diffMs = d.getTime() - now.getTime()
    const rtf = getRelativeFormat(locale)
    const absSec = Math.abs(Math.round(diffMs / 1000))
    const sign = diffMs < 0 ? -1 : diffMs > 0 ? 1 : 0

    const intervals: { unit: Intl.RelativeTimeFormatUnit; seconds: number }[] = [
      { unit: 'year', seconds: 31536000 },
      { unit: 'month', seconds: 2592000 },
      { unit: 'week', seconds: 604800 },
      { unit: 'day', seconds: 86400 },
      { unit: 'hour', seconds: 3600 },
      { unit: 'minute', seconds: 60 },
      { unit: 'second', seconds: 1 },
    ]

    for (const { unit, seconds } of intervals) {
      const count = Math.floor(absSec / seconds)
      if (count >= 1) {
        return rtf.format(sign * count, unit)
      }
    }
    return rtf.format(0, 'second')
  }

  function formatNumber(
    value: number,
    options: Intl.NumberFormatOptions = {},
  ): string {
    if (!Number.isFinite(value)) return '—'
    return getNumberFormat(locale, options).format(value)
  }

  function formatPercent(value: number, options: Intl.NumberFormatOptions = {}): string {
    if (!Number.isFinite(value)) return '—'
    return getNumberFormat(locale, {
      style: 'percent',
      maximumFractionDigits: 1,
      ...options,
    }).format(value)
  }

  function formatCurrency(
    value: number,
    currencyCode: string,
    options: Intl.NumberFormatOptions = {},
  ): string {
    if (!Number.isFinite(value)) return '—'
    return getNumberFormat(locale, {
      style: 'currency',
      currency: currencyCode,
      ...options,
    }).format(value)
  }

  /** Locale-aware mm:ss (e.g. quiz timer). */
  function formatTimerSeconds(totalSeconds: number): string {
    const sec = Math.max(0, Math.floor(totalSeconds))
    const mm = Math.floor(sec / 60)
    const ss = sec % 60
    const pad = getNumberFormat(locale, { minimumIntegerDigits: 2, useGrouping: false })
    return `${pad.format(mm)}:${pad.format(ss)}`
  }

  function dateTimeFormat(options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
    return getDateTimeFormat(locale, timeZone, options)
  }

  function numberFormat(options: Intl.NumberFormatOptions): Intl.NumberFormat {
    return getNumberFormat(locale, options)
  }

  return {
    locale,
    timeZone,
    formatDate,
    formatDateTime,
    formatRelativeTime,
    formatNumber,
    formatPercent,
    formatCurrency,
    formatTimerSeconds,
    dateTimeFormat,
    numberFormat,
  }
}

/** @internal Exposed for unit tests */
export function clearFormatterCaches(): void {
  dateTimeCache.clear()
  numberCache.clear()
  relativeCache.clear()
}
