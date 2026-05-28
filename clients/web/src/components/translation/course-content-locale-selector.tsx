import { useCallback, useEffect, useState } from 'react'
import { Languages } from 'lucide-react'
import {
  fetchTranslationCoverage,
  isTranslationMemoryEnabled,
  patchMyContentLocale,
} from '../../lib/course-translation-api'

const COMMON_LOCALES = [
  { code: 'en', label: 'English' },
  { code: 'es', label: 'Spanish' },
  { code: 'fr', label: 'French' },
  { code: 'de', label: 'German' },
  { code: 'ar', label: 'Arabic' },
]

type Props = {
  courseCode: string
  className?: string
}

/** Student course content language selector (plan 11.5 FR-6). */
export function CourseContentLocaleSelector({ courseCode, className }: Props) {
  const enabled = isTranslationMemoryEnabled()
  const [locales, setLocales] = useState<Array<{ code: string; label: string; percent: number }>>([])
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!enabled || !courseCode) return
    let cancelled = false
    void (async () => {
      try {
        const data = await fetchTranslationCoverage(courseCode)
        const list = 'locales' in data ? data.locales : [data]
        const opts = list
          .filter((l) => l.percent > 0)
          .map((l) => {
            const meta = COMMON_LOCALES.find((c) => c.code === l.targetLocale)
            return {
              code: l.targetLocale,
              label: meta?.label ?? l.targetLocale,
              percent: l.percent,
            }
          })
        if (!cancelled) setLocales(opts)
      } catch {
        if (!cancelled) setLocales([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, enabled])

  const onChange = useCallback(
    async (next: string) => {
      setValue(next)
      setSaving(true)
      try {
        await patchMyContentLocale(courseCode, next === '' ? null : next)
      } finally {
        setSaving(false)
      }
    },
    [courseCode],
  )

  if (!enabled || locales.length === 0) return null

  return (
    <label
      className={`inline-flex items-center gap-2 text-sm text-stone-700 dark:text-neutral-300 ${className ?? ''}`}
    >
      <Languages className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
      <span className="sr-only">Course content language</span>
      <select
        className="rounded-md border border-stone-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-900"
        value={value}
        disabled={saving}
        onChange={(e) => void onChange(e.target.value)}
        aria-label="Course content language"
      >
        <option value="">Source language</option>
        {locales.map((l) => (
          <option key={l.code} value={l.code}>
            {l.label} ({Math.round(l.percent)}%)
          </option>
        ))}
      </select>
    </label>
  )
}
