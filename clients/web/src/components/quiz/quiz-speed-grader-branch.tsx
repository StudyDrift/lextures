import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { Check, CircleHelp, X } from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import {
  fetchCourseCanvasLink,
  fetchQuizAttemptGrading,
  fetchQuizAttemptsList,
  putQuizAttemptGrading,
  type CourseCanvasLinkApi,
  type QuizAttemptGradingPayload,
  type QuizGradingQuestion,
} from '../../lib/courses-api'
import { queueCanvasQuizGradeSync } from '../canvas/canvas-quiz-grade-sync'
import { ResizableSplitPane } from '../layout/resizable-split-pane'
import { FullScreenModalShell } from '../ui/fullscreen-modal-shell'
import { SubmissionNavigator } from '../annotation/submission-navigator'
import { useSpeedGraderHotkeys } from '../annotation/speed-grader-shortcuts'
import type { GradedFilter } from '../annotation/submission-navigator-utils'
import { QuizResponseDisplay } from './quiz-response-display'
import { formatQuizResponseText } from './quiz-response-format'
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
  const [savedMessage, setSavedMessage] = useState('Saved')
  const [scoreInputs, setScoreInputs] = useState<Record<number, string>>({})
  const [canvasLink, setCanvasLink] = useState<CourseCanvasLinkApi | null>(null)
  const [canvasSyncPending, setCanvasSyncPending] = useState(false)
  const canvasSyncAbortRef = useRef<(() => void) | null>(null)

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
    let cancelled = false
    void (async () => {
      try {
        const link = await fetchCourseCanvasLink(courseCode)
        if (!cancelled) setCanvasLink(link)
      } catch {
        if (!cancelled) setCanvasLink({ linked: false, gradeSyncEnabled: false })
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  useEffect(
    () => () => {
      canvasSyncAbortRef.current?.()
      canvasSyncAbortRef.current = null
    },
    [],
  )

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

      const saved = await putQuizAttemptGrading(courseCode, itemId, grading.attemptId, { questions })
      handleGradeSaved()
      await Promise.all([reloadRoster(), loadGrading(attemptId)])

      const startedCanvasSync = startCanvasSync(grading.attemptId, saved.pointsEarned)
      if (startedCanvasSync) {
        setSavedMessage('Scores saved. Syncing to Canvas…')
        setSavedFlash(true)
      } else {
        setSavedMessage('Saved')
        setSavedFlash(true)
        window.setTimeout(() => setSavedFlash(false), 2000)
      }
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not save scores.')
    } finally {
      setSaving(false)
    }
  }

  function startCanvasSync(attemptId: string, pointsEarned?: number): boolean {
    if (!canvasLink) return false
    canvasSyncAbortRef.current?.()
    const handle = queueCanvasQuizGradeSync({
      courseCode,
      itemId,
      attemptId,
      canvasLink,
      pointsEarned,
      onComplete: () => {
        setCanvasSyncPending(false)
        setSavedMessage('Scores saved and synced to Canvas.')
        setSavedFlash(true)
        window.setTimeout(() => setSavedFlash(false), 2500)
      },
      onError: (message) => {
        setCanvasSyncPending(false)
        setSaveError(message)
        setSavedFlash(false)
      },
    })
    if (handle) {
      canvasSyncAbortRef.current = handle.abort
      setCanvasSyncPending(true)
      return true
    }
    return false
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
      <div className="flex h-full min-h-[40vh] items-center justify-center text-sm text-fg-muted">
        Loading…
      </div>
    ) : gradingError ? (
      <div className="flex h-full min-h-[40vh] items-center justify-center px-4 text-sm text-rose-700 dark:text-rose-300">
        {gradingError}
      </div>
    ) : !grading || grading.questions.length === 0 ? (
      <div className="flex h-full min-h-[40vh] items-center justify-center px-4 text-sm text-fg-muted">
        {current?.submittedAt
          ? 'This attempt has no recorded answers to display.'
          : 'No submission from this student yet.'}
      </div>
    ) : (
      <div className="h-full min-h-[40vh] overflow-y-auto px-4 py-4">
        {grading.score ? (
          <p className="mb-4 text-sm text-fg-default">
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
      className="flex h-full min-h-0 w-full flex-col overflow-y-auto bg-surface-sunken"
      aria-label="Quiz grading"
    >
      <div className="border-b border-border-default px-4 py-3 dark:border-border-default">
        <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
          Quiz attempt
        </p>
        {current ? (
          <p className="mt-1 text-sm font-medium text-fg-default">
            {current.submittedByDisplayName ?? 'Student'}
          </p>
        ) : null}
        {current?.submittedAt ? (
          <p className="mt-0.5 text-xs text-fg-muted">
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
          <p className="text-xs font-medium text-emerald-700 dark:text-emerald-300">{savedMessage}</p>
        ) : null}
        <button
          type="button"
          disabled={saving || canvasSyncPending || !grading || grading.questions.length === 0}
          onClick={() => void saveScores()}
          className="w-full rounded-xl bg-accent-solid px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
        >
          {saving ? 'Saving…' : canvasSyncPending ? 'Syncing to Canvas…' : 'Save scores'}
        </button>
        <p className="text-xs text-fg-muted">
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
      <FullScreenModalShell
        open={modalOpen}
        onClose={onModalClose}
        backdropLabel="Close SpeedGrader backdrop"
      >
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={modalTitleId}
          className="relative z-10 flex w-full max-w-[min(96vw,1600px)] flex-col overflow-hidden rounded-2xl border border-border-strong bg-surface-raised shadow-[0_24px_80px_-12px_rgba(15,23,42,0.55)] ring-1 ring-slate-900/10 dark:border-neutral-500 dark:bg-surface-raised dark:shadow-[0_24px_80px_-12px_rgba(0,0,0,0.85)] dark:ring-white/10"
          style={{ height: 'min(92vh, 1080px)', maxHeight: 'calc(100dvh - 1.5rem)' }}
        >
          <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-border-default bg-surface-base px-4 py-3 dark:border-border-default dark:bg-surface-overlay">
            <h2 id={modalTitleId} className="text-base font-semibold text-fg-default">
              {sectionTitle} — {quizTitle}
            </h2>
            <div className="flex flex-1 flex-wrap items-center justify-end gap-2">{headerNav}</div>
            <button
              ref={modalCloseRef}
              type="button"
              onClick={onModalClose}
              className="rounded-lg p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-muted dark:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
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
            primary={<div className="h-full min-h-[40vh] bg-surface-sunken/60">{mainContent}</div>}
            secondary={gradingSidebar}
          />
        </div>
      </FullScreenModalShell>
    )
  }

  return (
    <section
      id="submission-preview"
      tabIndex={-1}
      aria-label="Quiz SpeedGrader"
      className="scroll-mt-20 mt-8 space-y-4 rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 dark:border-border-default dark:bg-surface-base"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-fg-default">
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
          primary={<div className="h-full min-h-0 bg-surface-sunken/60">{mainContent}</div>}
          secondary={gradingSidebar}
        />
      </div>
    </section>
  )
}

function questionPointsLabel(question: QuizGradingQuestion): string {
  if (question.needsGrading) return ''
  if (question.pointsAwarded != null && Number.isFinite(question.pointsAwarded)) {
    return ` · ${question.pointsAwarded}/${question.maxPoints} pts`
  }
  return ''
}

type QuestionStatus = 'needs-grading' | 'unanswered' | 'correct' | 'incorrect' | 'none'

// Question types that are auto-graded against a correct-answer key. Only these can be shown as
// correct/incorrect — subjective or keyless questions are never "wrong", just graded or ungraded.
const OBJECTIVE_QUESTION_TYPES = new Set(['multiple_choice', 'true_false', 'numeric'])

function questionStatus(question: QuizGradingQuestion, answered: boolean): QuestionStatus {
  if (question.needsGrading) return 'needs-grading'
  if (!answered) return 'unanswered'
  if (OBJECTIVE_QUESTION_TYPES.has(question.questionType)) {
    if (question.isCorrect === true) return 'correct'
    if (question.isCorrect === false) return 'incorrect'
  }
  return 'none'
}

function QuestionStatusBadge({ status }: { status: QuestionStatus }) {
  if (status === 'none') return null
  const config: Record<
    Exclude<QuestionStatus, 'none'>,
    { label: string; className: string; icon: typeof Check | null }
  > = {
    correct: {
      label: 'Correct',
      icon: Check,
      className:
        'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-300',
    },
    incorrect: {
      label: 'Wrong answer',
      icon: X,
      className: 'bg-rose-100 text-rose-800 dark:bg-rose-950/50 dark:text-rose-300',
    },
    'needs-grading': {
      label: 'Needs grading',
      icon: CircleHelp,
      className: 'bg-amber-100 text-amber-800 dark:bg-amber-950/50 dark:text-amber-300',
    },
    unanswered: {
      label: 'Not answered',
      icon: null,
      className: 'bg-slate-200 text-fg-muted dark:bg-surface-overlay dark:text-fg-muted',
    },
  }
  const { label, className, icon: Icon } = config[status]
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold ${className}`}
    >
      {Icon ? <Icon className="h-3.5 w-3.5" aria-hidden /> : null}
      {label}
    </span>
  )
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
  const answered =
    formatQuizResponseText(question.responseJson, question.questionType, question.choices ?? null) !==
    ''
  const status = questionStatus(question, answered)
  const cardBorder =
    status === 'needs-grading'
      ? 'border-amber-200 bg-amber-50/50 dark:border-amber-900/50 dark:bg-amber-950/20'
      : status === 'incorrect'
        ? 'border-rose-200 bg-rose-50/40 dark:border-rose-900/50 dark:bg-rose-950/20'
        : status === 'correct'
          ? 'border-emerald-200 bg-emerald-50/40 dark:border-emerald-900/50 dark:bg-emerald-950/20'
          : 'border-border-default bg-surface-raised dark:border-border-default dark:bg-surface-raised'
  return (
    <article className={`rounded-xl border p-4 ${cardBorder}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex flex-wrap items-center gap-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
              Question {question.questionIndex + 1}
              {questionPointsLabel(question)}
            </p>
            <QuestionStatusBadge status={status} />
          </div>
          {question.promptSnapshot ? (
            <div className="mt-1 text-sm font-medium text-fg-default">
              <MathPlainText text={question.promptSnapshot} />
            </div>
          ) : null}
        </div>
        <label className="shrink-0 text-xs text-fg-muted">
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
              className="w-20 rounded-lg border border-border-default px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            />
            <span className="text-sm text-fg-muted">/ {question.maxPoints}</span>
          </div>
        </label>
      </div>
      <div className="mt-3 rounded-lg border border-border-subtle bg-slate-50/80 p-3 dark:border-border-subtle/60">
        <p className="mb-1 text-xs font-medium text-fg-muted">Student answer</p>
        <QuizResponseDisplay
          responseJson={question.responseJson}
          questionType={question.questionType}
          choices={question.choices}
        />
      </div>
    </article>
  )
}