import { useEffect, useId, useMemo, useState } from 'react'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import type { OutcomeMappingProposal } from '../../../lib/course-checklist-api'
import {
  createOutcomeLink,
  suggestOutcomeLinks,
} from '../../../lib/course-checklist-api'
import { emitChecklistTelemetry } from '../../../lib/checklist-telemetry'

type Props = {
  courseCode: string
  itemId: string
  open: boolean
  onClose: () => void
  onApplied: () => void
  manualHref?: string | null
}

export function ChecklistMappingAssistDialog({
  courseCode,
  itemId,
  open,
  onClose,
  onApplied,
  manualHref,
}: Props) {
  const titleId = useId()
  const [loading, setLoading] = useState(false)
  const [applying, setApplying] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [proposals, setProposals] = useState<OutcomeMappingProposal[]>([])
  const [selected, setSelected] = useState<Record<string, boolean>>({})

  const keyOf = (p: OutcomeMappingProposal, i: number) =>
    `${p.structureItemId}|${p.outcomeId}|${p.measurementLevel}|${i}`

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setLoading(true)
    setError(null)
    setProposals([])
    setSelected({})
    emitChecklistTelemetry('checklist_assist_started', {
      itemId,
      actionKind: 'suggest_outcome_mappings',
    })
    void (async () => {
      try {
        const list = await suggestOutcomeLinks(courseCode)
        if (cancelled) return
        setProposals(list)
        const init: Record<string, boolean> = {}
        list.forEach((p, i) => {
          // No accept-all default (CC.10 risk mitigation) — start unselected.
          init[keyOf(p, i)] = false
        })
        setSelected(init)
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : courseChecklistI18n.assistFailed)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, courseCode, itemId])

  const selectedCount = useMemo(
    () => Object.values(selected).filter(Boolean).length,
    [selected],
  )

  if (!open) return null

  const apply = async () => {
    setApplying(true)
    setError(null)
    const accepted = proposals.filter((p, i) => selected[keyOf(p, i)])
    let written = 0
    try {
      for (const p of accepted) {
        await createOutcomeLink(courseCode, p.outcomeId, {
          structureItemId: p.structureItemId,
          measurementLevel: p.measurementLevel,
          intensityLevel: p.intensityLevel,
          targetKind: p.itemKind === 'quiz' ? 'quiz' : 'assignment',
        })
        written++
      }
      emitChecklistTelemetry('checklist_assist_accepted', {
        itemId,
        acceptedCount: written,
        proposedCount: proposals.length,
        actionKind: 'suggest_outcome_mappings',
      })
      onApplied()
      onClose()
    } catch (e) {
      setError(
        written > 0
          ? `${e instanceof Error ? e.message : courseChecklistI18n.assistFailed} (${written} applied)`
          : e instanceof Error
            ? e.message
            : courseChecklistI18n.assistFailed,
      )
    } finally {
      setApplying(false)
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      onClick={onClose}
    >
      <div
        className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-border-default bg-surface-raised shadow-xl dark:border-border-default dark:bg-surface-raised"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-border-default px-5 py-4 dark:border-border-default">
          <h2 id={titleId} className="text-base font-semibold text-fg-default">
            {courseChecklistI18n.assistReviewTitle}
          </h2>
          <p className="mt-1 text-xs text-fg-muted" aria-live="polite">
            {courseChecklistI18n.assistAiLabel} ·{' '}
            {courseChecklistI18n.assistSelectedCount(selectedCount, proposals.length)}
          </p>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {loading ? (
            <p className="text-sm text-fg-muted" aria-live="polite">
              {courseChecklistI18n.assistWorking}
            </p>
          ) : null}

          {error ? (
            <div className="mb-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200">
              <p>{error}</p>
              {manualHref ? (
                <a
                  href={manualHref}
                  className="mt-2 inline-flex min-h-11 items-center font-semibold underline-offset-2 hover:underline"
                >
                  {courseChecklistI18n.assistOpenManual}
                </a>
              ) : null}
            </div>
          ) : null}

          {!loading && !error && proposals.length === 0 ? (
            <p className="text-sm text-fg-muted">
              No proposals. {manualHref ? courseChecklistI18n.assistOpenManual : null}
            </p>
          ) : null}

          <ul className="space-y-2">
            {proposals.map((p, i) => {
              const k = keyOf(p, i)
              const checked = !!selected[k]
              return (
                <li
                  key={k}
                  className="rounded-lg border border-border-default p-3 dark:border-border-default"
                >
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0 flex-1 text-sm">
                      <p className="font-medium text-fg-default">
                        {p.itemTitle || p.structureItemId} → {p.outcomeTitle || p.outcomeId}
                      </p>
                      <p className="mt-0.5 text-xs text-fg-muted">
                        {p.measurementLevel} · {p.intensityLevel}
                        {typeof p.confidence === 'number'
                          ? ` · ${Math.round(p.confidence * 100)}%`
                          : ''}
                      </p>
                      {p.rationale ? (
                        <p className="mt-1 text-xs text-fg-muted">
                          {p.rationale}
                        </p>
                      ) : null}
                    </div>
                    <div className="flex shrink-0 gap-2">
                      <button
                        type="button"
                        aria-label={`${courseChecklistI18n.assistAccept}: ${p.itemTitle || p.structureItemId}`}
                        className={`inline-flex min-h-11 items-center rounded-lg px-3 text-xs font-semibold ${ checked ? 'bg-amber-700 text-white' : 'border border-border-strong text-fg-muted dark:border-border-default dark:text-fg-default' }`}
                        onClick={() => setSelected((s) => ({ ...s, [k]: true }))}
                      >
                        {courseChecklistI18n.assistAccept}
                      </button>
                      <button
                        type="button"
                        aria-label={`${courseChecklistI18n.assistReject}: ${p.itemTitle || p.structureItemId}`}
                        className={`inline-flex min-h-11 items-center rounded-lg px-3 text-xs font-semibold ${ !checked ? 'bg-slate-800 text-white dark:bg-neutral-200 dark:text-neutral-900' : 'border border-border-strong text-fg-muted dark:border-border-default dark:text-fg-default' }`}
                        onClick={() => setSelected((s) => ({ ...s, [k]: false }))}
                      >
                        {courseChecklistI18n.assistReject}
                      </button>
                    </div>
                  </div>
                </li>
              )
            })}
          </ul>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border-default px-5 py-3 dark:border-border-default">
          <div className="flex gap-2">
            <button
              type="button"
              className="inline-flex min-h-11 items-center rounded-lg border border-border-strong px-3 text-xs font-semibold dark:border-border-default"
              onClick={() => {
                const next: Record<string, boolean> = {}
                proposals.forEach((p, i) => {
                  next[keyOf(p, i)] = true
                })
                setSelected(next)
              }}
            >
              {courseChecklistI18n.assistAcceptAll}
            </button>
            <button
              type="button"
              className="inline-flex min-h-11 items-center rounded-lg border border-border-strong px-3 text-xs font-semibold dark:border-border-default"
              onClick={() => {
                const next: Record<string, boolean> = {}
                proposals.forEach((p, i) => {
                  next[keyOf(p, i)] = false
                })
                setSelected(next)
              }}
            >
              {courseChecklistI18n.assistRejectAll}
            </button>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              className="inline-flex min-h-11 items-center rounded-lg px-3 text-sm font-semibold text-fg-default"
              onClick={onClose}
              disabled={applying}
            >
              {courseChecklistI18n.assistCancel}
            </button>
            <button
              type="button"
              disabled={applying || selectedCount === 0}
              className="inline-flex min-h-11 items-center rounded-lg bg-amber-700 px-4 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-60"
              onClick={() => void apply()}
            >
              {applying ? courseChecklistI18n.assistWorking : courseChecklistI18n.assistConfirm}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
