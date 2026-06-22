import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import {
  deleteGradeCurve,
  postAssignmentCurve,
  postAssignmentCurvePreview,
  type GradeCurveMethod,
  type GradeCurvePreviewResponse,
} from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type ColRef = { id: string; title: string; kind: string; maxPoints?: number | null }

const METHOD_LABELS: Record<GradeCurveMethod, string> = {
  flat_bonus: 'Flat bonus',
  linear_scale: 'Linear scale',
  sqrt_curve: 'Square-root curve',
  set_minimum: 'Set minimum',
  custom_mapping: 'Custom mapping',
}

export function GradeCurveModal(props: {
  open: boolean
  courseCode: string
  column: ColRef | null
  activeCurveId?: string | null
  studentsById: Record<string, { name: string }>
  onClose: () => void
  onApplied: () => void
}) {
  const { open, courseCode, column, activeCurveId, studentsById, onClose, onApplied } = props
  const baseId = useId()
  const [method, setMethod] = useState<GradeCurveMethod>('linear_scale')
  const [bonus, setBonus] = useState('5')
  const [targetMean, setTargetMean] = useState('75')
  const [targetMax, setTargetMax] = useState('')
  const [minimum, setMinimum] = useState('50')
  const [scaleTarget, setScaleTarget] = useState<'mean' | 'max'>('mean')
  const [allowAboveMax, setAllowAboveMax] = useState(false)
  const [preview, setPreview] = useState<GradeCurvePreviewResponse | null>(null)
  const [loadingPreview, setLoadingPreview] = useState(false)
  const [applying, setApplying] = useState(false)
  const [reverting, setReverting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const itemKind = column?.kind === 'quiz' ? 'quiz' : 'assignment'

  const params = useMemo(() => {
    switch (method) {
      case 'flat_bonus':
        return { bonus: Number.parseFloat(bonus) || 0 }
      case 'linear_scale':
        if (scaleTarget === 'max') {
          return { targetMax: Number.parseFloat(targetMax) || 0 }
        }
        return { targetMean: Number.parseFloat(targetMean) || 0 }
      case 'set_minimum':
        return { minimum: Number.parseFloat(minimum) || 0 }
      default:
        return {}
    }
  }, [method, bonus, targetMean, targetMax, minimum, scaleTarget])

  const requestBody = useMemo(
    () => ({ method, params, allowAboveMax }),
    [method, params, allowAboveMax],
  )

  const runPreview = useCallback(async () => {
    if (!column) return
    setLoadingPreview(true)
    setFormError(null)
    try {
      const p = await postAssignmentCurvePreview(courseCode, column.id, requestBody, itemKind)
      setPreview(p)
    } catch (e: unknown) {
      setFormError(e instanceof Error ? e.message : 'Preview failed.')
      setPreview(null)
    } finally {
      setLoadingPreview(false)
    }
  }, [column, courseCode, itemKind, requestBody])

  useEffect(() => {
    if (!open || !column) {
      setPreview(null)
      setFormError(null)
      return
    }
    void runPreview()
  }, [open, column, runPreview])

  const handleApply = useCallback(async () => {
    if (!column || !preview) return
    if (preview.preview.eligibleCount === 0) {
      setFormError('No graded, non-excused submissions to curve.')
      return
    }
    setApplying(true)
    setFormError(null)
    try {
      await postAssignmentCurve(courseCode, column.id, requestBody, itemKind)
      toastSaveOk('Curve applied.')
      onApplied()
      onClose()
    } catch (e: unknown) {
      toastMutationError(e, 'Could not apply curve.')
    } finally {
      setApplying(false)
    }
  }, [column, courseCode, itemKind, onApplied, onClose, preview, requestBody])

  const handleRevert = useCallback(async () => {
    if (!activeCurveId) return
    setReverting(true)
    setFormError(null)
    try {
      await deleteGradeCurve(activeCurveId)
      toastSaveOk('Curve reverted.')
      onApplied()
      onClose()
    } catch (e: unknown) {
      toastMutationError(e, 'Could not revert curve.')
    } finally {
      setReverting(false)
    }
  }, [activeCurveId, onApplied, onClose])

  if (!open || !column) return null

  const changedRows = preview?.preview.results.filter((r) => r.changed) ?? []

  return (
    <div
      className="fixed inset-0 z-[90] flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="presentation"
    >
      <button type="button" className="absolute inset-0 cursor-default" aria-label="Close" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={`${baseId}-title`}
        className="relative flex max-h-[min(90vh,720px)] w-full max-w-2xl flex-col overflow-hidden rounded-xl bg-white shadow-xl dark:bg-neutral-900"
      >
        <div className="flex items-start justify-between gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div>
            <h2 id={`${baseId}-title`} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Curve grades — {column.title}
            </h2>
            <p className="mt-0.5 text-sm text-slate-600 dark:text-neutral-400">
              Preview changes before applying. Raw scores are preserved and curves can be undone.
            </p>
          </div>
          <button
            type="button"
            className="rounded-md px-2 py-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            onClick={onClose}
            aria-label="Close dialog"
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-4">
          {activeCurveId ? (
            <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
              A curve is already applied to this column. Applying again replaces it; or undo the current curve first.
            </div>
          ) : null}

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Method</span>
              <select
                className="w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                value={method}
                onChange={(e) => setMethod(e.target.value as GradeCurveMethod)}
              >
                {(Object.keys(METHOD_LABELS) as GradeCurveMethod[]).map((m) => (
                  <option key={m} value={m}>
                    {METHOD_LABELS[m]}
                  </option>
                ))}
              </select>
            </label>

            {method === 'flat_bonus' ? (
              <label className="block text-sm">
                <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Bonus points</span>
                <input
                  type="number"
                  className="w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                  value={bonus}
                  onChange={(e) => setBonus(e.target.value)}
                />
              </label>
            ) : null}

            {method === 'linear_scale' ? (
              <>
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Scale to</span>
                  <select
                    className="w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    value={scaleTarget}
                    onChange={(e) => setScaleTarget(e.target.value as 'mean' | 'max')}
                  >
                    <option value="mean">Target mean</option>
                    <option value="max">Target max</option>
                  </select>
                </label>
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
                    {scaleTarget === 'mean' ? 'Target mean' : 'Target max score'}
                  </span>
                  <input
                    type="number"
                    className="w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    value={scaleTarget === 'mean' ? targetMean : targetMax}
                    onChange={(e) =>
                      scaleTarget === 'mean' ? setTargetMean(e.target.value) : setTargetMax(e.target.value)
                    }
                  />
                </label>
              </>
            ) : null}

            {method === 'set_minimum' ? (
              <label className="block text-sm">
                <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Minimum score</span>
                <input
                  type="number"
                  className="w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                  value={minimum}
                  onChange={(e) => setMinimum(e.target.value)}
                />
              </label>
            ) : null}

            <label className="flex items-center gap-2 text-sm sm:col-span-2">
              <input
                type="checkbox"
                checked={allowAboveMax}
                onChange={(e) => setAllowAboveMax(e.target.checked)}
              />
              <span>Allow scores above max (extra credit)</span>
            </label>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
              onClick={() => void runPreview()}
              disabled={loadingPreview}
            >
              {loadingPreview ? 'Refreshing…' : 'Refresh preview'}
            </button>
          </div>

          {formError ? <p className="mt-3 text-sm text-red-600 dark:text-red-400">{formError}</p> : null}

          {preview ? (
            <div className="mt-4 space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <StatCard label="Mean before" value={preview.preview.meanBefore} />
                <StatCard label="Mean after" value={preview.preview.meanAfter} />
                <StatCard label="Median before" value={preview.preview.medianBefore} />
                <StatCard label="Median after" value={preview.preview.medianAfter} />
              </div>

              <div>
                <h3 className="mb-2 text-sm font-medium text-slate-800 dark:text-neutral-200">
                  Distribution (text summary)
                </h3>
                <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-700">
                  <table className="min-w-full text-left text-xs">
                    <caption className="sr-only">Before and after score distribution buckets</caption>
                    <thead className="bg-slate-50 dark:bg-neutral-800/80">
                      <tr>
                        <th scope="col" className="px-2 py-1.5 font-medium">
                          Range
                        </th>
                        <th scope="col" className="px-2 py-1.5 font-medium">
                          Before
                        </th>
                        <th scope="col" className="px-2 py-1.5 font-medium">
                          After
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {preview.preview.histogramBefore.map((b, i) => (
                        <tr key={b.label} className="border-t border-slate-100 dark:border-neutral-800">
                          <td className="px-2 py-1.5">{b.label}</td>
                          <td className="px-2 py-1.5 tabular-nums">{b.count}</td>
                          <td className="px-2 py-1.5 tabular-nums">
                            {preview.preview.histogramAfter[i]?.count ?? 0}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              <div>
                <h3 className="mb-2 text-sm font-medium text-slate-800 dark:text-neutral-200">
                  Students with changes ({changedRows.length})
                </h3>
                {changedRows.length === 0 ? (
                  <p className="text-sm text-slate-600 dark:text-neutral-400">No scores would change.</p>
                ) : (
                  <div className="max-h-48 overflow-y-auto rounded-lg border border-slate-200 dark:border-neutral-700">
                    <table className="min-w-full text-left text-xs">
                      <caption className="sr-only">Per-student grade changes from curve preview</caption>
                      <thead className="sticky top-0 bg-slate-50 dark:bg-neutral-800/95">
                        <tr>
                          <th scope="col" className="px-2 py-1.5 font-medium">
                            Student
                          </th>
                          <th scope="col" className="px-2 py-1.5 font-medium">
                            Before
                          </th>
                          <th scope="col" className="px-2 py-1.5 font-medium">
                            After
                          </th>
                          <th scope="col" className="px-2 py-1.5 font-medium">
                            Δ
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {changedRows.map((row) => (
                          <tr key={row.studentId} className="border-t border-slate-100 dark:border-neutral-800">
                            <td className="px-2 py-1.5">{studentsById[row.studentId]?.name ?? row.studentId}</td>
                            <td className="px-2 py-1.5 tabular-nums">{row.rawScore}</td>
                            <td className="px-2 py-1.5 tabular-nums">{row.adjustedScore}</td>
                            <td className="px-2 py-1.5 tabular-nums">
                              {row.delta > 0 ? `+${row.delta}` : row.delta}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-4 py-3 dark:border-neutral-700">
          {activeCurveId ? (
            <button
              type="button"
              className="rounded-md border border-red-300 px-3 py-1.5 text-sm text-red-700 dark:border-red-800 dark:text-red-300"
              onClick={() => void handleRevert()}
              disabled={reverting || applying}
            >
              {reverting ? 'Reverting…' : 'Undo curve'}
            </button>
          ) : null}
          <button
            type="button"
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            type="button"
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
            onClick={() => void handleApply()}
            disabled={applying || reverting || loadingPreview || !preview}
          >
            {applying ? 'Applying…' : 'Apply curve'}
          </button>
        </div>
      </div>
    </div>
  )
}

function StatCard(props: { label: string; value?: number | null }) {
  const { label, value } = props
  return (
    <div className="rounded-lg border border-slate-200 px-3 py-2 dark:border-neutral-700">
      <div className="text-xs text-slate-500 dark:text-neutral-400">{label}</div>
      <div className="text-lg font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
        {value != null ? value.toFixed(2) : '—'}
      </div>
    </div>
  )
}
