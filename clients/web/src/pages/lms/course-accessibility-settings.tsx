import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchCourseAccessibility,
  type CourseAccessibilityInfo,
} from '../../lib/alt-text-api'
import { formatCoverageLabel } from '../../lib/image-alt-validation'

type CourseAccessibilitySettingsSectionProps = {
  courseCode: string
}

export function CourseAccessibilitySettingsSection({
  courseCode,
}: CourseAccessibilitySettingsSectionProps) {
  const { altTextEnforcementEnabled, loading: featuresLoading } = usePlatformFeatures()
  const enabled = altTextEnforcementEnabled
  const [fetchState, setFetchState] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<CourseAccessibilityInfo | null>(null)

  useEffect(() => {
    if (featuresLoading || !enabled) return
    let cancelled = false
    void (async () => {
      setFetchState('loading')
      try {
        const res = await fetchCourseAccessibility(courseCode)
        if (!cancelled) {
          setData(res)
          setFetchState('done')
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load accessibility data')
          setFetchState('error')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, enabled, featuresLoading])

  const loading = fetchState === 'loading'

  if (featuresLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-300">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
        Loading platform settings…
      </div>
    )
  }

  if (!enabled) {
    return (
      <p className="text-sm text-slate-600 dark:text-neutral-300">
        Alt-text enforcement is not enabled on this platform. Ask a global admin to turn it on under
        Settings → Global platform.
      </p>
    )
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-300">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
        Loading accessibility coverage…
      </div>
    )
  }

  if (error) {
    return <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>
  }

  const cov = data?.altTextCoverage
  if (!cov) return null

  return (
    <section className="space-y-4">
      <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          Image alt-text coverage
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
          Percentage of images in course content and assignments that have descriptive alt text or
          are marked decorative.
        </p>
        <p className="mt-3 text-2xl font-bold text-slate-900 dark:text-neutral-50">
          {formatCoverageLabel(cov.withAlt, cov.total)}
        </p>
        {data?.hardBlockSave ? (
          <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
            Save is blocked for pages with missing alt text until images are updated.
          </p>
        ) : null}
      </div>

      {cov.uncoveredItems.length > 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Items needing alt text
          </h3>
          <ul className="mt-2 space-y-2">
            {cov.uncoveredItems.map((item) => (
              <li key={item.itemId}>
                <Link
                  to={
                    item.kind === 'assignment'
                      ? `/courses/${encodeURIComponent(courseCode)}/assignments/${encodeURIComponent(item.itemId)}`
                      : `/courses/${encodeURIComponent(courseCode)}/modules/content/${encodeURIComponent(item.itemId)}`
                  }
                  className="text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-300"
                >
                  {item.title || 'Untitled'}
                </Link>
                <span className="ms-2 text-xs text-slate-500 dark:text-neutral-400">
                  {item.missing} missing / {item.total} images
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <p className="text-sm text-emerald-700 dark:text-emerald-300">
          All images in this course have alt text or are marked decorative.
        </p>
      )}
    </section>
  )
}
