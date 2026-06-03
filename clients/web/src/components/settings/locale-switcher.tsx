import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { applyDocumentLocale } from '../../i18n/apply-document-locale'
import { i18n } from '../../i18n'
import { writeStoredLocaleTag } from '../../i18n/locale-storage'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LOCALE_OPTIONS, resolveResourceLanguage } from '../../i18n/supported-locales'

type Props = {
  initialLocale?: string | null
  onLocaleChange?: (tag: string) => void
}

export function LocaleSwitcher({ initialLocale, onLocaleChange }: Props) {
  const { t } = useTranslation('common')
  const { rtlEnabled } = usePlatformFeatures()
  const selectId = useId()
  const [localeTag, setLocaleTag] = useState(() => initialLocale?.trim() || i18n.language || 'en')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (initialLocale?.trim()) {
      setLocaleTag(initialLocale.trim())
    }
  }, [initialLocale])

  const applyLocale = useCallback(
    async (tag: string) => {
      const resourceLang = resolveResourceLanguage(tag)
      writeStoredLocaleTag(tag)
      applyDocumentLocale(tag, rtlEnabled)
      await i18n.changeLanguage(resourceLang)
    },
    [rtlEnabled],
  )

  const onSelect = useCallback(
    async (tag: string) => {
      const prev = localeTag
      setLocaleTag(tag)
      setError(null)
      setSaving(true)
      try {
        await applyLocale(tag)
        const res = await authorizedFetch('/api/v1/settings/locale', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ locale: tag }),
        })
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok) {
          setLocaleTag(prev)
          await applyLocale(prev)
          setError(readApiErrorMessage(raw))
          toastMutationError(t('common.locale.saveError'))
          return
        }
        const data = raw as { locale?: string }
        const saved = data.locale?.trim() || tag
        setLocaleTag(saved)
        onLocaleChange?.(saved)
        toastSaveOk(t('common.locale.saved'))
      } catch {
        setLocaleTag(prev)
        await applyLocale(prev)
        setError(t('common.locale.saveError'))
        toastMutationError(t('common.locale.saveError'))
      } finally {
        setSaving(false)
      }
    },
    [applyLocale, localeTag, onLocaleChange, t],
  )

  return (
    <div className="mt-8">
      <div className="flex items-center gap-2">
        <Globe className="h-4 w-4 text-slate-500 dark:text-neutral-400" aria-hidden />
        <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">{t('common.locale.label')}</p>
      </div>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{t('common.locale.description')}</p>
      {!rtlEnabled ? (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
          RTL layout mirroring is not enabled on this platform; Arabic and Hebrew use left-to-right chrome until
          a platform admin enables RTL support.
        </p>
      ) : null}
      <label htmlFor={selectId} className="sr-only">
        {t('common.locale.label')}
      </label>
      <select
        id={selectId}
        data-testid="locale-switcher"
        className="mt-3 w-full max-w-xs rounded-xl border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
        value={localeTag}
        disabled={saving}
        onChange={(e) => void onSelect(e.target.value)}
      >
        {LOCALE_OPTIONS.map((opt) => (
          <option key={opt.tag} value={opt.tag}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && (
        <p className="mt-2 text-sm text-rose-600 dark:text-rose-400" role="status">
          {error}
        </p>
      )}
    </div>
  )
}
