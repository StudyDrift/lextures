import { useEffect, useId, useState } from 'react'
import { Loader2, Trash2, X } from 'lucide-react'
import {
  createBadgeDefinition,
  type DraftBadgeDefinition,
} from '../../lib/badges-api'
import type { CourseOutcome } from '../../lib/courses-api'

type DraftRow = {
  key: string
  name: string
  description: string
  outcomeId: string
}

type ExtractBadgesFromSyllabusModalProps = {
  open: boolean
  courseId: string
  drafts: DraftBadgeDefinition[]
  source: string
  outcomes: CourseOutcome[]
  onClose: () => void
  onCreated: () => void | Promise<void>
}

function toRows(drafts: DraftBadgeDefinition[]): DraftRow[] {
  return drafts.map((d, i) => ({
    key: `draft-${i}-${d.name.slice(0, 24)}`,
    name: d.name,
    description: d.description,
    outcomeId: d.outcomeId?.trim() || '',
  }))
}

export function ExtractBadgesFromSyllabusModal({
  open,
  courseId,
  drafts,
  source,
  outcomes,
  onClose,
  onCreated,
}: ExtractBadgesFromSyllabusModalProps) {
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

  const creatable = rows.filter((r) => r.name.trim().length > 0)
  const sourceLabel =
    source === 'outcomes'
      ? 'Drafted from learning outcomes (one badge per outcome).'
      : 'Drafted from the course syllabus (no outcomes on this course yet).'

  function updateRow(key: string, patch: Partial<Omit<DraftRow, 'key'>>) {
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
        await createBadgeDefinition(courseId, {
          name: row.name.trim(),
          description: row.description.trim(),
          outcomeId: row.outcomeId.trim() || undefined,
        })
        created += 1
      }
      await onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not create badges.'
      if (created > 0) {
        setError(`Created ${created} of ${creatable.length} badges, then failed: ${msg}`)
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
      <div className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-600">
          <div className="min-w-0">
            <h3
              id={titleId}
              className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
            >
              Review extracted badges
            </h3>
            <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{sourceLabel}</p>
          </div>
          <button
            type="button"
            onClick={() => {
              if (!creating) onClose()
            }}
            className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
            aria-label="Close"
            disabled={creating}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
          {rows.length === 0 ? (
            <p className="rounded-xl border border-dashed border-slate-200 px-4 py-6 text-center text-sm text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
              No badges to create. Close and try again, or add badges manually.
            </p>
          ) : (
            rows.map((row, index) => (
              <div
                key={row.key}
                className="rounded-xl border border-slate-200 bg-slate-50/60 p-3 dark:border-neutral-700 dark:bg-neutral-950/40"
              >
                <div className="mb-2 flex items-center justify-between gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    Badge {index + 1}
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
                  <span className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
                    Name
                  </span>
                  <input
                    value={row.name}
                    onChange={(e) => updateRow(row.key, { name: e.target.value })}
                    disabled={creating}
                    className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
                  />
                </label>
                <label className="mt-2 block">
                  <span className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
                    Description
                  </span>
                  <textarea
                    value={row.description}
                    onChange={(e) => updateRow(row.key, { description: e.target.value })}
                    disabled={creating}
                    rows={2}
                    className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
                  />
                </label>
                {outcomes.length > 0 ? (
                  <label className="mt-2 block">
                    <span className="mb-1 block text-xs font-medium text-slate-700 dark:text-neutral-300">
                      Linked outcome (optional)
                    </span>
                    <select
                      value={row.outcomeId}
                      onChange={(e) => updateRow(row.key, { outcomeId: e.target.value })}
                      disabled={creating}
                      className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
                    >
                      <option value="">None</option>
                      {outcomes.map((o) => (
                        <option key={o.id} value={o.id}>
                          {o.title}
                        </option>
                      ))}
                    </select>
                  </label>
                ) : null}
              </div>
            ))
          )}
        </div>

        {error ? (
          <p className="mx-4 mb-2 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
            {error}
          </p>
        ) : null}

        <div className="flex flex-wrap items-center justify-end gap-2 border-t border-slate-200 px-4 py-3 dark:border-neutral-600">
          <button
            type="button"
            onClick={onClose}
            disabled={creating}
            className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void onCreate()}
            disabled={creating || creatable.length === 0}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:shadow-none"
          >
            {creating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                Creating…
              </>
            ) : (
              `Create ${creatable.length} badge${creatable.length === 1 ? '' : 's'}`
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
