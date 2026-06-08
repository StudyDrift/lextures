import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  fetchModuleAssignment,
  fetchSubmissionGrade,
  putSubmissionGrade,
  type RubricDefinition,
  type SubmissionGradeApi,
} from '../../lib/courses-api'
import { RubricGradePicker } from '../grading/rubric-grade-picker'
import { formatPointsCell, rubricScoresComplete, rubricTotal } from '../../lib/rubric-utils'

type GradeMode = 'rubric' | 'points'

type SubmissionGradingPanelProps = {
  courseCode: string
  itemId: string
  submissionId: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  disabled?: boolean
}

function initialGradeMode(grade: SubmissionGradeApi, hasRubric: boolean): GradeMode {
  if (!hasRubric) return 'points'
  if (grade.rubricScores && Object.keys(grade.rubricScores).length > 0) return 'rubric'
  if (grade.pointsEarned != null && Number.isFinite(grade.pointsEarned)) return 'points'
  return 'rubric'
}

export function SubmissionGradingPanel({
  courseCode,
  itemId,
  submissionId,
  rubric: rubricProp,
  maxPoints,
  disabled = false,
}: SubmissionGradingPanelProps) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedFlash, setSavedFlash] = useState(false)
  const [comment, setComment] = useState('')
  const [pointsInput, setPointsInput] = useState('')
  const [rubricScores, setRubricScores] = useState<Record<string, number>>({})
  const [posted, setPosted] = useState(false)
  const [gradeMode, setGradeMode] = useState<GradeMode>('points')
  const [fetchedRubric, setFetchedRubric] = useState<RubricDefinition | null>(null)

  const rubric = rubricProp ?? fetchedRubric
  const hasRubric = Boolean(rubric && rubric.criteria.length > 0)

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
    (grade: SubmissionGradeApi) => {
      setComment(grade.instructorComment ?? '')
      setPosted(Boolean(grade.posted))
      setGradeMode(initialGradeMode(grade, hasRubric))
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
    if (!submissionId) {
      setComment('')
      setPointsInput('')
      setRubricScores({})
      setPosted(false)
      setGradeMode(hasRubric ? 'rubric' : 'points')
      setLoadError(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setLoadError(null)
    void (async () => {
      try {
        const grade = await fetchSubmissionGrade(courseCode, itemId, submissionId)
        if (!cancelled) applyGrade(grade)
      } catch (e) {
        if (!cancelled) {
          setComment('')
          setPointsInput('')
          setRubricScores({})
          setPosted(false)
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
  }, [applyGrade, courseCode, itemId, submissionId])

  useEffect(() => {
    if (!hasRubric) return
    const hasSavedRubric = Object.keys(rubricScores).length > 0
    const hasSavedPoints = pointsInput.trim() !== ''
    if (hasSavedRubric || !hasSavedPoints) {
      setGradeMode('rubric')
    }
  }, [hasRubric, pointsInput, rubricScores])

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

  async function handleSave() {
    if (!submissionId) return
    setSaving(true)
    setSaveError(null)
    setSavedFlash(false)
    try {
      if (hasRubric && gradeMode === 'rubric' && rubric) {
        if (!rubricScoresComplete(rubric, rubricScores)) {
          setSaveError('Select a rating for every rubric criterion.')
          return
        }
        await putSubmissionGrade(courseCode, itemId, submissionId, {
          rubricScores,
          instructorComment: comment.trim() || null,
        })
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
        await putSubmissionGrade(courseCode, itemId, submissionId, {
          pointsEarned: n,
          instructorComment: comment.trim() || null,
        })
      }
      setSavedFlash(true)
      setPosted(true)
      window.setTimeout(() => setSavedFlash(false), 2500)
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Could not save grade.')
    } finally {
      setSaving(false)
    }
  }

  if (!submissionId) {
    return (
      <div className="flex h-full items-center justify-center p-6 text-center text-sm text-slate-600 dark:text-neutral-400">
        Select a submission to grade.
      </div>
    )
  }

  const formDisabled = disabled || saving || loading

  return (
    <section className="flex min-h-0 flex-1 flex-col" aria-label="Grade submission">
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
                  type="number"
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
              type="number"
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

        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block text-xs font-medium text-slate-500 dark:text-neutral-400">
            Feedback comment
          </span>
          <textarea
            className="min-h-28 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm leading-relaxed dark:border-neutral-600 dark:bg-neutral-950"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            disabled={formDisabled}
            placeholder="Share feedback the student will see with their grade…"
            rows={5}
          />
        </label>
      </div>

      <div className="shrink-0 space-y-2 border-t border-slate-200 bg-slate-50 p-4 dark:border-neutral-600 dark:bg-neutral-900/80">
        {saveError ? (
          <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
            {saveError}
          </p>
        ) : null}
        {savedFlash ? (
          <p className="text-sm font-medium text-emerald-700 dark:text-emerald-300" role="status">
            Grade saved{posted ? '' : ' (held until posted)'}.
          </p>
        ) : null}
        <button
          type="button"
          disabled={formDisabled}
          onClick={() => void handleSave()}
          className="w-full rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save grade'}
        </button>
      </div>
    </section>
  )
}
