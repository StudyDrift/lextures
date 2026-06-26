import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import {
  fetchQuizAttemptGrading,
  fetchQuizAttemptsList,
  putQuizAttemptGrading,
  type QuizAttemptGradingPayload,
  type QuizGradingQuestion,
} from '../../lib/courses-api'
import { ResizableSplitPane } from '../layout/resizable-split-pane'
import { SubmissionNavigator } from '../annotation/submission-navigator'
import { useSpeedGraderHotkeys } from '../annotation/speed-grader-shortcuts'
import type { GradedFilter } from '../annotation/submission-navigator-utils'
import { QuizResponseDisplay } from './quiz-response-display'
import { MathPlainText } from '../math/math-plain-text'
import {
  defaultQuizSubmissionIndex,
  filterQuizSubmissions,
  quizAttemptsToSubmissions,
  submissionsMatch,
} from './quiz-speed-grader-utils'
import type { ModuleAssignmentSubmissionApi } from '../../lib/courses-api'

export type QuizSpeedGraderBranchProps = {
  courseCode: string
  itemId: string
  quizTitle?: string
  presentation?: 'inline' | 'modal'
  modalOpen?: boolean
  onModalClose?: () => void
  initialStudentUserId?: string | null
}

export function QuizSpeedGraderBranch({
  courseCode,
  itemId,
  quizTitle = 'Quiz',
  presentation = 'modal',
  modalOpen = false,
  onModalClose,
  initialStudentUserId = null,
}: QuizSpeedGraderBranchProps) {
  const [allSubmissions, setAllSubmissions] = useState<ModuleAssignmentSubmissionApi[]>([])
  const [gradedFilter, setGradedFilter] = useState<GradedFilter>('all')
  const [idx, setIdx] = useState(0)
  const navRef = useRef({ submissions: allSubmissions, idx })
  const [loadError, setLoadError] = useState<string | null>(null)
  const [rosterLoading, setRosterLoading] = useState(false)

  const [grading, setGrading] = useState<QuizAttemptGradingPayload | null>(null)
  const [gradingLoading, setGradingLoading] = useState(false)
  const [gradingError, setGradingError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedFlash, setSavedFlash] = useState(false)
  const [scoreInputs, setScoreInputs] = useState<Record<number, string>>({})

  const submissions = useMemo(
    () => filterQuizSubmissions(allSubmissions, gradedFilter),
    [allSubmissions, gradedFilter],
  )
  navRef.current = { submissions, idx }

  const current = submissions[idx] ?? null
  const attemptId = current?.id ?? null

  const reloadRoster = useCallback(async () => {
    setRosterLoading(true)
    setLoadError(null)
    try {
      const data = await fetchQuizAttemptsList(courseCode, itemId)
      const sorted = quizAttemptsToSubmissions(data.attempts)
      const preserveCurrent = navRef.current.submissions[navRef.current.idx]
      setAllSubmissions(sorted)
      setIdx(() => {
        if (preserveCurrent) {
          const filtered = filterQuizSubmissions(sorted, gradedFilter)
          const nextIdx = filtered.findIndex((s) => submissionsMatch(s, preserveCurrent))
          if (nextIdx >= 0) return nextIdx
        }
        return defaultQuizSubmissionIndex(filterQuizSubmissions(sorted, gradedFilter), initialStudentUserId)
      })
    } catch (e) {
      setAllSubmissions([])
      setLoadError(e instanceof Error ? e.message : 'Could not load quiz attempts.')
    } finally {
      setRosterLoading(false)
    }
  }, [courseCode, gradedFilter, initialStudentUserId, itemId])

  const loadGrading = useCallback(
    async (id: string) => {
      setGradingLoading(true)
      setGradingError(null)
      setSaveError(null)
      try {
        const data = await fetchQuizAttemptGrading(courseCode, itemId, id)
        setGrading(data)
        const inputs: Record<number, string> = {}
        for (const q of data.questions) {
          inputs[q.questionIndex] =
            q.pointsAwarded != null && Number.isFinite(q.pointsAwarded)
              ? String(q.pointsAwarded)
              : ''
        }
        setScoreInputs(inputs)
      } catch (e) {
        setGrading(null)
        setScoreInputs({})
        setGradingError(e instanceof Error ? e.message : 'Could not load this attempt.')
      } finally {
        setGradingLoading(false)
      }
    },
    [courseCode, itemId],
  )

  useEffect(() => {
    if (presentation === 'modal' && !modalOpen) return
    void reloadRoster()
  }, [presentation, modalOpen, reloadRoster])

  useEffect(() => {
    if ((presentation === 'modal' && !modalOpen) || !attemptId) {
      setGrading(null)
      return
    }
    void loadGrading(attemptId)
  }, [attemptId, loadGrading, modalOpen, presentation])

  const handleGradeSaved = useCallback(() => {
    if (!current) return
    setAllSubmissions((prev) =>
      prev.map((row) => (submissionsMatch(row, current) ? { ...row, isGraded: true } : row)),
    )
  }, [current])

  useSpeedGraderHotkeys({
    enabled: presentation !== 'modal' || modalOpen,
    disabled: saving || rosterLoading,
    submissions,
    index: idx,
    onIndexChange: setIdx,
  })

  async function saveScores() {
    if (!grading || !attemptId) return
    setSaving(true)
    setSaveError(null)
    try {
      const questions = grading.questions
        .map((q) => {
          const raw = scoreInputs[q.questionIndex]?.trim() ?? ''
          if (raw === '') return null
          const pts = Number(raw)
          if (!Number.isFinite(pts)) return null
          return { questionIndex: q.questionIndex, pointsAwarded: pts }
        })
        .filter((q): q is { questionIndex: number; pointsAwarded: number } => q != null)

      if (questions.length === 0) {
        setSaveError('Enter a score for at least one question.')
        return
      }

      await putQuizAttemptGrading(courseCode, itemId, grading.attemptId, { questions })
      setSavedFlash(true)
      window.setTimeout(() => setSavedFlash(false), 2000)
      handleGradeSaved()
      await Promise.all([reloadRoster(), loadGrading(attemptId)])
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not save scores.')
    } finally {
      setSaving(false)
    }
  }

  const modalTitleId = useId()
  const modalCloseRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (presentation !== 'modal' || !modalOpen) return
    const t = window.setTimeout(() => modalCloseRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [presentation, modalOpen])

  useEffect(() => {
    if (presentation !== 'modal' || !modalOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onModalClose?.()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [presentation, modalOpen, onModalClose])

  if (presentation === 'modal' && !modalOpen) {
    return null
  }

  const sectionTitle = 'SpeedGrader'

  const mainContent =
    rosterLoading || gradingLoading ? (
      <div className="flex h-full min-h-[40vh] items-center justify-center text-sm text-slate-600 dark:text-neutral-400">
        Loading…
      </div>
    ) : gradingError ? (
      <div className="flex h-full min-h-[40vh] items-center justify-center px-4 text-sm text-rose-700 dark:text-rose-300">
        {gradingError}
      </div>
    ) : !grading || grading.questions.length === 0 ? (
      <div className="flex h-full min-h-[40vh] items-center justify-center px-4 text-sm text-slate-600 dark:text-neutral-400">
        {current?.submittedAt
          ? 'This attempt has no recorded answers to display.'
          : 'No submission from this student yet.'}
      </div>
    ) : (
      <div className="h-full min-h-[40vh] overflow-y-auto px-4 py-4">
        {grading.score ? (
          <p className="mb-4 text-sm text-slate-700 dark:text-neutral-200">
            Current total: {grading.score.pointsEarned}/{grading.score.pointsPossible} (
            {Math.round(grading.score.scorePercent)}%)
          </p>
        ) : null}
        <div className="space-y-4">
          {grading.questions.map((q, qi) => (
            <QuizQuestionGradeCard
              key={`${q.questionIndex}-${q.questionId ?? qi}`}
              question={q}
              scoreInput={scoreInputs[q.questionIndex] ?? ''}
              onScoreChange={(value) =>
                setScoreInputs((prev) => ({ ...prev, [q.questionIndex]: value }))
              }
            />
          ))}
        </div>
      </div>
    )

  const gradingSidebar = (
    <aside
      className="flex h-full min-h-0 w-full flex-col overflow-y-auto bg-slate-100 dark:bg-neutral-800"
      aria-label="Quiz grading"
    >
      <div className="border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
        <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Quiz attempt
        </p>
        {current ? (
          <p className="mt-1 text-sm font-medium text-slate-900 dark:text-neutral-100">
            {current.submittedByDisplayName ?? 'Student'}
          </p>
        ) : null}
        {current?.submittedAt ? (
          <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
            Submitted {formatDateTime(current.submittedAt, { dateStyle: 'medium', timeStyle: 'short' })}
          </p>
        ) : null}
      </div>
      <div className="flex flex-1 flex-col gap-3 px-4 py-4">
        {saveError ? (
          <p className="text-xs text-rose-700 dark:text-rose-300" role="alert">
            {saveError}
          </p>
        ) : null}
        {savedFlash ? (
          <p className="text-xs font-medium text-emerald-700 dark:text-emerald-300">Saved</p>
        ) : null}
        <button
          type="button"
          disabled={saving || !grading || grading.questions.length === 0}
          onClick={() => void saveScores()}
          className="w-full rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save scores'}
        </button>
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          Enter points per question, then save. Partial credit is supported.
        </p>
      </div>
    </aside>
  )

  const headerNav = (
    <SubmissionNavigator
      submissions={submissions}
      index={idx}
      onIndexChange={setIdx}
      gradedFilter={gradedFilter}
      onGradedFilterChange={(f) => {
        setGradedFilter(f)
        setIdx(0)
      }}
      disabled={saving || rosterLoading}
      showShortcuts
    />
  )

  if (presentation === 'modal') {
    return (
      <div className="fixed inset-0 z-[500] flex items-center justify-center p-3 sm:p-6" role="presentation">
        <button
          type="button"
          aria-label="Close SpeedGrader backdrop"
          className="absolute inset-0 cursor-default border-0 bg-slate-950/55 p-0 backdrop-blur-[2px] dark:bg-black/80"
          onClick={onModalClose}
          tabIndex={-1}
        />
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={modalTitleId}
          className="relative z-10 flex w-full max-w-[min(96vw,1600px)] flex-col overflow-hidden rounded-2xl border border-slate-300 bg-white shadow-[0_24px_80px_-12px_rgba(15,23,42,0.55)] ring-1 ring-slate-900/10 dark:border-neutral-500 dark:bg-neutral-900 dark:shadow-[0_24px_80px_-12px_rgba(0,0,0,0.85)] dark:ring-white/10"
          style={{ height: 'min(92vh, 1080px)', maxHeight: 'calc(100dvh - 1.5rem)' }}
        >
          <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 bg-slate-50 px-4 py-3 dark:border-neutral-600 dark:bg-neutral-800">
            <h2 id={modalTitleId} className="text-base font-semibold text-slate-900 dark:text-neutral-50">
              {sectionTitle} — {quizTitle}
            </h2>
            <div className="flex flex-1 flex-wrap items-center justify-end gap-2">{headerNav}</div>
            <button
              ref={modalCloseRef}
              type="button"
              onClick={onModalClose}
              className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
              aria-label="Close SpeedGrader"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {loadError ? (
            <p className="shrink-0 border-b border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
              {loadError}
            </p>
          ) : null}

          <ResizableSplitPane
            storageKey="lextures:quiz-grade-sidebar-width"
            primary={<div className="h-full min-h-[40vh] bg-slate-50 dark:bg-neutral-800/60">{mainContent}</div>}
            secondary={gradingSidebar}
          />
        </div>
      </div>
    )
  }

  return (
    <section
      id="submission-preview"
      tabIndex={-1}
      aria-label="Quiz SpeedGrader"
      className="scroll-mt-20 mt-8 space-y-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-950"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
          {sectionTitle} — {quizTitle}
        </h2>
        {headerNav}
      </div>
      {loadError ? (
        <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
          {loadError}
        </p>
      ) : null}
      <div className="min-h-[min(70vh,720px)]">
        <ResizableSplitPane
          storageKey="lextures:quiz-grade-sidebar-width-inline"
          primary={<div className="h-full min-h-0 bg-slate-50 dark:bg-neutral-800/60">{mainContent}</div>}
          secondary={gradingSidebar}
        />
      </div>
    </section>
  )
}

function questionStatusLabel(question: QuizGradingQuestion): string {
  if (question.needsGrading) return ' · Needs grading'
  if (question.pointsAwarded != null && Number.isFinite(question.pointsAwarded)) {
    return ` · ${question.pointsAwarded}/${question.maxPoints} pts`
  }
  return ''
}

function QuizQuestionGradeCard({
  question,
  scoreInput,
  onScoreChange,
}: {
  question: QuizGradingQuestion
  scoreInput: string
  onScoreChange: (value: string) => void
}) {
  return (
    <article
      className={`rounded-xl border p-4 ${
        question.needsGrading
          ? 'border-amber-200 bg-amber-50/50 dark:border-amber-900/50 dark:bg-amber-950/20'
          : 'border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900'
      }`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
            Question {question.questionIndex + 1}
            {questionStatusLabel(question)}
          </p>
          {question.promptSnapshot ? (
            <div className="mt-1 text-sm font-medium text-slate-900 dark:text-neutral-100">
              <MathPlainText text={question.promptSnapshot} />
            </div>
          ) : null}
        </div>
        <label className="shrink-0 text-xs text-slate-600 dark:text-neutral-400">
          Score
          <div className="mt-1 flex items-center gap-1">
            <input
              type="number"
              min={0}
              max={question.maxPoints > 0 ? question.maxPoints : undefined}
              step="any"
              value={scoreInput}
              onChange={(e) => onScoreChange(e.target.value)}
              data-speed-grader-score="true"
              className="w-20 rounded-lg border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
            <span className="text-sm text-slate-500 dark:text-neutral-400">/ {question.maxPoints}</span>
          </div>
        </label>
      </div>
      <div className="mt-3 rounded-lg border border-slate-100 bg-slate-50/80 p-3 dark:border-neutral-800 dark:bg-neutral-950/60">
        <p className="mb-1 text-xs font-medium text-slate-500 dark:text-neutral-400">Student answer</p>
        <QuizResponseDisplay
          responseJson={question.responseJson}
          questionType={question.questionType}
          choices={question.choices}
        />
      </div>
    </article>
  )
}