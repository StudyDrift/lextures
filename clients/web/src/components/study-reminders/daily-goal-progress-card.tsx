import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Target } from 'lucide-react'
import { fetchReminderConfig, type ReminderConfig } from '../../lib/study-reminders-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

export function DailyGoalProgressCard() {
  const { ffStudyReminders, loading: featuresLoading } = usePlatformFeatures()
  const [config, setConfig] = useState<ReminderConfig | null>(null)

  useEffect(() => {
    if (featuresLoading || !ffStudyReminders) return
    let cancelled = false
    void fetchReminderConfig()
      .then((c) => {
        if (!cancelled) setConfig(c)
      })
      .catch(() => {
        if (!cancelled) setConfig(null)
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffStudyReminders])

  if (featuresLoading || !ffStudyReminders || !config?.enabled) return null

  const pct =
    config.dailyGoalMinutes > 0
      ? Math.min(100, Math.round((config.minutesStudiedToday / config.dailyGoalMinutes) * 100))
      : 0
  const radius = 36
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (pct / 100) * circumference

  return (
    <section aria-label="Daily study goal" className="rounded-2xl border border-indigo-100 bg-indigo-50/80 px-5 py-4 dark:border-indigo-900/40 dark:bg-indigo-950/30">
      {config.streakAtRiskBanner ? (
        <p className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm font-medium text-amber-900 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-100">
          Streak at risk — study now to keep your momentum going.
        </p>
      ) : null}
      <div className="flex flex-wrap items-center gap-5">
        <div className="relative h-24 w-24 shrink-0" aria-hidden>
          <svg viewBox="0 0 96 96" className="h-full w-full -rotate-90">
            <circle cx="48" cy="48" r={radius} fill="none" stroke="currentColor" className="text-indigo-100 dark:text-indigo-900" strokeWidth="8" />
            <circle
              cx="48"
              cy="48"
              r={radius}
              fill="none"
              stroke="currentColor"
              className="text-indigo-600 dark:text-indigo-300"
              strokeWidth="8"
              strokeDasharray={circumference}
              strokeDashoffset={offset}
              strokeLinecap="round"
            />
          </svg>
          <span className="absolute inset-0 flex items-center justify-center text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {pct}%
          </span>
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Target className="h-5 w-5 text-indigo-600 dark:text-indigo-300" aria-hidden />
            <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Daily study goal</p>
          </div>
          <p className="mt-1 text-sm text-slate-700 dark:text-neutral-300">
            {config.minutesStudiedToday} of {config.dailyGoalMinutes} minutes today
            {config.goalMetToday ? ' — goal met!' : ''}
          </p>
          <Link to="/settings/account" className="mt-2 inline-block text-xs font-medium text-indigo-800 underline dark:text-indigo-200">
            Adjust reminders
          </Link>
        </div>
      </div>
    </section>
  )
}
