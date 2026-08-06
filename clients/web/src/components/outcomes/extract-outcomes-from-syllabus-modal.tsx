import { useEffect, useId, useState } from 'react'
import { Loader2, Trash2, X } from 'lucide-react'
import {
  createCourseOutcome,
  type DraftCourseOutcome,
} from '../../lib/courses-api'

type DraftRow = DraftCourseOutcome & { key: string }

type ExtractOutcomesFromSyllabusModalProps = {
  open: boolean
  courseCode: string
  drafts: DraftCourseOutcome[]
  onClose: () => void
  onCreated: () => void | Promise<void>
}

function toRows(drafts: DraftCourseOutcome[]): DraftRow[] {
  return drafts.map((d, i) => ({
    key: `draft-${i}-${d.title.slice(0, 24)}`,
    title: d.title,
    description: d.description,
  }))
}

export function ExtractOutcomesFromSyllabusModal({
  open,
  courseCode,
  drafts,
  onClose,
  onCreated,
}: ExtractOutcomesFromSyllabusModalProps) {
  const titleId = useId()
  const [rows, setRows] = useState<DraftRow[]>(() => toRows(drafts))
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setRows(toRows(drafts))
    setError(null)
    setCreating(false)
  }, [open, drafts])

  if (!open) return null

  const creatable = rows.filter((r) => r.title.trim().length > 0)

  function updateRow(key: string, patch: Partial<DraftCourseOutcome>) {
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, ...patch } : r)))
  }

  function removeRow(key: string) {
    setRows((prev) => prev.filter((r) => r.key !== key))
  }

  async function onCreate() {
    if (creating || creatable.length === 0) return
    setCreating(true)
    setError(null)
    let created = 0
    try {
      for (const row of creatable) {
        await createCourseOutcome(courseCode, {
          title: row.title.trim(),
          description: row.description.trim(),
        })
        created += 1
      }
      await onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not create outcomes.'
      if (created > 0) {
        setError(
          `Created ${created} of ${creatable.length} outcomes, then failed: ${msg}`,
        )
        await onCreated()
      } else {
        setError(msg)
      }
    } finally {
      setCreating(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget && !creating) onClose()
      }}
    >
      <div className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-xl dark:border-border-default dark:bg-surface-raised">
        <div className="flex items-center justify-between border-b border-border-default px-4 py-3 dark:border-border-default">
          <div className="min-w-0">
            <h3
              id={titleId}
              className="text-sm font-semibold text-fg-default"
            >
              Review extracted outcomes
            </h3>
            <p className="mt-0.5 text-xs text-fg-muted">
              Edit or remove drafts, then create them in this course.
            </p>
          </div>
          <button
            type="button"
            onClick={() => {
              if (!creating) onClose()
            }}
            className="rounded-lg p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-default dark:hover:bg-surface-overlay dark:hover:text-fg-default"
            aria-label="Close"
            disabled={creating}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
          {rows.length === 0 ? (
            <p className="rounded-xl border border-dashed border-border-default px-4 py-6 text-center text-sm text-fg-muted dark:border-border-default dark:text-fg-muted">
              No outcomes to create. Close and try again, or add outcomes manually.
            </p>
          ) : (
            rows.map((row, index) => (
              <div
                key={row.key}
                className="rounded-xl border border-border-default bg-slate-50/60 p-3 dark:border-border-default/40"
              >
                <div className="mb-2 flex items-center justify-between gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
                    Outcome {index + 1}
                  </span>
                  <button
                    type="button"
                    onClick={() => removeRow(row.key)}
                    disabled={creating}
                    className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:text-rose-300 dark:hover:bg-rose-950/40"
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden />
                    Remove
                  </button>
                </div>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-fg-muted">
                    Title
                  </span>
                  <input
                    value={row.title}
                    onChange={(e) => updateRow(row.key, { title: e.target.value })}
                    disabled={creating}
                    className="w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                  />
                </label>
                <label className="mt-2 block">
                  <span className="mb-1 block text-xs font-medium text-fg-muted">
                    Description (optional)
                  </span>
                  <textarea
                    value={row.description}
                    onChange={(e) => updateRow(row.key, { description: e.target.value })}
                    disabled={creating}
                    rows={2}
                    className="w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                  />
                </label>
              </div>
            ))
          )}
        </div>

        {error ? (
          <p className="mx-4 mb-2 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
            {error}
          </p>
        ) : null}

        <div className="flex flex-wrap items-center justify-end gap-2 border-t border-border-default px-4 py-3 dark:border-border-default">
          <button
            type="button"
            onClick={onClose}
            disabled={creating}
            className="rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-sunken disabled:opacity-50 dark:text-fg-muted dark:hover:bg-surface-overlay"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void onCreate()}
            disabled={creating || creatable.length === 0}
            className="inline-flex items-center gap-2 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-surface-raised dark:shadow-none"
          >
            {creating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                Creating…
              </>
            ) : (
              `Create ${creatable.length} outcome${creatable.length === 1 ? '' : 's'}`
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
