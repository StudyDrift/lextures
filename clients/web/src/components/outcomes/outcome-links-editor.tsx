import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2, Plus, X } from 'lucide-react'
import { usePermissions } from '../../context/use-permissions'
import {
  addCourseOutcomeLink,
  courseItemCreatePermission,
  deleteCourseOutcomeLink,
  fetchCourseOutcomes,
  OUTCOME_INTENSITY_LEVEL_IDS,
  OUTCOME_MEASUREMENT_LEVEL_IDS,
  type CourseOutcome,
} from '../../lib/courses-api'
import {
  filterOutcomeLinksForTarget,
  formatOutcomeLinkLevels,
  OUTCOME_INTENSITY_LABELS,
  OUTCOME_MEASUREMENT_LABELS,
  type OutcomeLinkTargetKind,
} from './outcome-links-helpers'

const settingsInputClass =
  'w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500 dark:focus:ring-indigo-500'

const compactInputClass =
  'w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500'

type OutcomeLinksEditorProps = {
  courseCode: string
  itemId: string
  targetKind: OutcomeLinkTargetKind
  quizQuestionId?: string
  disabled?: boolean
  /** settings = dense side panel; compact = question cards / modal */
  variant?: 'settings' | 'compact'
  /** When provided, parent owns the fetch (e.g. Edit Questions modal). */
  outcomes?: CourseOutcome[]
  outcomesLoading?: boolean
  outcomesError?: string | null
  onOutcomesChange?: () => void | Promise<void>
  /** Disable add + show hint (unsaved question ids). */
  linkDisabledReason?: string | null
  emptyLabel?: string
  hideHeaderHint?: boolean
}

export function OutcomeLinksEditor({
  courseCode,
  itemId,
  targetKind,
  quizQuestionId,
  disabled,
  variant = 'settings',
  outcomes: outcomesProp,
  outcomesLoading,
  outcomesError,
  onOutcomesChange,
  linkDisabledReason,
  emptyLabel = 'No outcome links yet.',
  hideHeaderHint,
}: OutcomeLinksEditorProps) {
  const formId = useId()
  const { allows, loading: permLoading } = usePermissions()
  const canMap = !permLoading && allows(courseItemCreatePermission(courseCode))
  const controlled = outcomesProp !== undefined

  const [internalOutcomes, setInternalOutcomes] = useState<CourseOutcome[]>([])
  const [internalLoading, setInternalLoading] = useState(!controlled)
  const [internalError, setInternalError] = useState<string | null>(null)

  const [outcomeId, setOutcomeId] = useState('')
  const [measurementLevel, setMeasurementLevel] = useState('formative')
  const [intensityLevel, setIntensityLevel] = useState('medium')
  const [adding, setAdding] = useState(false)
  const [removingId, setRemovingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [formOpen, setFormOpen] = useState(variant === 'settings')

  const loadInternal = useCallback(async () => {
    if (controlled) return
    setInternalLoading(true)
    setInternalError(null)
    try {
      const data = await fetchCourseOutcomes(courseCode)
      setInternalOutcomes(data.outcomes)
    } catch (e) {
      setInternalError(e instanceof Error ? e.message : 'Could not load outcomes.')
    } finally {
      setInternalLoading(false)
    }
  }, [controlled, courseCode])

  useEffect(() => {
    if (!controlled) void loadInternal()
  }, [controlled, loadInternal])

  const outcomes = controlled ? outcomesProp : internalOutcomes
  const loading = controlled ? Boolean(outcomesLoading) : internalLoading
  const loadError = controlled ? (outcomesError ?? null) : internalError

  const mappedRows = useMemo(
    () => filterOutcomeLinksForTarget(outcomes, itemId, targetKind, quizQuestionId),
    [outcomes, itemId, targetKind, quizQuestionId],
  )

  async function refresh() {
    if (controlled) {
      await onOutcomesChange?.()
    } else {
      await loadInternal()
    }
  }

  async function onAdd(e: React.FormEvent) {
    e.preventDefault()
    setActionError(null)
    if (!outcomeId) {
      setActionError('Choose an outcome.')
      return
    }
    if (targetKind === 'quiz_question' && !quizQuestionId?.trim()) {
      setActionError('Save the question before linking outcomes.')
      return
    }
    setAdding(true)
    try {
      await addCourseOutcomeLink(courseCode, outcomeId, {
        structureItemId: itemId,
        targetKind,
        quizQuestionId: targetKind === 'quiz_question' ? quizQuestionId?.trim() : undefined,
        measurementLevel,
        intensityLevel,
      })
      setOutcomeId('')
      setMeasurementLevel('formative')
      setIntensityLevel('medium')
      if (variant === 'compact') setFormOpen(false)
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not add mapping.')
    } finally {
      setAdding(false)
    }
  }

  async function onRemove(outcomeOid: string, linkId: string) {
    setActionError(null)
    setRemovingId(linkId)
    try {
      await deleteCourseOutcomeLink(courseCode, outcomeOid, linkId)
      await refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not remove mapping.')
    } finally {
      setRemovingId(null)
    }
  }

  const settingsOutcomesUrl = `/courses/${encodeURIComponent(courseCode)}/settings/outcomes`
  const inputClass = variant === 'compact' ? compactInputClass : settingsInputClass
  const textSize = variant === 'compact' ? 'text-xs' : 'text-[11px]'
  const linkBlocked = Boolean(linkDisabledReason) || disabled

  return (
    <div className={variant === 'compact' ? 'space-y-2.5' : 'space-y-3'}>
      {!hideHeaderHint && variant === 'settings' ? (
        <p className="text-[11px] leading-snug text-slate-400 dark:text-neutral-500">
          Link with measurement and intensity.{' '}
          <Link
            to={settingsOutcomesUrl}
            className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
          >
            Open full outcomes page
          </Link>
          .
        </p>
      ) : null}

      {!canMap ? (
        <p className={`${textSize} text-slate-500 dark:text-neutral-500`}>
          You need course edit permission to change outcome mappings.
        </p>
      ) : null}

      {loadError ? (
        <p className={`${textSize} text-rose-600 dark:text-rose-400`}>{loadError}</p>
      ) : null}
      {actionError ? (
        <p className={`${textSize} text-rose-600 dark:text-rose-400`}>{actionError}</p>
      ) : null}

      {loading ? (
        <p className={`flex items-center gap-1.5 ${textSize} text-slate-500 dark:text-neutral-500`}>
          <Loader2 className="h-3.5 w-3.5 motion-safe:animate-spin" aria-hidden />
          Loading…
        </p>
      ) : null}

      {!loading && canMap ? (
        <>
          {mappedRows.length === 0 ? (
            <p className={`${textSize} text-slate-500 dark:text-neutral-500`}>{emptyLabel}</p>
          ) : (
            <ul className="flex flex-wrap gap-1.5">
              {mappedRows.map(({ outcome, link }) => (
                <li key={link.id}>
                  <span className="inline-flex max-w-full items-center gap-1 rounded-full border border-slate-200/90 bg-slate-50 py-1 pe-1 ps-2.5 text-slate-700 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200">
                    <span className={`min-w-0 truncate font-medium ${textSize}`}>{outcome.title}</span>
                    <span className={`hidden shrink-0 text-slate-400 sm:inline ${textSize} dark:text-neutral-500`}>
                      {formatOutcomeLinkLevels(link)}
                    </span>
                    <button
                      type="button"
                      disabled={disabled || removingId === link.id}
                      onClick={() => void onRemove(outcome.id, link.id)}
                      className="shrink-0 rounded-full p-1 text-slate-400 motion-safe:transition-colors hover:bg-rose-50 hover:text-rose-700 disabled:opacity-50 dark:hover:bg-rose-950/40 dark:hover:text-rose-300"
                      aria-label={`Remove ${outcome.title}`}
                    >
                      {removingId === link.id ? (
                        <Loader2 className="h-3 w-3 motion-safe:animate-spin" aria-hidden />
                      ) : (
                        <X className="h-3 w-3" aria-hidden />
                      )}
                    </button>
                  </span>
                </li>
              ))}
            </ul>
          )}

          {linkDisabledReason ? (
            <p className={`${textSize} text-amber-800 dark:text-amber-200/90`}>{linkDisabledReason}</p>
          ) : null}

          {variant === 'compact' && !formOpen ? (
            <button
              type="button"
              disabled={linkBlocked || outcomes.length === 0}
              onClick={() => setFormOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 shadow-sm motion-safe:transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden />
              Add outcome
            </button>
          ) : null}

          {(variant === 'settings' || formOpen) && !linkDisabledReason ? (
            <form
              onSubmit={onAdd}
              className={
                variant === 'compact'
                  ? 'space-y-2 rounded-xl border border-slate-200/90 bg-slate-50/70 p-3 dark:border-neutral-700 dark:bg-neutral-950/50'
                  : 'space-y-2 border-t border-slate-100 pt-2 dark:border-neutral-800/80'
              }
            >
              <div>
                <label
                  htmlFor={`${formId}-outcome`}
                  className={`mb-0.5 block font-medium text-slate-500 dark:text-neutral-400 ${textSize}`}
                >
                  Outcome
                </label>
                <select
                  id={`${formId}-outcome`}
                  value={outcomeId}
                  onChange={(e) => setOutcomeId(e.target.value)}
                  disabled={linkBlocked || adding}
                  className={inputClass}
                >
                  <option value="">Select outcome…</option>
                  {outcomes.map((o) => (
                    <option key={o.id} value={o.id}>
                      {o.title}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label
                    htmlFor={`${formId}-measurement`}
                    className={`mb-0.5 block font-medium text-slate-500 dark:text-neutral-400 ${textSize}`}
                  >
                    Measurement
                  </label>
                  <select
                    id={`${formId}-measurement`}
                    value={measurementLevel}
                    onChange={(e) => setMeasurementLevel(e.target.value)}
                    disabled={linkBlocked || adding}
                    className={inputClass}
                  >
                    {OUTCOME_MEASUREMENT_LEVEL_IDS.map((id) => (
                      <option key={id} value={id}>
                        {OUTCOME_MEASUREMENT_LABELS[id] ?? id}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label
                    htmlFor={`${formId}-intensity`}
                    className={`mb-0.5 block font-medium text-slate-500 dark:text-neutral-400 ${textSize}`}
                  >
                    Intensity
                  </label>
                  <select
                    id={`${formId}-intensity`}
                    value={intensityLevel}
                    onChange={(e) => setIntensityLevel(e.target.value)}
                    disabled={linkBlocked || adding}
                    className={inputClass}
                  >
                    {OUTCOME_INTENSITY_LEVEL_IDS.map((id) => (
                      <option key={id} value={id}>
                        {OUTCOME_INTENSITY_LABELS[id] ?? id}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="submit"
                  disabled={linkBlocked || adding || !outcomeId || outcomes.length === 0}
                  className={
                    variant === 'compact'
                      ? 'rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50'
                      : 'w-full rounded-lg bg-indigo-600 px-2 py-1.5 text-[12px] font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50'
                  }
                >
                  {adding ? 'Adding…' : 'Add mapping'}
                </button>
                {variant === 'compact' ? (
                  <button
                    type="button"
                    disabled={adding}
                    onClick={() => {
                      setFormOpen(false)
                      setActionError(null)
                    }}
                    className="rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
                  >
                    Cancel
                  </button>
                ) : null}
              </div>

              {outcomes.length === 0 ? (
                <p className={`${textSize} text-slate-500 dark:text-neutral-500`}>
                  Create outcomes under Course Settings → Outcomes first.{' '}
                  <Link
                    to={settingsOutcomesUrl}
                    className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
                  >
                    Open outcomes
                  </Link>
                </p>
              ) : null}
            </form>
          ) : null}
        </>
      ) : null}
    </div>
  )
}
