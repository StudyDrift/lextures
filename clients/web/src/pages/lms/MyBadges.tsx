import { useCallback, useEffect, useId, useState } from 'react'
import { Award, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchMyBadges,
  patchMyBadge,
  type AwardedBadge,
} from '../../lib/badges-api'
import { LmsPage } from './lms-page'

export default function MyBadges() {
  const titleId = useId()
  const { ffCompetencyBadges, loading: featuresLoading } = usePlatformFeatures()
  const [badges, setBadges] = useState<AwardedBadge[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMyBadges()
      setBadges(data.badges)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load badges.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffCompetencyBadges) return
    void load()
  }, [featuresLoading, ffCompetencyBadges, load])

  async function togglePublic(badge: AwardedBadge) {
    setBusyId(badge.id)
    setError(null)
    try {
      const updated = await patchMyBadge(badge.id, !badge.isPublic)
      setBadges((prev) => prev.map((b) => (b.id === badge.id ? { ...b, ...updated } : b)))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update visibility.')
    } finally {
      setBusyId(null)
    }
  }

  if (!ffCompetencyBadges && !featuresLoading) {
    return (
      <LmsPage title="My badges">
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Competency badges are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="My badges">
      <header className="mb-6">
        <h1 id={titleId} className="flex items-center gap-2 text-2xl font-semibold text-slate-900 dark:text-white">
          <Award className="h-7 w-7 text-indigo-600" aria-hidden />
          My badges
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          Micro-credentials earned for mastering course outcomes. Toggle public visibility to share on your badge page.
        </p>
      </header>

      {loading ? (
        <p className="inline-flex items-center gap-2 text-sm text-slate-600">
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
          Loading badges…
        </p>
      ) : null}

      {error ? (
        <p role="alert" className="mb-4 text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      {!loading && badges.length === 0 ? (
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Earn badges by mastering course outcomes. Your instructor can award competency micro-badges.
        </p>
      ) : null}

      <ul className="grid gap-4 md:grid-cols-2">
        {badges.map((badge) => (
          <li
            key={badge.id}
            aria-label={`Badge: ${badge.name ?? 'Badge'}, issued ${new Date(badge.issuedAt).toLocaleDateString()}`}
            className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
          >
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {badge.name ?? 'Badge'}
            </h2>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              Issued {new Date(badge.issuedAt).toLocaleDateString()}
              {badge.revoked ? ' · Revoked' : ''}
              {badge.isPublic ? ' · Public' : ' · Private'}
            </p>
            {badge.description ? (
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">{badge.description}</p>
            ) : null}
            {!badge.revoked ? (
              <button
                type="button"
                className="mt-4 rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
                disabled={busyId === badge.id}
                onClick={() => void togglePublic(badge)}
              >
                {badge.isPublic ? 'Make private' : 'Make public'}
              </button>
            ) : null}
          </li>
        ))}
      </ul>
    </LmsPage>
  )
}
