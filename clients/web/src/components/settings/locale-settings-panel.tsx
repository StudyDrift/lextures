import { useId, useState } from 'react'
import { useLocale } from '../../context/locale-context'
import { SUPPORTED_LOCALES } from '../../i18n/supported-locales'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

export function LocaleSettingsPanel() {
  const selectId = useId()
  const { locale, setLocale, loading, rtlEnabled } = useLocale()
  const [saving, setSaving] = useState(false)

  return (
    <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">Language</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Choose your interface language. Arabic and Hebrew use right-to-left layout when RTL support is
        enabled for this platform.
      </p>
      {!rtlEnabled ? (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
          RTL mirroring is not enabled yet; layout stays left-to-right until a platform admin turns on RTL
          support.
        </p>
      ) : null}
      <div className="mt-4 max-w-md">
        <label htmlFor={selectId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
          Display language
        </label>
        <select
          id={selectId}
          className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          value={locale}
          disabled={loading || saving}
          onChange={(e) => {
            const next = e.target.value
            setSaving(true)
            void setLocale(next)
              .then(() => toastSaveOk('Language updated'))
              .catch((err: unknown) =>
                toastMutationError(err instanceof Error ? err.message : 'Could not update language'),
              )
              .finally(() => setSaving(false))
          }}
        >
          {SUPPORTED_LOCALES.map((loc) => (
            <option key={loc.code} value={loc.code}>
              {loc.nativeLabel} ({loc.label})
            </option>
          ))}
        </select>
      </div>
    </section>
  )
}
