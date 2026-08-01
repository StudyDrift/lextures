import { IntlMessageFormat } from 'intl-messageformat'

type ParseInfo = {
  resolved?: { res?: string }
}

/**
 * Convert legacy i18next `{{name}}` placeholders to ICU `{name}`.
 * Complex ICU forms (`{count, plural, ...}`) already use single braces and are left alone.
 */
export function normalizeI18nextPlaceholders(res: string): string {
  return res.replace(/\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}/g, '{$1}')
}

/**
 * i18nFormat plugin for ICU MessageFormat (plan 11.1).
 * Replaces i18next-icu to avoid ESM default-import issues with intl-messageformat in Vitest/Vite.
 *
 * Also accepts legacy double-brace placeholders so mixed locale catalogs keep interpolating.
 */
export class IcuFormatPlugin {
  readonly type = 'i18nFormat' as const

  private readonly cache = new Map<string, IntlMessageFormat>()

  init(): void {
    /* i18next calls init(i18next, options) — no-op */
  }

  parse(
    res: string,
    options: Record<string, unknown>,
    lng: string,
    _ns: string,
    key: string,
    info?: ParseInfo,
  ): string {
    const hadLookup = Boolean(info?.resolved?.res)
    if (!hadLookup && !res) return res
    const message = normalizeI18nextPlaceholders(res)
    const cacheKey = `${lng}|${key}|${message}`
    let formatter = this.cache.get(cacheKey)
    if (!formatter) {
      try {
        formatter = new IntlMessageFormat(message, lng, undefined, { ignoreTag: true })
        this.cache.set(cacheKey, formatter)
      } catch {
        return res
      }
    }
    try {
      return formatter.format(options) as string
    } catch {
      return res
    }
  }
}
