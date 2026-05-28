import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createReflectionJournalEntry,
  deleteReflectionJournalEntry,
  fetchCoachingTips,
  fetchReflectionJournal,
  fetchStudyGoal,
  fetchStudyStats,
  putStudyGoal,
  rateCoachingTip,
  type CoachingTip,
  type JournalEntry,
  type StudyStats,
} from '../../lib/study-reflection-api'

function formatHours(h: number): string {
  return h >= 10 ? h.toFixed(0) : h.toFixed(1)
}

export default function StudyInsightsPage() {
  const { selfReflectionEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [stats, setStats] = useState<StudyStats | null>(null)
  const [journal, setJournal] = useState<JournalEntry[]>([])
  const [tips, setTips] = useState<CoachingTip[]>([])
  const [weeklyHours, setWeeklyHours] = useState(10)
  const [optedIn, setOptedIn] = useState(false)
  const [journalDraft, setJournalDraft] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const loadGeneration = useRef(0)

  const reload = useCallback(async () => {
    const generation = ++loadGeneration.current

    const isActive = () => generation === loadGeneration.current

    try {
      let goalOptedIn = false
      let goalHours = 0

      try {
        const goal = await fetchStudyGoal()
        if (!isActive()) return
        goalOptedIn = goal.optedIn
        if (goal.weeklyHours > 0) goalHours = goal.weeklyHours
      } catch {
        const statsProbe = await fetchStudyStats().catch(() => null)
        if (!isActive()) return
        if (statsProbe) {
          goalOptedIn = statsProbe.optedIn
          if (statsProbe.weeklyGoalHours != null) goalHours = statsProbe.weeklyGoalHours
        }
      }

      setOptedIn(goalOptedIn)
      if (goalHours > 0) setWeeklyHours(goalHours)

      if (!goalOptedIn) {
        setStats(null)
        setJournal([])
        setTips([])
        setError(null)
        return
      }

      const [statsResult, journalResult, tipsResult] = await Promise.allSettled([
        fetchStudyStats(),
        fetchReflectionJournal(),
        fetchCoachingTips(),
      ])
      if (!isActive()) return

      if (statsResult.status === 'fulfilled') {
        setStats(statsResult.value)
        if (statsResult.value.weeklyGoalHours != null) {
          setWeeklyHours(statsResult.value.weeklyGoalHours)
        }
      } else {
        setStats(null)
      }

      if (journalResult.status === 'fulfilled') {
        setJournal(journalResult.value)
      }

      if (tipsResult.status === 'fulfilled') {
        setTips(tipsResult.value.history)
      }

      const failures = [statsResult, journalResult, tipsResult].filter((r) => r.status === 'rejected')
      if (failures.length === 3) {
        const first = failures[0] as PromiseRejectedResult
        setError(first.reason instanceof Error ? first.reason.message : 'Failed to load')
      } else {
        setError(null)
      }
    } catch (e) {
      if (!isActive()) return
      setError(e instanceof Error ? e.message : 'Failed to load')
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !selfReflectionEnabled) return
    void reload()
    return () => {
      loadGeneration.current += 1
    }
  }, [featuresLoading, selfReflectionEnabled, reload])

  async function saveGoals() {
    setSaving(true)
    try {
      await putStudyGoal({ weeklyHours, optedIn })
      await reload()
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save')
    } finally {
      setSaving(false)
    }
  }

  async function addJournal() {
    const text = journalDraft.trim()
    if (!text) return
    await createReflectionJournalEntry({ entryText: text })
    setJournalDraft('')
    await reload()
  }

  if (featuresLoading) {
    return (
      <LmsPage title="Study insights">
        <p className="text-sm text-slate-600">Loading…</p>
      </LmsPage>
    )
  }

  if (!selfReflectionEnabled) {
    return (
      <LmsPage title="Study insights">
        <p className="text-sm text-slate-600">Study insights are not enabled on this platform.</p>
        <Link to="/" className="mt-4 inline-block text-sm font-medium text-indigo-600">
          Back to dashboard
        </Link>
      </LmsPage>
    )
  }

  const hoursStudied = (stats?.timeOnTaskSecondsThisWeek ?? 0) / 3600
  const timeAllocation = stats?.timeAllocation ?? []
  const maxAlloc = Math.max(1, ...timeAllocation.map((r) => r.minutes))

  return (
    <LmsPage title="Study insights">
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <section className="rounded-2xl border border-slate-200 p-5 dark:border-neutral-700" aria-label="My goals">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">My goals</h2>
        <label className="mt-4 flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={optedIn}
            onChange={(e) => setOptedIn(e.target.checked)}
          />
          Enable study stats, coaching tips, and private journal
        </label>
        <div className="mt-4">
          <label htmlFor="weekly-hours" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            Weekly study goal: {weeklyHours} hours
          </label>
          <input
            id="weekly-hours"
            type="range"
            min={0}
            max={40}
            step={0.5}
            value={weeklyHours}
            onChange={(e) => setWeeklyHours(Number(e.target.value))}
            className="mt-2 w-full max-w-md"
            disabled={!optedIn}
          />
        </div>
        <button
          type="button"
          disabled={saving}
          onClick={() => void saveGoals()}
          className="mt-4 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save goals'}
        </button>
      </section>

      {optedIn ? (
        <>
          {stats ? (
            <>
              <section
                className="mt-8 rounded-2xl border border-slate-200 p-5 dark:border-neutral-700"
                aria-label="This week"
              >
                <h2 className="text-lg font-semibold">This week</h2>
                <p className="mt-2 text-sm" aria-label={`${stats.loginStreakDays}-day study streak`}>
                  Streak: {stats.loginStreakDays} day{stats.loginStreakDays === 1 ? '' : 's'}
                </p>
                <p className="mt-1 text-sm">
                  Time on task: {formatHours(hoursStudied)} hours
                  {stats.weeklyGoalHours != null && stats.weeklyGoalHours > 0
                    ? ` (${formatHours(hoursStudied)} of ${formatHours(stats.weeklyGoalHours)} toward your goal)`
                    : ''}
                </p>
              </section>

              {timeAllocation.length > 0 ? (
                <section
                  className="mt-8 rounded-2xl border border-slate-200 p-5 dark:border-neutral-700"
                  aria-label="Time allocation"
                >
                  <h2 className="text-lg font-semibold">Time by module (last 14 days)</h2>
                  <ul className="mt-4 space-y-3">
                    {timeAllocation.map((row) => (
                      <li key={row.moduleId}>
                        <div className="flex justify-between text-sm">
                          <span className="truncate pe-2">{row.moduleTitle}</span>
                          <span className="shrink-0 text-slate-600">{Math.round(row.minutes)} min</span>
                        </div>
                        <div className="mt-1 h-2 rounded-full bg-slate-100 dark:bg-neutral-800">
                          <div
                            className="h-full rounded-full bg-indigo-500"
                            style={{ width: `${Math.round((row.minutes / maxAlloc) * 100)}%` }}
                          />
                        </div>
                      </li>
                    ))}
                  </ul>
                </section>
              ) : null}
            </>
          ) : null}

          <section
            className="mt-8 rounded-2xl border border-slate-200 p-5 dark:border-neutral-700"
            aria-label="Private journal"
          >
            <h2 className="text-lg font-semibold">Private journal</h2>
            <p className="mt-1 text-xs text-slate-500">Only you can see these entries.</p>
            <textarea
              rows={3}
              maxLength={280}
              value={journalDraft}
              onChange={(e) => setJournalDraft(e.target.value)}
              className="mt-3 w-full rounded-xl border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              aria-label="New journal entry"
              placeholder="How did studying feel today?"
            />
            <button
              type="button"
              className="mt-2 rounded-lg bg-slate-800 px-3 py-1.5 text-sm font-medium text-white dark:bg-neutral-100 dark:text-neutral-900"
              disabled={!journalDraft.trim()}
              onClick={() => void addJournal().catch((e) => setError(String(e)))}
            >
              Add entry
            </button>
            <ul className="mt-6 space-y-3">
              {journal.map((e) => (
                <li
                  key={e.id}
                  data-testid="journal-entry"
                  className="rounded-xl bg-slate-50 px-4 py-3 text-sm dark:bg-neutral-900/60"
                >
                  <p className="text-xs text-slate-500">{new Date(e.createdAt).toLocaleString()}</p>
                  <p className="mt-1">{e.entryText}</p>
                  <button
                    type="button"
                    className="mt-2 text-xs text-red-600"
                    onClick={() =>
                      void deleteReflectionJournalEntry(e.id)
                        .then(reload)
                        .catch((err) => setError(String(err)))
                    }
                  >
                    Delete
                  </button>
                </li>
              ))}
            </ul>
          </section>

          <section
            className="mt-8 rounded-2xl border border-slate-200 p-5 dark:border-neutral-700"
            aria-label="Coaching tips"
          >
            <h2 className="text-lg font-semibold">Coaching tip history</h2>
            <ul className="mt-4 space-y-4">
              {tips.map((tip) => (
                <li
                  key={tip.id}
                  className="rounded-xl border border-amber-100 bg-amber-50/50 px-4 py-3 text-sm dark:border-amber-900/40 dark:bg-amber-950/20"
                >
                  <p className="text-xs text-slate-500">Week of {tip.weekOf}</p>
                  <p className="mt-1">{tip.tipText}</p>
                  <div className="mt-2 flex gap-2">
                    <button
                      type="button"
                      aria-label="Helpful"
                      className="text-xs underline"
                      onClick={() => void rateCoachingTip(tip.id, 1).then(reload)}
                    >
                      Helpful
                    </button>
                    <button
                      type="button"
                      aria-label="Not helpful"
                      className="text-xs underline"
                      onClick={() => void rateCoachingTip(tip.id, -1).then(reload)}
                    >
                      Not helpful
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        </>
      ) : null}
    </LmsPage>
  )
}
