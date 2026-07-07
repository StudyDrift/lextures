import { useCallback, useEffect, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { ArrowRight, Sparkles } from 'lucide-react'
import { getAccountType } from '../../lib/auth'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchDiagnosticQuestions,
  fetchOnboardingStatus,
  grantMarketingConsent,
  ONBOARDING_TOPICS,
  postOnboarding,
  saveStudyReminderPrefs,
  type DiagnosticQuestion,
  type LearnerGoals,
  type PriorKnowledgeLevel,
} from '../../lib/onboarding-api'
import { OnboardingShell } from './onboarding-shell'

type WizardStep = 0 | 1 | 2 | 3 | 4 | 5 | 6

const EXPERIENCE_LEVELS = ['beginner', 'intermediate', 'advanced'] as const satisfies readonly PriorKnowledgeLevel[]

export default function OnboardingPage() {
  const { t } = useTranslation('onboarding')
  const navigate = useNavigate()
  const { ffOnboardingFlow, gdprModuleEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [step, setStep] = useState<WizardStep>(0)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [goals, setGoals] = useState<LearnerGoals | null>(null)

  const [topic, setTopic] = useState('')
  const [goalText, setGoalText] = useState('')
  const [targetDate, setTargetDate] = useState('')
  const [priorLevel, setPriorLevel] = useState<PriorKnowledgeLevel>('beginner')
  const [dailyMinutes, setDailyMinutes] = useState(20)
  const [reminderOptIn, setReminderOptIn] = useState(false)
  const [reminderTime, setReminderTime] = useState('09:00')
  const [termsAccepted, setTermsAccepted] = useState(false)
  const [marketingConsent, setMarketingConsent] = useState(false)

  const [questions, setQuestions] = useState<DiagnosticQuestion[]>([])
  const [questionIndex, setQuestionIndex] = useState(0)
  const [answers, setAnswers] = useState<Record<string, number>>({})

  const loadStatus = useCallback(async () => {
    setLoading(true)
    try {
      const status = await fetchOnboardingStatus()
      if (status?.completed) {
        navigate('/', { replace: true })
        return
      }
      if (status && status.step > 0) {
        setStep(Math.min(status.step, 6) as WizardStep)
      }
    } catch {
      /* first visit */
    } finally {
      setLoading(false)
    }
  }, [navigate])

  useEffect(() => {
    if (featuresLoading) return
    if (!ffOnboardingFlow) {
      setLoading(false)
      return
    }
    void loadStatus()
  }, [featuresLoading, ffOnboardingFlow, loadStatus])

  useEffect(() => {
    if (step !== 3 || !topic) return
    let cancelled = false
    void fetchDiagnosticQuestions(topic)
      .then((qs) => {
        if (!cancelled) {
          setQuestions(qs)
          setQuestionIndex(0)
          setAnswers({})
        }
      })
      .catch(() => {
        if (!cancelled) setQuestions([])
      })
    return () => {
      cancelled = true
    }
  }, [step, topic])

  if (getAccountType() === 'parent') {
    return <Navigate to="/parent" replace />
  }

  if (!featuresLoading && !ffOnboardingFlow) {
    return <Navigate to="/" replace />
  }

  if (loading || featuresLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-slate-50 dark:bg-neutral-950">
        <p className="text-sm text-slate-500 dark:text-neutral-400">{t('onboarding.loading')}</p>
      </div>
    )
  }

  async function saveStep(nextStep: WizardStep, body: Record<string, unknown> = {}) {
    setSubmitting(true)
    setError(null)
    try {
      const row = await postOnboarding({ step: nextStep, ...body })
      setGoals(row)
      setStep(nextStep)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('onboarding.errors.saveProgress'))
    } finally {
      setSubmitting(false)
    }
  }

  async function finishOnboarding(extra: Record<string, unknown> = {}) {
    setSubmitting(true)
    setError(null)
    try {
      if (marketingConsent && gdprModuleEnabled) {
        await grantMarketingConsent()
      }
      await saveStudyReminderPrefs(reminderOptIn, reminderTime)
      const row = await postOnboarding({
        step: 6,
        complete: true,
        termsAccepted: true,
        reminderOptIn,
        reminderTime,
        ...extra,
      })
      setGoals(row)
      setStep(6)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('onboarding.errors.complete'))
    } finally {
      setSubmitting(false)
    }
  }

  async function skipAll() {
    setSubmitting(true)
    setError(null)
    try {
      await postOnboarding({ skipAll: true })
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : t('onboarding.errors.skip'))
    } finally {
      setSubmitting(false)
    }
  }

  const errorBanner = error ? (
    <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100">
      {error}
    </p>
  ) : null

  if (step === 0) {
    return (
      <OnboardingShell step={0} title={t('onboarding.welcome.title')}>
        {errorBanner}
        <p className="text-sm text-slate-600 dark:text-neutral-400">{t('onboarding.welcome.description')}</p>
        <div className="mt-6 flex flex-wrap gap-3">
          <button
            type="button"
            disabled={submitting}
            onClick={() => void saveStep(1)}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
          >
            {t('onboarding.welcome.start')}
            <ArrowRight className="h-4 w-4" aria-hidden />
          </button>
          <button
            type="button"
            disabled={submitting}
            onClick={() => void skipAll()}
            className="rounded-xl border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('onboarding.welcome.skip')}
          </button>
        </div>
      </OnboardingShell>
    )
  }

  if (step === 1) {
    return (
      <OnboardingShell step={1} title={t('onboarding.goal.title')} onBack={() => setStep(0)}>
        {errorBanner}
        <p className="text-sm text-slate-600 dark:text-neutral-400">{t('onboarding.goal.description')}</p>
        <div className="mt-4 flex flex-wrap gap-2">
          {ONBOARDING_TOPICS.map((topicOption) => (
            <button
              key={topicOption.id}
              type="button"
              onClick={() => setTopic(topicOption.id)}
              className={`rounded-full border px-3 py-1.5 text-sm font-medium ${
                topic === topicOption.id
                  ? 'border-indigo-600 bg-indigo-50 text-indigo-800 dark:border-indigo-400 dark:bg-indigo-950 dark:text-indigo-100'
                  : 'border-slate-200 text-slate-700 dark:border-neutral-600 dark:text-neutral-300'
              }`}
            >
              {topicOption.label}
            </button>
          ))}
        </div>
        <label className="mt-4 block text-sm font-medium text-slate-700 dark:text-neutral-300" htmlFor="goal-text">
          {t('onboarding.goal.label')}
        </label>
        <input
          id="goal-text"
          type="text"
          value={goalText}
          onChange={(e) => setGoalText(e.target.value)}
          placeholder={t('onboarding.goal.placeholder')}
          className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
        <label className="mt-4 block text-sm font-medium text-slate-700 dark:text-neutral-300" htmlFor="target-date">
          {t('onboarding.goal.targetDate')}
        </label>
        <input
          id="target-date"
          type="date"
          value={targetDate}
          onChange={(e) => setTargetDate(e.target.value)}
          className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
        <button
          type="button"
          disabled={submitting || !topic}
          onClick={() =>
            void saveStep(2, {
              topic,
              goalText,
              targetDate: targetDate || null,
            })
          }
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {t('onboarding.goal.continue')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </button>
      </OnboardingShell>
    )
  }

  if (step === 2) {
    return (
      <OnboardingShell step={2} title={t('onboarding.experience.title')} onBack={() => setStep(1)}>
        {errorBanner}
        <fieldset>
          <legend className="sr-only">{t('onboarding.experience.legend')}</legend>
          <div className="space-y-2">
            {EXPERIENCE_LEVELS.map((value) => (
              <label
                key={value}
                className={`flex cursor-pointer flex-col rounded-xl border p-4 ${
                  priorLevel === value
                    ? 'border-indigo-600 bg-indigo-50 dark:border-indigo-400 dark:bg-indigo-950/40'
                    : 'border-slate-200 dark:border-neutral-700'
                }`}
              >
                <span className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="priorLevel"
                    checked={priorLevel === value}
                    onChange={() => setPriorLevel(value)}
                  />
                  <span className="font-medium text-slate-900 dark:text-neutral-100">
                    {t(`onboarding.experience.${value}.label`)}
                  </span>
                </span>
                <span className="mt-1 pl-6 text-xs text-slate-500 dark:text-neutral-400">
                  {t(`onboarding.experience.${value}.hint`)}
                </span>
              </label>
            ))}
          </div>
        </fieldset>
        <button
          type="button"
          disabled={submitting}
          onClick={() => void saveStep(3, { priorKnowledgeLevel: priorLevel })}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {t('onboarding.goal.continue')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </button>
      </OnboardingShell>
    )
  }

  if (step === 3) {
    const q = questions[questionIndex]
    return (
      <OnboardingShell step={3} title={t('onboarding.diagnostic.title')} onBack={() => setStep(2)}>
        {errorBanner}
        <p className="text-sm text-slate-600 dark:text-neutral-400">{t('onboarding.diagnostic.description')}</p>
        {q ? (
          <div className="mt-4">
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('onboarding.diagnostic.questionProgress', {
                current: questionIndex + 1,
                total: questions.length,
              })}
            </p>
            <p className="mt-2 font-medium text-slate-900 dark:text-neutral-100">{q.prompt}</p>
            <div className="mt-3 space-y-2">
              {q.choices.map((choice, idx) => (
                <label
                  key={choice}
                  className={`flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm ${
                    answers[q.id] === idx
                      ? 'border-indigo-600 bg-indigo-50 dark:border-indigo-400 dark:bg-indigo-950/40'
                      : 'border-slate-200 dark:border-neutral-700'
                  }`}
                >
                  <input
                    type="radio"
                    name={q.id}
                    checked={answers[q.id] === idx}
                    onChange={() => setAnswers((prev) => ({ ...prev, [q.id]: idx }))}
                  />
                  {choice}
                </label>
              ))}
            </div>
            <div className="mt-4 flex flex-wrap gap-3">
              {questionIndex < questions.length - 1 ? (
                <button
                  type="button"
                  disabled={answers[q.id] === undefined || submitting}
                  onClick={() => setQuestionIndex((i) => i + 1)}
                  className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
                >
                  {t('onboarding.diagnostic.nextQuestion')}
                </button>
              ) : (
                <button
                  type="button"
                  disabled={answers[q.id] === undefined || submitting}
                  onClick={() => void saveStep(4, { diagnosticAnswers: answers })}
                  className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
                >
                  {t('onboarding.goal.continue')}
                </button>
              )}
            </div>
          </div>
        ) : null}
        <button
          type="button"
          disabled={submitting}
          onClick={() => void saveStep(4, { skipDiagnostic: true })}
          className="mt-4 text-sm font-medium text-slate-600 underline hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
        >
          {t('onboarding.diagnostic.skip')}
        </button>
      </OnboardingShell>
    )
  }

  if (step === 4) {
    return (
      <OnboardingShell step={4} title={t('onboarding.habits.title')} onBack={() => setStep(3)}>
        {errorBanner}
        <label className="block text-sm font-medium text-slate-700 dark:text-neutral-300" htmlFor="daily-minutes">
          {t('onboarding.habits.dailyGoal')}
        </label>
        <input
          id="daily-minutes"
          type="number"
          min={5}
          max={480}
          value={dailyMinutes}
          onChange={(e) => setDailyMinutes(Number(e.target.value))}
          className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
        <div className="mt-4 rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
          <label className="flex items-start gap-3">
            <input
              type="checkbox"
              checked={reminderOptIn}
              onChange={(e) => setReminderOptIn(e.target.checked)}
              className="mt-1"
            />
            <span>
              <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                {t('onboarding.habits.reminders.label')}
              </span>
              <span className="block text-xs text-slate-500 dark:text-neutral-400">
                {t('onboarding.habits.reminders.hint')}
              </span>
            </span>
          </label>
          {reminderOptIn ? (
            <label className="mt-3 block text-sm text-slate-700 dark:text-neutral-300" htmlFor="reminder-time">
              {t('onboarding.habits.reminders.time')}
              <input
                id="reminder-time"
                type="time"
                value={reminderTime}
                onChange={(e) => setReminderTime(e.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              />
            </label>
          ) : null}
        </div>
        <button
          type="button"
          disabled={submitting}
          onClick={() => void saveStep(5, { dailyMinutes, reminderOptIn, reminderTime })}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {t('onboarding.goal.continue')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </button>
      </OnboardingShell>
    )
  }

  if (step === 5) {
    return (
      <OnboardingShell step={5} title={t('onboarding.consent.title')} onBack={() => setStep(4)}>
        {errorBanner}
        <div className="space-y-4">
          <label className="flex items-start gap-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
            <input
              type="checkbox"
              checked={termsAccepted}
              onChange={(e) => setTermsAccepted(e.target.checked)}
              className="mt-1"
              required
            />
            <span className="text-sm text-slate-700 dark:text-neutral-300">
              <Trans
                i18nKey="onboarding.consent.terms"
                ns="onboarding"
                components={{
                  termsLink: (
                    <Link
                      to="/trust"
                      className="font-medium text-indigo-600 underline dark:text-indigo-400"
                    />
                  ),
                }}
              />
            </span>
          </label>
          {gdprModuleEnabled ? (
            <label className="flex items-start gap-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-700">
              <input
                type="checkbox"
                checked={marketingConsent}
                onChange={(e) => setMarketingConsent(e.target.checked)}
                className="mt-1"
              />
              <span className="text-sm text-slate-700 dark:text-neutral-300">{t('onboarding.consent.marketing')}</span>
            </label>
          ) : null}
        </div>
        <button
          type="button"
          disabled={submitting || !termsAccepted}
          onClick={() => void finishOnboarding()}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          {t('onboarding.consent.finish')}
          <ArrowRight className="h-4 w-4" aria-hidden />
        </button>
      </OnboardingShell>
    )
  }

  const recommended = goals?.recommendedCourseTitle ?? goals?.recommendedCourseCode

  return (
    <OnboardingShell step={6} title={t('onboarding.done.title')}>
      <p className="text-sm text-slate-600 dark:text-neutral-400">{t('onboarding.done.description')}</p>
      {recommended ? (
        <article className="mt-4 rounded-xl border border-emerald-100 bg-emerald-50/80 p-4 dark:border-emerald-900/40 dark:bg-emerald-950/30">
          <div className="flex items-center gap-2 text-xs font-medium text-emerald-800 dark:text-emerald-200">
            <Sparkles className="h-4 w-4" aria-hidden />
            {t('onboarding.done.startHere')}
          </div>
          <p className="mt-2 font-semibold text-slate-900 dark:text-neutral-50">{recommended}</p>
          {goals?.recommendedCourseCode ? (
            <Link
              to={`/courses/${encodeURIComponent(goals.recommendedCourseCode)}`}
              className="mt-3 inline-flex items-center gap-2 text-sm font-semibold text-emerald-700 hover:text-emerald-600 dark:text-emerald-300"
            >
              {t('onboarding.done.openCourse')}
              <ArrowRight className="h-4 w-4" aria-hidden />
            </Link>
          ) : null}
        </article>
      ) : (
        <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">{t('onboarding.done.browseCatalog')}</p>
      )}
      <Link
        to="/"
        className="mt-6 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500"
      >
        {t('onboarding.done.goToDashboard')}
        <ArrowRight className="h-4 w-4" aria-hidden />
      </Link>
    </OnboardingShell>
  )
}
