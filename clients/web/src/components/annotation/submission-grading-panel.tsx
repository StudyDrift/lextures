import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { GraderAgentDrawer } from './grader-agent-drawer'
import { postGraderAgentRegradeRequest } from '../../lib/courses-api'
import {
  fetchAssignmentStudentGrade,
  fetchCourseCanvasLink,
  fetchCourseEnrollmentsList,
  fetchModuleAssignment,
  fetchSubmissionGrade,
  putAssignmentStudentGrade,
  putSubmissionGrade,
  type CourseCanvasLinkApi,
  type GradeCommentApi,
  type RubricDefinition,
  type SubmissionGradeApi,
} from '../../lib/courses-api'
import {
  SubmissionCommentThread,
  type CommentRosterPerson,
} from './submission-comment-thread'
import { queueCanvasGradeSync, type CanvasGradePushPayload } from '../canvas/canvas-grade-sync'
import { RubricGradePicker } from '../grading/rubric-grade-picker'
import { formatPointsCell, rubricScoresComplete, rubricTotal } from '../../lib/rubric-utils'
import { altKeyHint } from './speed-grader-shortcuts'

type GradeMode = 'rubric' | 'points'

type SubmissionGradingPanelProps = {
  mode?: 'staff' | 'student'
  courseCode: string
  itemId: string
  submissionId: string | null
  /** Used when the roster row has no submission yet. */
  studentUserId?: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  disabled?: boolean
  /** Increment to reload grade from server (e.g. after Canvas sync from toolbar). */
  gradeRefreshKey?: number
  /** Focus the score input after the grade loads (SpeedGrader). */
  autoFocusScore?: boolean
  onGradeSaved?: () => void
  onGradeCleared?: () => void
}

function initialGradeMode(grade: SubmissionGradeApi, hasRubric: boolean): GradeMode {
  if (!hasRubric) return 'points'
  if (grade.rubricScores && Object.keys(grade.rubricScores).length > 0) return 'rubric'
  if (grade.pointsEarned != null && Number.isFinite(grade.pointsEarned)) return 'points'
  return 'rubric'
}

export function SubmissionGradingPanel({
  mode = 'staff',
  courseCode,
  itemId,
  submissionId,
  studentUserId = null,
  rubric: rubricProp,
  maxPoints,
  disabled = false,
  gradeRefreshKey = 0,
  autoFocusScore = false,
  onGradeSaved,
  onGradeCleared,
}: SubmissionGradingPanelProps) {
  const { t } = useTranslation('common')
  const { graderAgentEnabled } = usePlatformFeatures()
  const [agentOpen, setAgentOpen] = useState(false)
  const [agentApplyKey, setAgentApplyKey] = useState(0)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedFlash, setSavedFlash] = useState(false)
  const [comment, setComment] = useState('')
  const [threadComments, setThreadComments] = useState<GradeCommentApi[]>([])
  const [rosterByUserId, setRosterByUserId] = useState<Map<string, CommentRosterPerson>>(
    () => new Map(),
  )
  const [pointsInput, setPointsInput] = useState('')
  const [rubricScores, setRubricScores] = useState<Record<string, number>>({})
  const [posted, setPosted] = useState(false)
  const [gradedByAi, setGradedByAi] = useState(false)
  const [gradeMode, setGradeMode] = useState<GradeMode>('points')
  const [fetchedRubric, setFetchedRubric] = useState<RubricDefinition | null>(null)
  const [hasGrade, setHasGrade] = useState(false)
  const [flashMsg, setFlashMsg] = useState('')
  const [canvasLink, setCanvasLink] = useState<CourseCanvasLinkApi | null>(null)
  const [canvasSyncPending, setCanvasSyncPending] = useState(false)
  const scoreInputRef = useRef<HTMLInputElement>(null)
  const canvasSyncAbortRef = useRef<(() => void) | null>(null)
  const focusTarget = submissionId ?? studentUserId ?? null
  const focusTargetRef = useRef(focusTarget)
  focusTargetRef.current = focusTarget
  const pendingScoreFocusRef = useRef<string | null>(null)

  const rubric = rubricProp ?? fetchedRubric
  const hasRubric = Boolean(rubric && rubric.criteria.length > 0)

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

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const roster = await fetchCourseEnrollmentsList(courseCode)
        if (cancelled) return
        const map = new Map<string, CommentRosterPerson>()
        for (const person of roster) {
          map.set(person.userId.toLowerCase(), {
            userId: person.userId,
            displayName: person.displayName,
            avatarUrl: person.avatarUrl,
          })
        }
        setRosterByUserId(map)
      } catch {
        if (!cancelled) setRosterByUserId(new Map())
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  useEffect(() => {
    canvasSyncAbortRef.current?.()
    canvasSyncAbortRef.current = null
    setCanvasSyncPending(false)
  }, [submissionId])

  useEffect(() => {
    return () => {
      canvasSyncAbortRef.current?.()
    }
  }, [])

  useEffect(() => {
    if (rubricProp) return
    let cancelled = false
    void (async () => {
      try {
        const data = await fetchModuleAssignment(courseCode, itemId)
        if (!cancelled) setFetchedRubric(data.rubric)
      } catch {
        if (!cancelled) setFetchedRubric(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, itemId, rubricProp])

  const applyGrade = useCallback(
    (grade: SubmissionGradeApi, options?: { preserveDraft?: boolean }) => {
      setThreadComments(grade.comments ?? [])
      if (!options?.preserveDraft) {
        setComment('')
      }
      setPosted(Boolean(grade.posted))
      setGradedByAi(Boolean(grade.gradedByAi))
      setGradeMode(initialGradeMode(grade, hasRubric))
      const hasPoints = grade.pointsEarned != null && Number.isFinite(grade.pointsEarned)
      const hasRubricScores = Boolean(grade.rubricScores && Object.keys(grade.rubricScores).length > 0)
      setHasGrade(hasPoints || hasRubricScores)
      if (grade.rubricScores && Object.keys(grade.rubricScores).length > 0) {
        setRubricScores(grade.rubricScores)
        setPointsInput('')
      } else if (grade.pointsEarned != null && Number.isFinite(grade.pointsEarned)) {
        setPointsInput(formatPointsCell(grade.pointsEarned))
        setRubricScores({})
      } else {
        setPointsInput('')
        setRubricScores({})
      }
    },
    [hasRubric],
  )

  useEffect(() => {
    if (!submissionId && !studentUserId) {
      setComment('')
      setThreadComments([])
      setPointsInput('')
      setRubricScores({})
      setPosted(false)
      setHasGrade(false)
      setGradeMode(hasRubric ? 'rubric' : 'points')
      setLoadError(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setLoadError(null)
    void (async () => {
      try {
        const grade = submissionId
          ? await fetchSubmissionGrade(courseCode, itemId, submissionId)
          : await fetchAssignmentStudentGrade(courseCode, itemId, studentUserId!)
        if (!cancelled) applyGrade(grade)
      } catch (e) {
        if (!cancelled) {
          setComment('')
          setThreadComments([])
          setPointsInput('')
          setRubricScores({})
          setPosted(false)
          setHasGrade(false)
          setGradeMode(hasRubric ? 'rubric' : 'points')
          setLoadError(e instanceof Error ? e.message : 'Could not load grade.')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [applyGrade, courseCode, gradeRefreshKey, agentApplyKey, hasRubric, itemId, studentUserId, submissionId])

  useEffect(() => {
    if (!autoFocusScore || !focusTarget) {
      pendingScoreFocusRef.current = null
      return
    }
    pendingScoreFocusRef.current = focusTarget
  }, [autoFocusScore, focusTarget])

  useEffect(() => {
    if (!autoFocusScore || !focusTarget || loading) return
    if (pendingScoreFocusRef.current !== focusTarget) return
    if (hasRubric && gradeMode === 'rubric' && Object.keys(rubricScores).length === 0) {
      setGradeMode('points')
      return
    }
    if (hasRubric && gradeMode === 'rubric') {
      pendingScoreFocusRef.current = null
      return
    }
    const frame = window.requestAnimationFrame(() => {
      const input = scoreInputRef.current
      if (!input || input.disabled) return
      input.focus()
      input.select()
      pendingScoreFocusRef.current = null
    })
    return () => window.cancelAnimationFrame(frame)
  }, [autoFocusScore, focusTarget, gradeMode, hasRubric, loading, rubricScores])

  useEffect(() => {
    if (!hasRubric) return
    const hasSavedRubric = Object.keys(rubricScores).length > 0
    const hasSavedPoints = pointsInput.trim() !== ''
    if (hasSavedRubric) {
      setGradeMode('rubric')
      return
    }
    if (!hasSavedPoints && !autoFocusScore) {
      setGradeMode('rubric')
    }
  }, [autoFocusScore, hasRubric, pointsInput, rubricScores])

  useEffect(() => {
    if (hasRubric && gradeMode === 'points' && !pointsInput && rubric) {
      const total = rubricTotal(rubric, rubricScores)
      if (total > 0) setPointsInput(formatPointsCell(total))
    }
  }, [gradeMode, hasRubric, pointsInput, rubric, rubricScores])

  const displayScore = useMemo(() => {
    if (hasRubric && gradeMode === 'rubric' && rubric) {
      return formatPointsCell(rubricTotal(rubric, rubricScores))
    }
    const trimmed = pointsInput.trim()
    if (trimmed) return trimmed
    return '—'
  }, [gradeMode, hasRubric, pointsInput, rubric, rubricScores])

  const canvasGradePayload = useMemo((): CanvasGradePushPayload | undefined => {
    const instructorComment = comment.trim() || null
    if (hasRubric && rubric && rubricScoresComplete(rubric, rubricScores)) {
      return {
        rubricScores,
        instructorComment,
      }
    }
    const trimmed = pointsInput.trim()
    if (trimmed !== '') {
      const n = Number.parseFloat(trimmed.replace(',', ''))
      if (Number.isFinite(n) && n >= 0) {
        return {
          pointsEarned: n,
          instructorComment,
        }
      }
    }
    if (hasGrade && instructorComment) {
      if (Object.keys(rubricScores).length > 0) {
        return { rubricScores, instructorComment }
      }
      return { instructorComment }
    }
    return undefined
  }, [comment, hasGrade, hasRubric, pointsInput, rubric, rubricScores])

  const handleSave = useCallback(async () => {
    if (!submissionId && !studentUserId) return
    setSaving(true)
    setSaveError(null)
    setSavedFlash(false)
    try {
      const saveGrade = submissionId
        ? (body: Parameters<typeof putSubmissionGrade>[3]) =>
            putSubmissionGrade(courseCode, itemId, submissionId, body)
        : (body: Parameters<typeof putAssignmentStudentGrade>[3]) =>
            putAssignmentStudentGrade(courseCode, itemId, studentUserId!, body)
      const trimmedComment = comment.trim()
      const commentOnlySave =
        hasGrade &&
        trimmedComment !== '' &&
        pointsInput.trim() === '' &&
        (!hasRubric || !rubric || Object.keys(rubricScores).length === 0)
      if (commentOnlySave) {
        const saved = await saveGrade({ instructorComment: trimmedComment })
        applyGrade(saved)
        const savedMsg = 'Comment saved' + (posted ? '' : ' (held until posted)')
        setFlashMsg(savedMsg)
        setSavedFlash(true)
        setPosted(true)
        pendingScoreFocusRef.current = null
        scoreInputRef.current?.blur()
        if (focusTargetRef.current === focusTarget) {
          onGradeSaved?.()
        }
        let startedCanvasSync = false
        if (submissionId && canvasLink && canvasGradePayload) {
          canvasSyncAbortRef.current?.()
          const syncHandle = queueCanvasGradeSync({
            courseCode,
            itemId,
            submissionId,
            canvasLink,
            gradePayload: canvasGradePayload,
            onComplete: () => {
              if (focusTargetRef.current !== focusTarget) return
              setCanvasSyncPending(false)
              setFlashMsg('Comment saved and synced to Canvas.')
              setSavedFlash(true)
              onGradeSaved?.()
              window.setTimeout(() => setSavedFlash(false), 2500)
            },
            onError: (message) => {
              if (focusTargetRef.current !== focusTarget) return
              setCanvasSyncPending(false)
              setFlashMsg(message)
              setSavedFlash(true)
              window.setTimeout(() => setSavedFlash(false), 4000)
            },
          })
          if (syncHandle) {
            canvasSyncAbortRef.current = syncHandle.abort
            setCanvasSyncPending(true)
            startedCanvasSync = true
          }
        }
        if (!startedCanvasSync) {
          window.setTimeout(() => setSavedFlash(false), 2500)
        }
        return
      }
      if (hasRubric && gradeMode === 'rubric' && rubric) {
        if (!rubricScoresComplete(rubric, rubricScores)) {
          setSaveError('Select a rating for every rubric criterion.')
          return
        }
        const saved = await saveGrade({
          rubricScores,
          instructorComment: trimmedComment || null,
        })
        applyGrade(saved)
      } else {
        const trimmed = pointsInput.trim()
        if (trimmed === '') {
          setSaveError('Enter a score.')
          return
        }
        const n = Number.parseFloat(trimmed.replace(',', ''))
        if (!Number.isFinite(n) || n < 0) {
          setSaveError('Enter a valid score.')
          return
        }
        if (maxPoints != null && n > maxPoints) {
          setSaveError(`Score cannot exceed ${maxPoints} points.`)
          return
        }
        const saved = await saveGrade({
          pointsEarned: n,
          instructorComment: trimmedComment || null,
        })
        applyGrade(saved)
      }
      const savedMsg = 'Grade saved' + (posted ? '' : ' (held until posted)')
      setFlashMsg(savedMsg)
      setSavedFlash(true)
      setPosted(true)
      setHasGrade(true)
      pendingScoreFocusRef.current = null
      scoreInputRef.current?.blur()
      if (focusTargetRef.current === focusTarget) {
        onGradeSaved?.()
      }
      let startedCanvasSync = false
      if (submissionId && canvasLink && canvasGradePayload) {
        canvasSyncAbortRef.current?.()
        const syncHandle = queueCanvasGradeSync({
          courseCode,
          itemId,
          submissionId,
          canvasLink,
          gradePayload: canvasGradePayload,
          onComplete: (grade) => {
            if (focusTargetRef.current !== focusTarget) return
            setCanvasSyncPending(false)
            if (grade.comments?.length) {
              applyGrade(grade)
            }
            setFlashMsg(
              trimmedComment
                ? 'Grade and comment saved and synced to Canvas.'
                : 'Grade saved and synced to Canvas.',
            )
            setSavedFlash(true)
            onGradeSaved?.()
            window.setTimeout(() => setSavedFlash(false), 2500)
          },
          onError: (message) => {
            if (focusTargetRef.current !== focusTarget) return
            setCanvasSyncPending(false)
            setFlashMsg(message)
            setSavedFlash(true)
            window.setTimeout(() => setSavedFlash(false), 4000)
          },
        })
        if (syncHandle) {
          canvasSyncAbortRef.current = syncHandle.abort
          setCanvasSyncPending(true)
          startedCanvasSync = true
        }
      }
      if (!startedCanvasSync) {
        window.setTimeout(() => setSavedFlash(false), 2500)
      }
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not save grade.')
    } finally {
      setSaving(false)
    }
  }, [
    applyGrade,
    canvasGradePayload,
    canvasLink,
    comment,
    hasGrade,
    courseCode,
    gradeMode,
    hasRubric,
    itemId,
    maxPoints,
    onGradeSaved,
    pointsInput,
    posted,
    rubric,
    rubricScores,
    focusTarget,
    studentUserId,
    submissionId,
  ])

  async function handleClearGrade() {
    if (!submissionId && !studentUserId) return
    setSaving(true)
    setSaveError(null)
    setSavedFlash(false)
    try {
      if (submissionId) {
        await putSubmissionGrade(courseCode, itemId, submissionId, {
          clearGrade: true,
        })
      } else {
        await putAssignmentStudentGrade(courseCode, itemId, studentUserId!, {
          clearGrade: true,
        })
      }
      setPointsInput('')
      setRubricScores({})
      setComment('')
      setThreadComments([])
      setPosted(false)
      setHasGrade(false)
      setFlashMsg('Grade cleared.')
      setSavedFlash(true)
      onGradeCleared?.()
      window.setTimeout(() => setSavedFlash(false), 2500)
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not clear grade.')
    } finally {
      setSaving(false)
    }
  }

  if (!submissionId && !studentUserId) {
    return (
      <div className="flex h-full items-center justify-center p-6 text-center text-sm text-slate-600 dark:text-neutral-400">
        Select a student to grade.
      </div>
    )
  }

  const formDisabled = disabled || saving || loading

  return (
    <section
      className="flex min-h-0 flex-1 flex-col"
      aria-label="Grade submission"
      onKeyDown={(e) => {
        if (e.key !== 'Enter' || (!e.altKey && !e.getModifierState('Alt'))) return
        if (formDisabled) return
        e.preventDefault()
        void handleSave()
      }}
    >
      <div className="flex-1 space-y-4 overflow-y-auto p-5">
        <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-600 dark:bg-neutral-900/60">
          <div className="flex items-end justify-between gap-3">
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Current score
              </p>
              <p className="mt-1 text-3xl font-semibold tabular-nums text-slate-900 dark:text-neutral-50">
                {displayScore}
                {maxPoints != null ? (
                  <span className="ms-1 text-lg font-normal text-slate-500 dark:text-neutral-400">
                    / {maxPoints}
                  </span>
                ) : null}
              </p>
            </div>
            <div className="flex flex-col items-end gap-2">
              {mode === 'staff' && graderAgentEnabled && submissionId ? (
                <button
                  type="button"
                  onClick={() => setAgentOpen(true)}
                  className="rounded-lg border border-indigo-300 px-2.5 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-50 dark:border-indigo-800 dark:text-indigo-300"
                >
                  {t('gradingAgent.button')}
                </button>
              ) : null}
              {posted ? (
                <span className="rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-semibold text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-200">
                  Posted
                </span>
              ) : (
                <span className="rounded-full bg-amber-100 px-2.5 py-1 text-xs font-semibold text-amber-900 dark:bg-amber-950/60 dark:text-amber-200">
                  Draft
                </span>
              )}
            </div>
          </div>
        </div>

        {mode === 'student' && posted && gradedByAi ? (
          <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-neutral-600 dark:bg-neutral-900/40">
            <p className="text-slate-700 dark:text-neutral-200">{t('gradingAgent.student.disclosure')}</p>
            <button
              type="button"
              className="mt-2 text-sm font-semibold text-indigo-700 hover:underline dark:text-indigo-300"
              onClick={() => void postGraderAgentRegradeRequest(courseCode, itemId)}
            >
              {t('gradingAgent.student.regradeRequest')}
            </button>
          </div>
        ) : null}

        {loading ? (
          <p className="text-sm text-slate-500 dark:text-neutral-400" role="status">
            Loading grade…
          </p>
        ) : null}
        {loadError ? (
          <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200" role="alert">
            {loadError}
          </p>
        ) : null}

        {hasRubric && rubric ? (
          <>
            <div
              className="inline-flex w-full rounded-xl border border-slate-200 bg-slate-100 p-1 dark:border-neutral-600 dark:bg-neutral-900"
              role="tablist"
              aria-label="Grading method"
            >
              <button
                type="button"
                role="tab"
                aria-selected={gradeMode === 'rubric'}
                disabled={formDisabled}
                onClick={() => setGradeMode('rubric')}
                className={`flex-1 rounded-lg px-3 py-2 text-sm font-semibold transition disabled:opacity-50 ${
                  gradeMode === 'rubric'
                    ? 'bg-white text-indigo-700 shadow-sm dark:bg-neutral-800 dark:text-indigo-300'
                    : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100'
                }`}
              >
                Rubric
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={gradeMode === 'points'}
                disabled={formDisabled}
                onClick={() => setGradeMode('points')}
                className={`flex-1 rounded-lg px-3 py-2 text-sm font-semibold transition disabled:opacity-50 ${
                  gradeMode === 'points'
                    ? 'bg-white text-indigo-700 shadow-sm dark:bg-neutral-800 dark:text-indigo-300'
                    : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100'
                }`}
              >
                Points
              </button>
            </div>

            {gradeMode === 'rubric' ? (
              <RubricGradePicker
                rubric={rubric}
                scores={rubricScores}
                onScoresChange={setRubricScores}
                disabled={formDisabled}
                compact
              />
            ) : (
              <label className="block text-sm text-slate-700 dark:text-neutral-200">
                <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
                  Override score{maxPoints != null ? ` (out of ${maxPoints})` : ''}
                </span>
                <input
                  ref={scoreInputRef}
                  type="number"
                  data-speed-grader-score="true"
                  min={0}
                  max={maxPoints ?? undefined}
                  step="any"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm tabular-nums dark:border-neutral-600 dark:bg-neutral-950"
                  value={pointsInput}
                  onChange={(e) => setPointsInput(e.target.value)}
                  disabled={formDisabled}
                />
                <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
                  Use this when you want to enter a score without following the rubric.
                </p>
              </label>
            )}
          </>
        ) : (
          <label className="block text-sm text-slate-700 dark:text-neutral-200">
            <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
              Score{maxPoints != null ? ` (out of ${maxPoints})` : ''}
            </span>
            <input
              ref={scoreInputRef}
              type="number"
              data-speed-grader-score="true"
              min={0}
              max={maxPoints ?? undefined}
              step="any"
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm tabular-nums dark:border-neutral-600 dark:bg-neutral-950"
              value={pointsInput}
              onChange={(e) => setPointsInput(e.target.value)}
              disabled={formDisabled}
            />
          </label>
        )}

        <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
          <div>
            <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
              Feedback conversation
            </span>
            <SubmissionCommentThread
              comments={threadComments}
              roster={rosterByUserId}
              emptyLabel={
                mode === 'student'
                  ? 'No feedback has been posted yet.'
                  : 'No feedback yet. Add a comment below when you save the grade.'
              }
            />
          </div>
          {mode === 'staff' ? (
            <label className="block">
              <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
                Add comment
              </span>
              <textarea
                className="min-h-24 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm leading-relaxed dark:border-neutral-600 dark:bg-neutral-950"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                disabled={formDisabled}
                placeholder="Write feedback to add to the conversation…"
                rows={4}
              />
              <p className="mt-1.5 text-xs text-slate-500 dark:text-neutral-400">
                Your comment is added to the thread when you save the grade.
              </p>
            </label>
          ) : null}
        </div>
      </div>

      <div className="shrink-0 space-y-2 border-t border-slate-200 bg-slate-50 p-4 dark:border-neutral-600 dark:bg-neutral-900/80">
        {saveError ? (
          <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
            {saveError}
          </p>
        ) : null}
        {savedFlash || canvasSyncPending ? (
          <p
            className={`text-sm font-medium ${canvasSyncPending ? 'text-sky-700 dark:text-sky-300' : 'text-emerald-700 dark:text-emerald-300'}`}
            role="status"
          >
            {canvasSyncPending ? 'Syncing grade to Canvas…' : flashMsg}
          </p>
        ) : null}
        <div className="flex gap-2">
          {hasGrade && (
            <button
              type="button"
              disabled={formDisabled}
              onClick={() => void handleClearGrade()}
              className="flex-1 rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm font-semibold text-rose-600 hover:bg-rose-50 dark:border-neutral-700 dark:bg-neutral-950 dark:text-rose-400 dark:hover:bg-rose-950/30 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Mark ungraded
            </button>
          )}
          <button
            type="button"
            disabled={formDisabled}
            onClick={() => void handleSave()}
            title={`Save grade (${altKeyHint()}+Enter)`}
            className={`${hasGrade ? 'flex-1' : 'w-full'} rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50`}
          >
            {saving ? 'Saving…' : 'Save grade'}
          </button>
        </div>
      </div>
      <GraderAgentDrawer
        open={agentOpen}
        onClose={() => setAgentOpen(false)}
        courseCode={courseCode}
        itemId={itemId}
        submissionId={submissionId}
        rubric={rubric}
        maxPoints={maxPoints}
        onApplied={() => {
          onGradeSaved?.()
          setAgentApplyKey((k) => k + 1)
        }}
      />
    </section>
  )
}
