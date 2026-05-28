import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Flame, Lightbulb, X } from 'lucide-react'
import { fetchCoachingTips, fetchStudyStats, type CoachingTip, type StudyStats } from '../../lib/study-reflection-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

function formatHours(h: number): string {
  if (h < 0.1) return '0'
  return h >= 10 ? h.toFixed(0) : h.toFixed(1)
}

export function StudyStatsCard() {
  const { selfReflectionEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [stats, setStats] = useState<StudyStats | null>(null)
  const [tip, setTip] = useState<CoachingTip | null>(null)
  const [tipDismissed, setTipDismissed] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (featuresLoading || !selfReflectionEnabled) return
    let cancelled = false
    ;(async () => {
      try {
        const [s, tips] = await Promise.all([fetchStudyStats(), fetchCoachingTips()])
        if (!cancelled) {
          setStats(s)
          setTip(tips.latest)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load study stats.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [featuresLoading, selfReflectionEnabled])

  if (featuresLoading || !selfReflectionEnabled) return null
  if (error) return null
  if (!stats?.optedIn) return null

  const hoursStudied = stats.timeOnTaskSecondsThisWeek / 3600
  const goalHours = stats.weeklyGoalHours
  const goalPct =
    goalHours != null && goalHours > 0
      ? Math.min(100, Math.round((hoursStudied / goalHours) * 100))
      : null

  return (
    <section aria-label="Study stats" className="space-y-3">
      {tip && !tipDismissed ? (
        <div className="relative rounded-2xl border border-amber-100 bg-amber-50/90 px-5 py-4 dark:border-amber-900/40 dark:bg-amber-950/30">
          <button
            type="button"
            className="absolute end-3 top-3 rounded-lg p-1 text-amber-800 hover:bg-amber-100/80 dark:text-amber-200 dark:hover:bg-amber-900/50"
            aria-label="Dismiss coaching tip"
            onClick={() => setTipDismissed(true)}
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
          <div className="flex gap-3 pe-8">
            <Lightbulb className="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-300" aria-hidden />
            <div>
              <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Weekly coaching tip</p>
              <p className="mt-1 text-sm text-slate-700 dark:text-neutral-300">{tip.tipText}</p>
              <Link
                to="/me/study-insights"
                className="mt-2 inline-block text-xs font-medium text-amber-800 underline dark:text-amber-200"
              >
                View study insights
              </Link>
            </div>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-emerald-100 bg-emerald-50/80 px-5 py-4 dark:border-emerald-900/40 dark:bg-emerald-950/30">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Study stats</p>
            {stats.loginStreakDays > 0 ? (
              <p
                className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-emerald-800 dark:text-emerald-200"
                aria-label={`${stats.loginStreakDays}-day study streak`}
              >
                <Flame className="h-4 w-4" aria-hidden />
                {stats.loginStreakDays}-day study streak
              </p>
            ) : (
              <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">Log in daily to build your streak</p>
            )}
            <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">
              {formatHours(hoursStudied)} hour{hoursStudied === 1 ? '' : 's'} studied this week
            </p>
          </div>
          <Link
            to="/me/study-insights"
            className="shrink-0 text-xs font-semibold text-emerald-800 underline dark:text-emerald-200"
          >
            My goals &amp; journal
          </Link>
        </div>

        {goalHours != null && goalHours > 0 ? (
          <div className="mt-4">
            <div
              className="h-2 overflow-hidden rounded-full bg-emerald-200/80 dark:bg-emerald-900/50"
              role="progressbar"
              aria-valuenow={goalPct ?? 0}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`Weekly study goal: ${formatHours(hoursStudied)} of ${formatHours(goalHours)} hours`}
            >
              <div
                className="h-full rounded-full bg-emerald-600 transition-all dark:bg-emerald-500"
                style={{ width: `${goalPct ?? 0}%` }}
              />
            </div>
            <p className="mt-2 text-xs text-slate-700 dark:text-neutral-300">
              {formatHours(hoursStudied)} of {formatHours(goalHours)} hours
              {stats.goalRemainingHours != null && stats.goalRemainingHours > 0
                ? ` — ${formatHours(stats.goalRemainingHours)} more to reach your goal this week`
                : ' — goal reached this week'}
            </p>
          </div>
        ) : null}

        {stats.lowStudyEfficiency ? (
          <p className="mt-3 text-xs text-amber-800 dark:text-amber-200">
            You&apos;ve put in study time but quiz scores aren&apos;t improving much — try practice tests or
            teaching concepts back to yourself.
          </p>
        ) : null}
      </div>
    </section>
  )
}
