import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { LmsPage } from './lms-page'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import {
  fetchCourseStructure,
  fetchEnrollmentDiagnostic,
  learnerCourseItemHref,
  postDiagnosticBypass,
  postDiagnosticRespond,
  postDiagnosticStart,
  type AdaptiveQuizGeneratedQuestion,
  type CoursePublic,
  type CourseStructureItem,
} from '../../lib/courses-api'
import { MathPlainText } from '../../components/math/math-plain-text'

function findItemById(items: CourseStructureItem[], id: string): CourseStructureItem | null {
  for (const it of items) {
    if (it.id === id) return it
  }
  return null
}

type PlacementSummaryPayload = {
  concepts: Array<{
    conceptId: string
    name: string
    theta: number
    mastery: number
    proficiencyKey: string
    proficiencyLabel: string
  }>
  placementItemId: string
  placementTitle: string
}

type Phase =
  | { kind: 'loading' }
  | { kind: 'unavailable'; message: string }
  | { kind: 'intro' }
  | { kind: 'quiz'; attemptId: string; question: AdaptiveQuizGeneratedQuestion }
  | { kind: 'summary'; summary: PlacementSummaryPayload }

export default function CourseDiagnosticPage() {
  const { courseCode: courseCodeParam } = useParams<{ courseCode: string }>()
  const courseCode = courseCodeParam ? decodeURIComponent(courseCodeParam) : ''

  const [course, setCourse] = useState<CoursePublic | null>(null)
  const [phase, setPhase] = useState<Phase>({ kind: 'loading' })
  const [error, setError] = useState<string | null>(null)
  const [choice, setChoice] = useState<number | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const enrollmentId = course?.viewerStudentEnrollmentId ?? null

  const load = useCallback(async () => {
    if (!courseCode) return
    setError(null)
    setPhase({ kind: 'loading' })
    try {
      const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}`)
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setCourse(null)
        setPhase({ kind: 'unavailable', message: readApiErrorMessage(raw) })
        return
      }
      const c = raw as CoursePublic
      setCourse(c)
      const eid = c.viewerStudentEnrollmentId
      if (!eid) {
        setPhase({ kind: 'unavailable', message: 'Placement is only available to enrolled students.' })
        return
      }
      const g = await fetchEnrollmentDiagnostic(eid)
      if (g.status === 'off' || g.status === 'not_configured') {
        setPhase({
          kind: 'unavailable',
          message:
            g.status === 'off'
              ? 'Placement diagnostic is not enabled for this course on the server.'
              : 'Your instructor has not configured a placement diagnostic yet.',
        })
        return
      }
      if (g.status === 'completed' || g.status === 'bypassed') {
        setPhase({
          kind: 'unavailable',
          message:
            g.status === 'bypassed'
              ? 'You skipped the placement diagnostic for this course.'
              : 'You have already completed the placement diagnostic.',
        })
        return
      }
      if (g.status === 'in_progress' && g.attempt) {
        const start = await postDiagnosticStart(eid)
        setPhase({ kind: 'quiz', attemptId: start.attemptId, question: start.firstQuestion })
        return
      }
      setPhase({ kind: 'intro' })
    } catch (e) {
      setPhase({
        kind: 'unavailable',
        message: e instanceof Error ? e.message : 'Could not load placement.',
      })
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  const estimatedMinutes = useMemo(() => Math.max(5, Math.round(20 * 0.75)), [])

  const onStart = async () => {
    if (!enrollmentId) return
    setSubmitting(true)
    setError(null)
    try {
      const start = await postDiagnosticStart(enrollmentId)
      setChoice(null)
      setPhase({ kind: 'quiz', attemptId: start.attemptId, question: start.firstQuestion })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not start.')
    } finally {
      setSubmitting(false)
    }
  }

  const onSkip = async () => {
    if (!enrollmentId) return
    setSubmitting(true)
    setError(null)
    try {
      await postDiagnosticBypass(enrollmentId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not skip.')
    } finally {
      setSubmitting(false)
    }
  }

  const onSubmitAnswer = async () => {
    if (phase.kind !== 'quiz' || choice == null) return
    const q = phase.question
    const qid = q.questionId ?? ''
    if (!qid) {
      setError('This question is missing an identifier; you cannot continue.')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const out = await postDiagnosticRespond(phase.attemptId, {
        questionId: qid,
        choiceIndex: choice,
      })
      if (out.completed && out.summary) {
        setPhase({ kind: 'summary', summary: out.summary })
        return
      }
      if (out.nextQuestion) {
        setChoice(null)
        setPhase({ kind: 'quiz', attemptId: phase.attemptId, question: out.nextQuestion })
        return
      }
      setError('Unexpected response from server.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not submit.')
    } finally {
      setSubmitting(false)
    }
  }

  const [continueHref, setContinueHref] = useState<string | null>(null)

  useEffect(() => {
    if (phase.kind !== 'summary' || !courseCode) return
    let cancelled = false
    ;(async () => {
      try {
        const items = await fetchCourseStructure(courseCode)
        if (cancelled) return
        const it = findItemById(items, phase.summary.placementItemId)
        if (it) {
          setContinueHref(learnerCourseItemHref(courseCode, it))
        } else {
          setContinueHref(`/courses/${encodeURIComponent(courseCode)}/modules`)
        }
      } catch {
        if (!cancelled) {
          setContinueHref(`/courses/${encodeURIComponent(courseCode)}/modules`)
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [phase, courseCode])

  if (!courseCode) {
    return (
      <LmsPage title="Placement" description="">
        <p className="mt-6 text-sm text-fg-muted">Invalid link.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage
      title="Placement assessment"
      description="A short adaptive check so we can suggest where to begin in this course."
    >
      <div className="mt-2">
        <Link
          to={`/courses/${encodeURIComponent(courseCode)}`}
          className="text-sm font-medium text-accent-fg hover:text-indigo-500"
        >
          ← Back to course
        </Link>
      </div>

      {error && (
        <p className="mt-6 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100">
          {error}
        </p>
      )}

      {phase.kind === 'loading' && <p className="mt-8 text-sm text-fg-muted">Loading…</p>}

      {phase.kind === 'unavailable' && (
        <p className="mt-8 max-w-xl text-sm text-fg-muted">{phase.message}</p>
      )}

      {phase.kind === 'intro' && (
        <section className="mt-8 max-w-xl space-y-4 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-subtle dark:bg-surface-base">
          <h2 className="text-lg font-semibold text-fg-default">How this works</h2>
          <p className="text-sm text-fg-muted">
            You will answer a series of multiple-choice or true/false questions from the course
            question bank. The set adapts as you go (typically around {estimatedMinutes} minutes).
            You can skip and start from the beginning of the course if you prefer.
          </p>
          <div className="flex flex-wrap gap-3 pt-2">
            <button
              type="button"
              disabled={submitting}
              onClick={() => void onStart()}
              className="inline-flex rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:opacity-50"
            >
              Start placement assessment
            </button>
            <button
              type="button"
              disabled={submitting}
              onClick={() => void onSkip()}
              className="inline-flex rounded-xl border border-border-strong bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              Skip — start from the beginning
            </button>
          </div>
          <p className="text-xs text-fg-muted">
            <button
              type="button"
              className="font-medium text-accent-fg underline hover:text-indigo-500 dark:text-indigo-400"
              disabled={submitting}
              onClick={() => void onSkip()}
            >
              Skip placement (same as button)
            </button>{' '}
            for screen-reader users who prefer an inline control.
          </p>
        </section>
      )}

      {phase.kind === 'quiz' && (
        <section className="mt-8 max-w-2xl space-y-6 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-subtle dark:bg-surface-base">
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">
              Placement question
            </p>
            <div className="mt-2 text-base text-fg-default">
              <MathPlainText text={phase.question.prompt} />
            </div>
          </div>
          <ul className="space-y-2" role="list">
            {phase.question.choices.map((c, i) => {
              const selected = choice === i
              return (
                <li key={i}>
                  <button
                    type="button"
                    onClick={() => setChoice(i)}
                    className={`flex w-full items-start gap-3 rounded-xl border px-4 py-3 text-start text-sm transition-[background-color,color,border-color] ${ selected ? 'border-indigo-500 bg-indigo-50 dark:border-indigo-400 dark:bg-indigo-950/40' : 'border-border-default bg-surface-raised hover:border-border-strong dark:border-border-subtle dark:bg-surface-raised dark:hover:border-border-default' }`}
                  >
                    <span
                      className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-slate-400 text-[10px] font-bold dark:border-neutral-500"
                      aria-hidden
                    >
                      {String.fromCharCode(65 + i)}
                    </span>
                    <span className="text-fg-default">
                      <MathPlainText text={c} />
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
          <div className="flex justify-end border-t border-border-subtle pt-4 dark:border-border-subtle">
            <button
              type="button"
              disabled={submitting || choice == null}
              onClick={() => void onSubmitAnswer()}
              className="inline-flex rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:opacity-50"
            >
              Submit answer
            </button>
          </div>
        </section>
      )}

      {phase.kind === 'summary' && (
        <section className="mt-8 max-w-3xl space-y-6">
          <h2 className="text-lg font-semibold text-fg-default">Your results</h2>
          <p className="text-sm text-fg-muted">
            Recommended starting point:{' '}
            <span className="font-semibold text-fg-default">
              {phase.summary.placementTitle}
            </span>
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            {phase.summary.concepts.map((c) => (
              <div
                key={c.conceptId}
                className="rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm dark:border-border-subtle dark:bg-surface-base"
              >
                <p className="text-sm font-semibold text-fg-default">{c.name}</p>
                <p className="mt-2 text-sm text-fg-muted">
                  <span className="sr-only">Proficiency: </span>
                  <span className="inline-flex items-center gap-2">
                    <span aria-hidden className="text-lg">
                      {c.proficiencyLabel === 'Beginner'
                        ? '◔'
                        : c.proficiencyLabel === 'Developing'
                          ? '◑'
                          : c.proficiencyLabel === 'Proficient'
                            ? '●'
                            : '◉'}
                    </span>
                    <span>{c.proficiencyLabel}</span>
                  </span>
                </p>
              </div>
            ))}
          </div>
          {continueHref && (
            <Link
              to={continueHref}
              className="inline-flex rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500"
            >
              Continue to your starting point
            </Link>
          )}
        </section>
      )}
    </LmsPage>
  )
}
