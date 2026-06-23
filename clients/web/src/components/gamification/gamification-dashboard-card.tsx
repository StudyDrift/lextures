import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Award, Flame, Sparkles, X } from 'lucide-react'
import {
  badgeLabel,
  fetchGamificationProfile,
  spendStreakFreeze,
  type GamificationProfile,
} from '../../lib/gamification-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

export function GamificationDashboardCard() {
  const { ffGamification, loading: featuresLoading } = usePlatformFeatures()
  const [profile, setProfile] = useState<GamificationProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dismissedEnded, setDismissedEnded] = useState(false)
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
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load gamification.')
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffGamification])

  if (featuresLoading || !ffGamification) return null
  if (error) return null
  if (!profile) return null

  const empty = profile.xpTotal === 0 && profile.currentStreak === 0

  return (
    <section aria-label="Learning gamification" className="space-y-3">
      {profile.streakEnded && !dismissedEnded ? (
        <div className="relative rounded-2xl border border-rose-100 bg-rose-50/90 px-5 py-4 dark:border-rose-900/40 dark:bg-rose-950/30">
          <button
            type="button"
            className="lex-icon-hit absolute end-3 top-3 rounded-lg text-rose-800 hover:bg-rose-100/80 dark:text-rose-200 dark:hover:bg-rose-900/50"
            aria-label="Dismiss streak ended notice"
            onClick={() => setDismissedEnded(true)}
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
          <p className="pe-8 text-sm font-semibold text-rose-900 dark:text-rose-100">Your streak ended</p>
          <p className="mt-1 text-sm text-rose-800 dark:text-rose-200">
            Complete a module today to start a new streak.
          </p>
        </div>
      ) : null}

      {profile.streakAtRisk ? (
        <div className="rounded-2xl border border-amber-100 bg-amber-50/90 px-5 py-4 dark:border-amber-900/40 dark:bg-amber-950/30">
          <p className="text-sm font-semibold text-amber-900 dark:text-amber-100">
            You&apos;ll lose your {profile.currentStreak}-day streak
            {profile.streakHoursLeft != null && profile.streakHoursLeft > 0
              ? ` in about ${Math.max(1, Math.round(profile.streakHoursLeft))} hour${Math.round(profile.streakHoursLeft) === 1 ? '' : 's'}`
              : ''}{' '}
            — study now!
          </p>
          {profile.streakFreezes > 0 ? (
            <button
              type="button"
              disabled={freezing}
              className="mt-3 rounded-lg border border-amber-300 bg-white px-3 py-1.5 text-xs font-semibold text-amber-900 hover:bg-amber-50 disabled:opacity-60 dark:border-amber-700 dark:bg-neutral-900 dark:text-amber-100"
              onClick={() => {
                setFreezing(true)
                void spendStreakFreeze()
                  .then(setProfile)
                  .finally(() => setFreezing(false))
              }}
            >
              Use streak freeze ({profile.streakFreezes} left)
            </button>
          ) : null}
        </div>
      ) : null}

      <div className="rounded-2xl border border-orange-100 bg-gradient-to-br from-orange-50/90 to-white p-5 shadow-sm dark:border-orange-900/40 dark:from-orange-950/30 dark:to-neutral-900">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Your progress</p>
            {profile.currentStreak > 0 ? (
              <p
                className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-orange-700 dark:text-orange-200"
                aria-label={`${profile.currentStreak}-day learning streak`}
              >
                <Flame className="h-4 w-4" aria-hidden />
                <span className="lex-num">{profile.currentStreak.toLocaleString()}</span>-day streak
              </p>
            ) : empty ? (
              <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
                Start your journey — earn your first 5 XP.
              </p>
            ) : (
              <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">Complete a module to start your streak</p>
            )}
            <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">
              Level <span className="lex-num">{profile.level}</span> ·{' '}
              <span className="lex-num">{profile.xpTotal.toLocaleString()}</span> XP
            </p>
          </div>
          <Link
            to="/me/profile"
            className="shrink-0 text-xs font-semibold text-orange-800 underline dark:text-orange-200"
          >
            My profile
          </Link>
        </div>

        <div className="mt-4">
          <div
            className="h-2 overflow-hidden rounded-full bg-orange-200/80 dark:bg-orange-900/50"
            role="progressbar"
            aria-valuenow={profile.levelProgressPct}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={`XP progress toward level ${profile.level + 1}`}
          >
            <div
              className="h-full rounded-full bg-orange-600 motion-safe:transition-[width] motion-safe:duration-300 dark:bg-orange-500"
              style={{ width: `${profile.levelProgressPct}%` }}
            />
          </div>
          <p className="mt-2 text-xs text-slate-700 dark:text-neutral-300">
            <span className="lex-num">{profile.xpToNextLevel.toLocaleString()}</span> XP to level{' '}
            <span className="lex-num">{profile.level + 1}</span>
          </p>
        </div>

        {profile.recentBadges.length > 0 ? (
          <div className="mt-4 border-t border-orange-100 pt-4 dark:border-orange-900/40">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Recent badges
            </p>
            <ul className="mt-2 flex gap-2 overflow-x-auto pb-1">
              {profile.recentBadges.map((b) => (
                <li
                  key={`${b.badgeType}-${b.awardedAt}`}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-orange-200 bg-white px-3 py-1 text-xs font-medium text-orange-900 dark:border-orange-800 dark:bg-neutral-900 dark:text-orange-100"
                >
                  <Award className="h-3.5 w-3.5" aria-hidden />
                  <span>{badgeLabel(b.badgeType)}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <p className="mt-4 flex items-center gap-1.5 text-xs text-slate-500 dark:text-neutral-400">
            <Sparkles className="h-3.5 w-3.5" aria-hidden />
            Earn badges for streaks, XP milestones, and course completions.
          </p>
        )}
      </div>
    </section>
  )
}
