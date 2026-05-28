import { IntlMessageFormat } from 'intl-messageformat'

type ParseInfo = {
  resolved?: { res?: string }
}

/**
 * i18nFormat plugin for ICU MessageFormat (plan 11.1).
 * Replaces i18next-icu to avoid ESM default-import issues with intl-messageformat in Vitest/Vite.
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
    const cacheKey = `${lng}|${key}|${res}`
    let formatter = this.cache.get(cacheKey)
    if (!formatter) {
      try {
        formatter = new IntlMessageFormat(res, lng, undefined, { ignoreTag: true })
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
