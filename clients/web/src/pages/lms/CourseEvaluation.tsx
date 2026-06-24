import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  fetchEvaluationStatus,
  submitEvaluation,
  type EvaluationQuestion,
  type EvaluationStatus,
} from '../../lib/course-evaluations-api'
import { authorizedFetch } from '../../lib/api'
import { LmsPage } from './lms-page'

function RatingQuestion({
  question,
  index,
  value,
  onChange,
}: {
  question: EvaluationQuestion
  index: number
  value: string
  onChange: (val: string) => void
}) {
  const ratings = ['1', '2', '3', '4', '5']
  const labels: Record<string, string> = {
    '1': '1 – Strongly Disagree',
    '2': '2 – Disagree',
    '3': '3 – Neutral',
    '4': '4 – Agree',
    '5': '5 – Strongly Agree',
  }
  return (
    <fieldset className="mb-6">
      <legend className="mb-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
        {index + 1}. {question.text}
        {question.required && <span className="ml-1 text-red-500">*</span>}
      </legend>
      <div
        role="radiogroup"
        aria-label={question.text}
        className="flex flex-wrap gap-2"
      >
        {ratings.map((r) => (
          <label
            key={r}
            className={`flex cursor-pointer items-center gap-1.5 rounded-lg border px-3 py-2 text-sm transition-[background-color,color,border-color] ${
              value === r
                ? 'border-indigo-500 bg-indigo-50 font-semibold text-indigo-700 ring-2 ring-indigo-400/30 dark:border-indigo-400 dark:bg-indigo-950/40 dark:text-indigo-300'
                : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300'
            }`}
          >
            <input
              type="radio"
              name={`q-${index}`}
              value={r}
              checked={value === r}
              onChange={() => onChange(r)}
              className="sr-only"
              aria-label={labels[r]}
            />
            {r}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

function MultipleChoiceQuestion({
  question,
  index,
  value,
  onChange,
}: {
  question: EvaluationQuestion
  index: number
  value: string
  onChange: (val: string) => void
}) {
  return (
    <fieldset className="mb-6">
      <legend className="mb-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
        {index + 1}. {question.text}
        {question.required && <span className="ml-1 text-red-500">*</span>}
      </legend>
      <div className="flex flex-col gap-2">
        {(question.options ?? []).map((opt) => (
          <label
            key={opt}
            className="flex cursor-pointer items-center gap-2 text-sm text-slate-700 dark:text-neutral-300"
          >
            <input
              type="radio"
              name={`q-${index}`}
              value={opt}
              checked={value === opt}
              onChange={() => onChange(opt)}
              className="h-4 w-4 accent-indigo-500"
            />
            {opt}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

function OpenTextQuestion({
  question,
  index,
  value,
  onChange,
}: {
  question: EvaluationQuestion
  index: number
  value: string
  onChange: (val: string) => void
}) {
  return (
    <div className="mb-6">
      <label
        htmlFor={`q-${index}`}
        className="mb-1.5 block text-sm font-medium text-slate-900 dark:text-neutral-100"
      >
        {index + 1}. {question.text}
        {question.required && <span className="ml-1 text-red-500">*</span>}
      </label>
      <textarea
        id={`q-${index}`}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        className="w-full resize-y rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder-neutral-500"
        placeholder="Your response (optional)"
      />
    </div>
  )
}

export default function CourseEvaluation() {
  const { courseCode } = useParams<{ courseCode: string }>()

  const [status, setStatus] = useState<EvaluationStatus | null>(null)
  const [questions, setQuestions] = useState<EvaluationQuestion[]>([])
  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    try {
      const s = await fetchEvaluationStatus(courseCode)
      setStatus(s)
      if (s.hasSubmitted) {
        setSubmitted(true)
      }
      if (s.windowOpen && s.windowId && !s.hasSubmitted) {
        // Load template questions via admin API (student-accessible subset)
        const tmplRes = await authorizedFetch(
          `/api/v1/admin/evaluation-templates`,
        )
        if (tmplRes.ok) {
          const body = await tmplRes.json() as { templates?: { questions: EvaluationQuestion[] }[] }
          if (body.templates && body.templates.length > 0) {
            setQuestions(body.templates[0].questions ?? [])
          }
        }
      }
    } catch {
      setError('Failed to load evaluation.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    load()
  }, [load])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!courseCode || !status?.windowId) return

    // Validate required questions.
    const missing = questions.some((q, i) => q.required && !answers[String(i)])
    if (missing) {
      setError('Please answer all required questions before submitting.')
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await submitEvaluation(courseCode, status.windowId, answers)
      setSubmitted(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit evaluation.')
    } finally {
      setSubmitting(false)
    }
  }

  const setAnswer = (index: number, val: string) => {
    setAnswers((prev) => ({ ...prev, [String(index)]: val }))
  }

  if (loading) {
    return (
      <LmsPage title="Course Evaluation">
        <div className="flex items-center justify-center py-20">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
        </div>
      </LmsPage>
    )
  }

  if (submitted) {
    return (
      <LmsPage title="Course Evaluation">
        <div className="mx-auto max-w-xl py-16 text-center">
          <div className="mb-4 text-4xl">✓</div>
          <h2 className="mb-2 text-xl font-semibold text-slate-900 dark:text-neutral-100">
            Thank you!
          </h2>
          <p className="text-slate-600 dark:text-neutral-400">
            Your response has been recorded. Your feedback is anonymous.
          </p>
        </div>
      </LmsPage>
    )
  }

  if (!status?.windowOpen) {
    return (
      <LmsPage title="Course Evaluation">
        <div className="mx-auto max-w-xl py-16 text-center">
          <h2 className="mb-2 text-xl font-semibold text-slate-900 dark:text-neutral-100">
            No evaluation open
          </h2>
          <p className="text-slate-600 dark:text-neutral-400">
            There is no active course evaluation at this time.
          </p>
        </div>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Course Evaluation">
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="mb-6 rounded-xl border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-800 dark:bg-indigo-950/30">
          <p className="text-sm text-indigo-800 dark:text-indigo-300">
            <strong>Your feedback is anonymous.</strong> Your responses cannot be linked back to
            you. Please be honest so instructors can improve.
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          {questions.map((q, i) => {
            if (q.type === 'rating') {
              return (
                <RatingQuestion
                  key={i}
                  question={q}
                  index={i}
                  value={answers[String(i)] ?? ''}
                  onChange={(v) => setAnswer(i, v)}
                />
              )
            }
            if (q.type === 'multiple_choice') {
              return (
                <MultipleChoiceQuestion
                  key={i}
                  question={q}
                  index={i}
                  value={answers[String(i)] ?? ''}
                  onChange={(v) => setAnswer(i, v)}
                />
              )
            }
            return (
              <OpenTextQuestion
                key={i}
                question={q}
                index={i}
                value={answers[String(i)] ?? ''}
                onChange={(v) => setAnswer(i, v)}
              />
            )
          })}

          {error && (
            <p className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-xl bg-indigo-600 px-6 py-3 text-sm font-semibold text-white transition-[background-color,color,border-color] hover:bg-indigo-700 disabled:opacity-60 dark:bg-indigo-500 dark:hover:bg-indigo-600"
          >
            {submitting ? 'Submitting…' : 'Submit Evaluation'}
          </button>
        </form>
      </div>
    </LmsPage>
  )
}
