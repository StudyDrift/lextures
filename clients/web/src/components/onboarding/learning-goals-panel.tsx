import { type FormEvent, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchLearnerGoals,
  ONBOARDING_TOPICS,
  patchLearnerGoals,
  type LearnerGoals,
  type PriorKnowledgeLevel,
} from '../../lib/onboarding-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

export function LearningGoalsPanel() {
  const { ffOnboardingFlow, loading: featuresLoading } = usePlatformFeatures()
  const [goals, setGoals] = useState<LearnerGoals | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [topic, setTopic] = useState('')
  const [goalText, setGoalText] = useState('')
  const [targetDate, setTargetDate] = useState('')
  const [dailyMinutes, setDailyMinutes] = useState(20)
  const [priorLevel, setPriorLevel] = useState<PriorKnowledgeLevel>('beginner')

  useEffect(() => {
    if (featuresLoading || !ffOnboardingFlow) {
      setLoading(false)
      return
    }
    let cancelled = false
    void fetchLearnerGoals()
      .then((g) => {
        if (cancelled) return
        setGoals(g)
        if (g) {
          setTopic(g.topic)
          setGoalText(g.goalText ?? '')
          setTargetDate(g.targetDate?.slice(0, 10) ?? '')
          setDailyMinutes(g.dailyMinutes)
          setPriorLevel(g.priorKnowledgeLevel)
        }
      })
      .catch(() => {
        if (!cancelled) setGoals(null)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffOnboardingFlow])

  if (!featuresLoading && !ffOnboardingFlow) return null

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    try {
      const updated = await patchLearnerGoals({
        topic,
        goalText,
        targetDate: targetDate || null,
        dailyMinutes,
        priorKnowledgeLevel: priorLevel,
      })
      setGoals(updated)
      toastSaveOk('Learning goals saved.')
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not save goals.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="mt-8 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <h3 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Learning goals</h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Update what you want to learn and your daily study target.
      </p>
      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      ) : (
        <form onSubmit={(e) => void onSubmit(e)} className="mt-4 space-y-4">
          <div>
            <label htmlFor="goals-topic" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Topic
            </label>
            <select
              id="goals-topic"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            >
              <option value="">Select a topic</option>
              {ONBOARDING_TOPICS.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="goals-text" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Goal
            </label>
            <input
              id="goals-text"
              type="text"
              value={goalText}
              onChange={(e) => setGoalText(e.target.value)}
              placeholder="e.g. I want to learn Python by July"
              className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="goals-date" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Target date
              </label>
              <input
                id="goals-date"
                type="date"
                value={targetDate}
                onChange={(e) => setTargetDate(e.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              />
            </div>
            <div>
              <label htmlFor="goals-minutes" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Daily minutes
              </label>
              <input
                id="goals-minutes"
                type="number"
                min={5}
                max={480}
                value={dailyMinutes}
                onChange={(e) => setDailyMinutes(Number(e.target.value))}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              />
            </div>
          </div>
          <fieldset>
            <legend className="text-sm font-medium text-slate-700 dark:text-neutral-300">Experience level</legend>
            <div className="mt-2 flex flex-wrap gap-2">
              {(
                [
                  ['beginner', 'Beginner'],
                  ['intermediate', 'Some experience'],
                  ['advanced', 'Advanced'],
                ] as const
              ).map(([value, label]) => (
                <label
                  key={value}
                  className={`cursor-pointer rounded-full border px-3 py-1.5 text-sm ${
                    priorLevel === value
                      ? 'border-indigo-600 bg-indigo-50 text-indigo-800 dark:border-indigo-400 dark:bg-indigo-950 dark:text-indigo-100'
                      : 'border-slate-200 text-slate-700 dark:border-neutral-600 dark:text-neutral-300'
                  }`}
                >
                  <input
                    type="radio"
                    name="priorLevel"
                    value={value}
                    checked={priorLevel === value}
                    onChange={() => setPriorLevel(value)}
                    className="sr-only"
                  />
                  {label}
                </label>
              ))}
            </div>
          </fieldset>
          {goals?.recommendedCourseTitle ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              Recommended course: {goals.recommendedCourseTitle}
            </p>
          ) : null}
          <button
            type="submit"
            disabled={saving}
            className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
          >
            {saving ? 'Saving…' : 'Save goals'}
          </button>
        </form>
      )}
    </section>
  )
}
