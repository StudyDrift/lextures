import { useEffect, useState } from 'react'
import { Award, Flame, Shield } from 'lucide-react'
import { LmsPage } from './lms-page'
import {
  badgeLabel,
  fetchGamificationProfile,
  spendStreakFreeze,
  type GamificationProfile,
} from '../../lib/gamification-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { Navigate } from 'react-router-dom'

export default function MyProfilePage() {
  const { ffGamification, loading: featuresLoading } = usePlatformFeatures()
  const [profile, setProfile] = useState<GamificationProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [freezing, setFreezing] = useState(false)

  useEffect(() => {
    if (featuresLoading || !ffGamification) return
    let cancelled = false
    void fetchGamificationProfile()
      .then((p) => {
        if (!cancelled) {
          setProfile(p)
          setError(null)
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load profile.')
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffGamification])

  if (!featuresLoading && !ffGamification) {
    return <Navigate to="/dashboard" replace />
  }

  return (
    <LmsPage title="My profile" description="Your streaks, XP, and badge collection.">
      {error ? (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100">
          {error}
        </p>
      ) : null}

      {!profile && !error ? (
        <p className="mt-6 text-sm text-fg-muted">Loading…</p>
      ) : null}

      {profile ? (
        <div className="mt-6 grid gap-6 lg:grid-cols-2">
          <section className="rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-fg-muted">
              Stats
            </h2>
            <dl className="mt-4 space-y-4">
              <div className="flex items-center justify-between gap-4">
                <dt className="inline-flex items-center gap-2 text-sm text-fg-muted">
                  <Flame className="h-4 w-4 text-orange-500" aria-hidden />
                  Current streak
                </dt>
                <dd
                  className="text-lg font-semibold tabular-nums text-fg-default"
                  aria-label={`${profile.currentStreak}-day learning streak`}
                >
                  {profile.currentStreak.toLocaleString()} days
                </dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt className="text-sm text-fg-muted">Longest streak</dt>
                <dd className="font-semibold tabular-nums text-fg-default">
                  {profile.longestStreak.toLocaleString()} days
                </dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt className="text-sm text-fg-muted">Total XP</dt>
                <dd className="font-semibold tabular-nums text-fg-default">
                  {profile.xpTotal.toLocaleString()}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt className="text-sm text-fg-muted">Level</dt>
                <dd className="font-semibold tabular-nums text-fg-default">{profile.level}</dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt className="inline-flex items-center gap-2 text-sm text-fg-muted">
                  <Shield className="h-4 w-4" aria-hidden />
                  Streak freezes
                </dt>
                <dd className="font-semibold tabular-nums text-fg-default">
                  {profile.streakFreezes}
                </dd>
              </div>
            </dl>
            {profile.streakFreezes > 0 && profile.currentStreak > 0 ? (
              <button
                type="button"
                disabled={freezing}
                className="mt-5 rounded-lg bg-orange-600 px-4 py-2 text-sm font-semibold text-white hover:bg-orange-500 disabled:opacity-60"
                onClick={() => {
                  setFreezing(true)
                  void spendStreakFreeze()
                    .then(setProfile)
                    .finally(() => setFreezing(false))
                }}
              >
                Use streak freeze
              </button>
            ) : null}
          </section>

          <section className="rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-fg-muted">
              Badge collection
            </h2>
            {profile.badges.length === 0 ? (
              <p className="mt-4 text-sm text-fg-muted">
                No badges yet. Complete modules and build your streak to earn your first badge.
              </p>
            ) : (
              <ul className="mt-4 grid gap-3 sm:grid-cols-2">
                {profile.badges.map((b) => (
                  <li
                    key={`${b.badgeType}-${b.awardedAt}`}
                    className="flex items-start gap-3 rounded-xl border border-border-subtle bg-slate-50/80 p-3 dark:border-border-subtle/60"
                  >
                    <Award className="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
                    <div>
                      <p className="text-sm font-semibold text-fg-default">
                        {badgeLabel(b.badgeType)}
                      </p>
                      <p className="text-xs text-fg-muted">
                        {new Date(b.awardedAt).toLocaleDateString()}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      ) : null}
    </LmsPage>
  )
}
