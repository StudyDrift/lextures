import { useEffect } from 'react'
import { apiUrl } from '../../lib/api'
import { applyDocumentLocale } from '../../i18n/apply-document-locale'
import { readStoredLocaleTag, writeStoredLocaleTag } from '../../i18n/locale-storage'
import { normalizeLocaleCode } from '../../i18n/supported-locales'
import { usePlatformFeatures } from '../../context/platform-features-context'

/**
 * Applies locale/dir before authenticated settings load (guest + first paint).
 */
export function LocaleBootstrapSync() {
  const { rtlEnabled } = usePlatformFeatures()

  useEffect(() => {
    const stored = readStoredLocaleTag() ?? 'en'
    applyDocumentLocale(stored, rtlEnabled)
    let cancelled = false
    async function detect() {
      try {
        const res = await fetch(apiUrl('/api/v1/public/locale-defaults'))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled) return
        const data = raw as { locale?: string }
        if (!readStoredLocaleTag() && data.locale) {
          const tag = normalizeLocaleCode(data.locale)
          writeStoredLocaleTag(tag)
          applyDocumentLocale(tag, rtlEnabled)
        }
      } catch {
        /* ignore */
      }
    }
    void detect()
    return () => {
      cancelled = true
    }
  }, [rtlEnabled])

  return null
}
