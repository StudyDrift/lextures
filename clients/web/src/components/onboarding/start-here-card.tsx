import { Link } from 'react-router-dom'
import { ArrowRight, Sparkles } from 'lucide-react'
import { useEffect, useState } from 'react'
import { fetchLearnerGoals, type LearnerGoals } from '../../lib/onboarding-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

export function StartHereCard() {
  const { ffOnboardingFlow, loading: featuresLoading } = usePlatformFeatures()
  const [goals, setGoals] = useState<LearnerGoals | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (featuresLoading || !ffOnboardingFlow) {
      setLoading(false)
      return
    }
    let cancelled = false
    void fetchLearnerGoals()
      .then((g) => {
        if (!cancelled) setGoals(g)
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

  if (featuresLoading || loading || !ffOnboardingFlow) return null
  if (!goals?.onboardingCompleted || !goals.recommendedCourseCode) return null

  const courseTitle = goals.recommendedCourseTitle ?? goals.recommendedCourseCode
  const topicLabel = goals.topic ? goals.topic.replace(/-/g, ' ') : 'your goal'

  return (
    <section aria-label="Start here recommendation">
      <article className="rounded-2xl border border-emerald-100 bg-gradient-to-br from-emerald-50/90 to-white p-5 shadow-sm dark:border-emerald-900/40 dark:from-emerald-950/30 dark:to-neutral-900">
        <div className="flex flex-wrap items-center gap-2 text-xs font-medium text-emerald-800 dark:text-emerald-200">
          <Sparkles className="h-4 w-4 shrink-0" aria-hidden />
          <span>Start here</span>
        </div>
        <h2 className="mt-2 text-lg font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
          {courseTitle}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Based on your goal to learn {topicLabel}
          {goals.goalText ? `: ${goals.goalText}` : ''}.
        </p>
        <Link
          to={`/courses/${encodeURIComponent(goals.recommendedCourseCode)}`}
          className="mt-4 inline-flex items-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-emerald-500"
        >
          Open course
          <ArrowRight className="h-4 w-4" aria-hidden />
        </Link>
      </article>
    </section>
  )
}
