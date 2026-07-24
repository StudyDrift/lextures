import { useEffect, useId, useMemo, useState } from 'react'
import { Loader2, Trash2, X } from 'lucide-react'
import {
  addCourseOutcomeLink,
  type CourseOutcome,
  type QuizOutcomeLinkSuggestion,
} from '../../lib/courses-api'
import {
  OUTCOME_INTENSITY_LABELS,
  OUTCOME_MEASUREMENT_LABELS,
} from './outcome-links-helpers'

type DraftRow = QuizOutcomeLinkSuggestion & { key: string }

type SuggestQuizOutcomeLinksModalProps = {
  open: boolean
  courseCode: string
  itemId: string
  suggestions: QuizOutcomeLinkSuggestion[]
  outcomes: CourseOutcome[]
  questions: { id: string; prompt: string }[]
  /** Question ids that exist on the saved quiz (can be linked immediately). */
  savedQuestionIds: Set<string>
  onClose: () => void
  onApplied: () => void | Promise<void>
}

function toRows(suggestions: QuizOutcomeLinkSuggestion[]): DraftRow[] {
  return suggestions.map((s, i) => ({
    key: `sug-${i}-${s.targetKind}-${s.quizQuestionId}-${s.outcomeId}`,
    ...s,
  }))
}

export function SuggestQuizOutcomeLinksModal({
  open,
  courseCode,
  itemId,
  suggestions,
  outcomes,
  questions,
  savedQuestionIds,
  onClose,
  onApplied,
}: SuggestQuizOutcomeLinksModalProps) {
  const titleId = useId()
  const [rows, setRows] = useState<DraftRow[]>(() => toRows(suggestions))
  const [applying, setApplying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setRows(toRows(suggestions))
    setError(null)
    setApplying(false)
  }, [open, suggestions])

  const outcomeTitleById = useMemo(() => {
    const m = new Map<string, string>()
    for (const o of outcomes) m.set(o.id, o.title)
    return m
  }, [outcomes])

  const questionLabelById = useMemo(() => {
    const m = new Map<string, string>()
    questions.forEach((q, index) => {
      const prompt = (q.prompt || q.id).replace(/\s+/g, ' ')
      const snippet = prompt.length > 64 ? `${prompt.slice(0, 64)}…` : prompt
      m.set(q.id, `Q${index + 1}: ${snippet}`)
    })
    return m
  }, [questions])

  if (!open) return null

  const applicable = rows.filter((r) => {
    if (r.targetKind === 'quiz') return true
    return savedQuestionIds.has(r.quizQuestionId)
  })
  const blockedUnsaved = rows.length - applicable.length

  function removeRow(key: string) {
    setRows((prev) => prev.filter((r) => r.key !== key))
  }

  async function onApply() {
    if (applying || applicable.length === 0) return
    setApplying(true)
    setError(null)
    let created = 0
    try {
      for (const row of applicable) {
        await addCourseOutcomeLink(courseCode, row.outcomeId, {
          structureItemId: itemId,
          targetKind: row.targetKind,
          quizQuestionId: row.targetKind === 'quiz_question' ? row.quizQuestionId : undefined,
          measurementLevel: row.measurementLevel,
          intensityLevel: row.intensityLevel,
        })
        created += 1
      }
      await onApplied()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not apply mappings.'
      if (created > 0) {
        setError(`Applied ${created} of ${applicable.length} mappings, then failed: ${msg}`)
        await onApplied()
      } else {
        setError(msg)
      }
    } finally {
      setApplying(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-[60] flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget && !applying) onClose()
      }}
    >
      <div className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
          <div className="min-w-0">
            <h3
              id={titleId}
              className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
            >
              Review suggested outcome links
            </h3>
            <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
              Remove any you do not want, then apply the rest to this quiz.
            </p>
          </div>
          <button
            type="button"
            onClick={() => {
              if (!applying) onClose()
            }}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
            aria-label="Close"
            disabled={applying}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
          {rows.length === 0 ? (
            <p className="rounded-xl border border-dashed border-slate-200 px-4 py-6 text-center text-sm text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
              No suggestions to apply. Close and try again, or map outcomes manually.
            </p>
          ) : (
            rows.map((row) => {
              const outcomeTitle = outcomeTitleById.get(row.outcomeId) ?? row.outcomeId
              const targetLabel =
                row.targetKind === 'quiz'
                  ? 'Whole quiz'
                  : (questionLabelById.get(row.quizQuestionId) ?? `Question ${row.quizQuestionId}`)
              const unsaved =
                row.targetKind === 'quiz_question' && !savedQuestionIds.has(row.quizQuestionId)
              return (
                <div
                  key={row.key}
                  className="rounded-xl border border-slate-200 bg-slate-50/60 p-3 dark:border-neutral-700 dark:bg-neutral-950/40"
                >
                  <div className="mb-2 flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                        {outcomeTitle}
                      </p>
                      <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                        {targetLabel}
                        {' · '}
                        {OUTCOME_MEASUREMENT_LABELS[row.measurementLevel] ?? row.measurementLevel}
                        {' · '}
                        {OUTCOME_INTENSITY_LABELS[row.intensityLevel] ?? row.intensityLevel}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeRow(row.key)}
                      disabled={applying}
                      className="inline-flex shrink-0 items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:text-rose-300 dark:hover:bg-rose-950/40"
                    >
                      <Trash2 className="h-3.5 w-3.5" aria-hidden />
                      Remove
                    </button>
                  </div>
                  {row.rationale ? (
                    <p className="text-xs leading-relaxed text-slate-600 dark:text-neutral-300">
                      {row.rationale}
                    </p>
                  ) : null}
                  {unsaved ? (
                    <p className="mt-2 text-xs text-amber-800 dark:text-amber-200/90">
                      Save questions first — this suggestion will be skipped until the question is
                      saved.
                    </p>
                  ) : null}
                </div>
              )
            })
          )}
        </div>

        {blockedUnsaved > 0 ? (
          <p className="mx-4 mb-2 text-xs text-amber-800 dark:text-amber-200/90">
            {blockedUnsaved} suggestion{blockedUnsaved === 1 ? '' : 's'} need saved questions and
            will be skipped.
          </p>
        ) : null}

        {error ? (
          <p className="mx-4 mb-2 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
            {error}
          </p>
        ) : null}

        <div className="flex flex-wrap items-center justify-end gap-2 border-t border-slate-200 px-4 py-3 dark:border-neutral-600">
          <button
            type="button"
            onClick={onClose}
            disabled={applying}
            className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void onApply()}
            disabled={applying || applicable.length === 0}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
          >
            {applying ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                Applying…
              </>
            ) : (
              `Apply ${applicable.length} mapping${applicable.length === 1 ? '' : 's'}`
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
