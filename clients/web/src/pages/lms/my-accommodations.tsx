import { useEffect, useState } from 'react'
import { ShieldCheck } from 'lucide-react'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchMyAccommodations,
  type AccommodationProfile,
  type AffectedCourse,
} from '../../lib/accessibility-api'

export default function MyAccommodationsPage() {
  const { ffAccessibilityIntake, loading: featuresLoading } = usePlatformFeatures()
  const [profiles, setProfiles] = useState<AccommodationProfile[]>([])
  const [courses, setCourses] = useState<AffectedCourse[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (featuresLoading || !ffAccessibilityIntake) {
      setLoading(false)
      return
    }
    let cancelled = false
    void fetchMyAccommodations()
      .then((res) => {
        if (cancelled) return
        setProfiles(res.profiles)
        setCourses(res.affectedCourses)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load your accommodations.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [ffAccessibilityIntake, featuresLoading])

  if (featuresLoading || loading) {
    return (
      <LmsPage title="My accommodations">
        <p className="text-sm text-slate-500">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffAccessibilityIntake) {
    return (
      <LmsPage title="My accommodations">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Accessibility services are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="My accommodations">
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        These accommodations are configured by your accessibility services office and applied
        automatically across your courses. Contact the office if anything looks incorrect.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {profiles.length === 0 ? (
        <p className="mt-6 text-sm text-slate-500">You have no active accommodations.</p>
      ) : (
        <ul className="mt-6 space-y-3" aria-label="Active accommodations">
          {profiles.map((p) => (
            <li
              key={p.id}
              className="rounded-xl border border-slate-200 px-4 py-3 dark:border-neutral-800"
            >
              <p className="flex items-center gap-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
                <ShieldCheck className="h-4 w-4 text-violet-500" aria-hidden="true" />
                {p.labels.join(', ')}
              </p>
              <p className="text-xs text-slate-500">
                Effective {p.effectiveFrom}
                {p.effectiveUntil ? ` – ${p.effectiveUntil}` : ''}
              </p>
            </li>
          ))}
        </ul>
      )}

      <section aria-label="Courses affected" className="mt-8">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Courses affected
        </h2>
        {courses.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">You are not currently enrolled in any courses.</p>
        ) : (
          <ul className="mt-2 space-y-1">
            {courses.map((c) => (
              <li key={c.courseId} className="text-sm text-slate-700 dark:text-neutral-300">
                <span className="font-mono text-xs text-slate-500">{c.courseCode}</span> — {c.title}
              </li>
            ))}
          </ul>
        )}
      </section>
    </LmsPage>
  )
}
