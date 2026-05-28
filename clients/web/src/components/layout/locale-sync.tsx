import { useEffect } from 'react'
import { apiUrl } from '../../lib/api'
import { applyDocumentLocale, readStoredLocale } from '../../i18n/document-locale'
import { normalizeLocaleCode } from '../../i18n/supported-locales'
import { usePlatformFeatures } from '../../context/platform-features-context'

/**
 * Applies locale/dir before authenticated account settings load (guest + first paint).
 */
export function LocaleBootstrapSync() {
  const { rtlEnabled } = usePlatformFeatures()

  useEffect(() => {
    applyDocumentLocale(readStoredLocale(), rtlEnabled)
    let cancelled = false
    async function detect() {
      try {
        const res = await fetch(apiUrl('/api/v1/public/locale-defaults'))
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled) return
        const data = raw as { locale?: string }
        const stored = readStoredLocale()
        if (stored === 'en' && data.locale) {
          applyDocumentLocale(normalizeLocaleCode(data.locale), rtlEnabled)
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
